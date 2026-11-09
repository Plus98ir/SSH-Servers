//go:build windows

package main

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// ---------------------------------------------------------------- paths & helpers

func sshDir() string { return filepath.Join(os.Getenv("USERPROFILE"), ".ssh") }

func knownHostsPath() string { return filepath.Join(sshDir(), "known_hosts") }

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

func addrOf(s server) string {
	p := s.Port
	if p == 0 {
		p = 22
	}
	return net.JoinHostPort(s.Host, strconv.Itoa(p))
}

func userOf(s server) string {
	if s.User == "" {
		return "root"
	}
	return s.User
}

func keygenPath() string {
	if p, err := exec.LookPath("ssh-keygen.exe"); err == nil {
		return p
	}
	return filepath.Join(os.Getenv("SystemRoot"), "System32", "OpenSSH", "ssh-keygen.exe")
}

func hiddenRun(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	return c.Run()
}

// ---------------------------------------------------------------- known_hosts

var khMu sync.Mutex // serializes reads/writes of known_hosts

func ensureKnownHosts() string {
	kh := knownHostsPath()
	if !fileExists(kh) {
		os.MkdirAll(sshDir(), 0o700)
		os.WriteFile(kh, nil, 0o644)
	}
	return kh
}

// hostPattern is how OpenSSH writes a host in known_hosts: "host" for port 22, "[host]:port" otherwise.
func hostPattern(s server) string { return knownhosts.Normalize(addrOf(s)) }

func matchesHost(field, pattern string) bool {
	if strings.HasPrefix(field, "|1|") { // hashed: |1|salt|hash
		parts := strings.Split(field, "|")
		if len(parts) != 4 {
			return false
		}
		salt, e1 := base64.StdEncoding.DecodeString(parts[2])
		want, e2 := base64.StdEncoding.DecodeString(parts[3])
		if e1 != nil || e2 != nil {
			return false
		}
		m := hmac.New(sha1.New, salt)
		m.Write([]byte(pattern))
		return hmac.Equal(m.Sum(nil), want)
	}
	for _, h := range strings.Split(field, ",") {
		if h == pattern {
			return true
		}
	}
	return false
}

// replaceHostKey removes every known_hosts entry for this server and saves the new key.
// (Same as `ssh-keygen -R` + accepting the new key, done for the user.)
func replaceHostKey(s server, key ssh.PublicKey) error {
	khMu.Lock()
	defer khMu.Unlock()
	kh := ensureKnownHosts()
	pat := hostPattern(s)
	b, err := os.ReadFile(kh)
	if err != nil {
		return err
	}
	var out []string
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		f := strings.Fields(line)
		if len(f) >= 3 && !strings.HasPrefix(f[0], "#") && !strings.HasPrefix(f[0], "@") {
			if strings.HasPrefix(f[0], "|1|") {
				if matchesHost(f[0], pat) {
					continue
				}
			} else {
				hosts := strings.Split(f[0], ",")
				keep := hosts[:0]
				for _, h := range hosts {
					if h != pat {
						keep = append(keep, h)
					}
				}
				if len(keep) == 0 {
					continue
				}
				if len(keep) != len(strings.Split(f[0], ",")) {
					line = strings.Join(keep, ",") + line[len(f[0]):]
				}
			}
		}
		out = append(out, line)
	}
	if key != nil {
		out = append(out, knownhosts.Line([]string{pat}, key))
	}
	data := strings.Join(out, "\n") + "\n"
	if strings.TrimSpace(data) == "" {
		data = ""
	}
	os.WriteFile(kh+".old", b, 0o644) // one-step backup, like ssh-keygen -R
	tmp := kh + ".tmp"
	if err := os.WriteFile(tmp, []byte(data), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, kh)
}

func fingerprint(k ssh.PublicKey) string {
	if k == nil {
		return ""
	}
	return k.Type() + "  " + ssh.FingerprintSHA256(k)
}

// ---------------------------------------------------------------- dialing

// hostKeyChanged carries the server's new key when it no longer matches known_hosts.
type hostKeyChanged struct{ Key ssh.PublicKey }

func (e *hostKeyChanged) Error() string { return "hostkey-changed" }

var errHostKeyChangedMsg = func() string {
	return tr("کلید هویت این سرور عوض شده (احتمالاً Rebuild شده)", "This server's host key changed (probably rebuilt)")
}

func algosFor(keyType string) []string {
	switch keyType {
	case ssh.KeyAlgoRSA:
		return []string{ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSA}
	default:
		return []string{keyType}
	}
}

func baseConfig(s server, auth []ssh.AuthMethod) *ssh.ClientConfig {
	sup, ins := ssh.SupportedAlgorithms(), ssh.InsecureAlgorithms()
	c := &ssh.ClientConfig{
		User:    userOf(s),
		Auth:    auth,
		Timeout: 10 * time.Second,
	}
	// also allow the older algorithms many routers/modems (Dropbear, old OpenSSH) still use
	c.KeyExchanges = append(sup.KeyExchanges, ins.KeyExchanges...)
	c.Ciphers = append(sup.Ciphers, ins.Ciphers...)
	c.MACs = append(sup.MACs, ins.MACs...)
	c.HostKeyAlgorithms = append(sup.HostKeys, ins.HostKeys...)
	return c
}

// dialChecked connects and checks the host key like OpenSSH does:
//   - unknown host: accepted (OpenSSH "accept-new" saves it on the first terminal login)
//   - known and matching: ok
//   - known but different: returns *hostKeyChanged with the new key
//
// If known_hosts only has other key types for this host, it asks the server for those
// types first (like OpenSSH), so a server that merely gained a new key type isn't flagged.
func dialChecked(s server, auth []ssh.AuthMethod) (*ssh.Client, error) {
	khMu.Lock()
	cb, err := knownhosts.New(ensureKnownHosts())
	khMu.Unlock()
	if err != nil {
		return nil, err
	}
	var presented ssh.PublicKey
	var wantTypes []string
	mkCallback := func() ssh.HostKeyCallback {
		return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			presented = key
			err := cb(hostname, remote, key)
			var ke *knownhosts.KeyError
			if errors.As(err, &ke) {
				if len(ke.Want) == 0 {
					return nil // first time we see this host
				}
				wantTypes = wantTypes[:0]
				for _, k := range ke.Want {
					if k.Key.Type() == key.Type() {
						return &hostKeyChanged{Key: key}
					}
					wantTypes = append(wantTypes, k.Key.Type())
				}
				return errTypeMismatch
			}
			return err
		}
	}

	conf := baseConfig(s, auth)
	conf.HostKeyCallback = mkCallback()
	c, err := ssh.Dial("tcp", addrOf(s), conf)
	if err == nil || !errors.Is(err, errTypeMismatch) {
		return c, unwrapChanged(err)
	}

	// retry asking only for the key types we already know
	first := presented
	var algs []string
	seen := map[string]bool{}
	for _, t := range wantTypes {
		for _, a := range algosFor(t) {
			if !seen[a] {
				seen[a] = true
				algs = append(algs, a)
			}
		}
	}
	conf = baseConfig(s, auth)
	conf.HostKeyAlgorithms = algs
	conf.HostKeyCallback = mkCallback()
	c, err = ssh.Dial("tcp", addrOf(s), conf)
	if err != nil && !errors.Is(err, errTypeMismatch) {
		var hc *hostKeyChanged
		if errors.As(err, &hc) {
			return nil, hc
		}
		if strings.Contains(err.Error(), "no common algorithm") {
			// server no longer has any key type we saved: it was rebuilt
			return nil, &hostKeyChanged{Key: first}
		}
		return nil, err
	}
	return c, unwrapChanged(err)
}

var errTypeMismatch = errors.New("hostkey-type-mismatch")

func unwrapChanged(err error) error {
	var hc *hostKeyChanged
	if errors.As(err, &hc) {
		return hc
	}
	return err
}

// signers loads the default private keys the ssh client would also try.
func signers() []ssh.Signer {
	var out []ssh.Signer
	for _, n := range []string{"id_ed25519", "id_ecdsa", "id_rsa"} {
		if b, err := os.ReadFile(filepath.Join(sshDir(), n)); err == nil {
			if s, err := ssh.ParsePrivateKey(b); err == nil {
				out = append(out, s)
			}
		}
	}
	return out
}

func keyAuth() []ssh.AuthMethod {
	if sg := signers(); len(sg) > 0 {
		return []ssh.AuthMethod{ssh.PublicKeys(sg...)}
	}
	return nil
}

// probe checks a server before opening a terminal: host key state and whether key login works.
type probeResult struct {
	Changed *hostKeyChanged
	AuthOK  bool
	NetErr  error // couldn't reach / talk to it; let the terminal show what's wrong
}

func probe(s server) probeResult {
	c, err := dialChecked(s, keyAuth())
	if err == nil {
		c.Close()
		return probeResult{AuthOK: true}
	}
	var hc *hostKeyChanged
	if errors.As(err, &hc) {
		return probeResult{Changed: hc}
	}
	if strings.Contains(err.Error(), "unable to authenticate") || strings.Contains(err.Error(), "no supported methods") {
		return probeResult{}
	}
	return probeResult{NetErr: err}
}

// ---------------------------------------------------------------- key setup

// ensureKey makes sure ~/.ssh/id_<kind>(.pub) exists. kind: "ed25519" or "rsa".
func ensureKey(kind string) (privPath, pubLine string, err error) {
	dir := sshDir()
	os.MkdirAll(dir, 0o700)
	privPath = filepath.Join(dir, "id_"+kind)
	pubPath := privPath + ".pub"

	if !fileExists(privPath) {
		comment := os.Getenv("USERNAME") + "@" + os.Getenv("COMPUTERNAME")
		args := []string{"-q", "-t", kind, "-N", "", "-C", comment, "-f", privPath}
		if kind == "rsa" {
			args = append([]string{"-b", "3072"}, args...)
		}
		if err = hiddenRun(keygenPath(), args...); err != nil || !fileExists(privPath) {
			return "", "", errors.New(tr("ساختن کلید SSH نشد (ssh-keygen)", "Couldn't create the SSH key (ssh-keygen)"))
		}
	}
	if !fileExists(pubPath) {
		b, e := os.ReadFile(privPath)
		if e != nil {
			return "", "", e
		}
		signer, e := ssh.ParsePrivateKey(b)
		if e != nil {
			return "", "", errors.New(tr("فایل کلید عمومی نیست و کلید خصوصی رمز داره؛ نمی‌تونم کلید عمومی رو بسازم", "The public key file is missing and the private key has a passphrase, so it can't be rebuilt"))
		}
		if e = os.WriteFile(pubPath, ssh.MarshalAuthorizedKey(signer.PublicKey()), 0o644); e != nil {
			return "", "", e
		}
	}
	b, err := os.ReadFile(pubPath)
	if err != nil {
		return "", "", err
	}
	pubLine = strings.TrimSpace(string(b))
	if !strings.HasPrefix(pubLine, "ssh-") || strings.ContainsAny(pubLine, "'\n") {
		return "", "", errors.New(tr("فایل کلید عمومی معتبر نیست", "The public key file is not valid"))
	}
	return privPath, pubLine, nil
}

// run executes cmd on the client with a timeout; returns output and exit code.
func run(c *ssh.Client, cmd string, timeout time.Duration) (string, int, error) {
	sess, err := c.NewSession()
	if err != nil {
		return "", -1, err
	}
	defer sess.Close()
	type res struct {
		out []byte
		err error
	}
	ch := make(chan res, 1)
	go func() { o, e := sess.CombinedOutput(cmd); ch <- res{o, e} }()
	select {
	case r := <-ch:
		code := 0
		if r.err != nil {
			var ee *ssh.ExitError
			if errors.As(r.err, &ee) {
				code = ee.ExitStatus()
			} else if strings.Contains(r.err.Error(), "exited without exit status") {
				code = 0
			} else {
				return string(r.out), -1, r.err
			}
		}
		return string(r.out), code, nil
	case <-time.After(timeout):
		c.Close()
		return "", -1, errors.New(tr("زمان اجرا تموم شد (بیشتر از ۹۰ ثانیه)", "Timed out (over 90 seconds)"))
	}
}

func installCmd(pubLine string, dropbear bool) string {
	add := func(dir string) string {
		f := dir + "/authorized_keys"
		return "mkdir -p " + dir + " && touch " + f + " && chmod 700 " + dir + " && chmod 600 " + f +
			" && (grep -qxF '" + pubLine + "' " + f + " || echo '" + pubLine + "' >> " + f + ")"
	}
	cmd := "umask 077; " + add("$HOME/.ssh")
	if dropbear {
		// OpenWrt & most routers: Dropbear reads root's keys from /etc/dropbear
		cmd = "umask 077; if [ -d /etc/dropbear ]; then " + add("/etc/dropbear") + "; fi; " + add("$HOME/.ssh") + " 2>/dev/null; true"
	}
	return cmd
}

var errVerify = errors.New("verify")

func friendlyDialErr(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "unable to authenticate"):
		return errors.New(tr("رمز یا یوزرنیم اشتباهه", "Wrong password or username"))
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "refused"), strings.Contains(msg, "no route"), strings.Contains(msg, "unreachable"):
		return errors.New(tr("به سرور وصل نشد", "Couldn't reach the server") + " (" + msg + ")")
	}
	return errors.New(tr("اتصال نشد: ", "Connection failed: ") + msg)
}

// installKey logs in once with the password, adds our public key, then verifies a key-only login.
// Returns legacy=true when the device needs the old ssh-rsa options for the ssh client.
// A changed host key comes back as *hostKeyChanged.
func installKey(s server, password string) (legacy bool, err error) {
	auth := []ssh.AuthMethod{
		ssh.Password(password),
		ssh.KeyboardInteractive(func(_, _ string, qs []string, _ []bool) ([]string, error) {
			ans := make([]string, len(qs))
			for i := range ans {
				ans[i] = password
			}
			return ans, nil
		}),
	}
	client, err := dialChecked(s, auth)
	if err != nil {
		var hc *hostKeyChanged
		if errors.As(err, &hc) {
			return false, hc
		}
		return false, friendlyDialErr(err)
	}
	defer client.Close()
	dropbear := strings.Contains(strings.ToLower(string(client.ServerVersion())), "dropbear")

	tryKind := func(kind string) error {
		privPath, pubLine, err := ensureKey(kind)
		if err != nil {
			return err
		}
		out, code, err := run(client, installCmd(pubLine, dropbear), 30*time.Second)
		if err != nil || code != 0 {
			o := strings.TrimSpace(out)
			lo := strings.ToLower(o)
			if strings.Contains(o, "not found") || strings.Contains(lo, "unknown command") || strings.Contains(lo, "invalid") {
				return errors.New(tr("این دستگاه شِل لینوکسی نداره (فقط منوی خودش رو داره)، برای همین نمی‌شه کلید رو خودکار روش گذاشت. خروجی: ", "This device has no Linux shell (only its own menu), so the key can't be installed automatically. Output: ") + o)
			}
			if o == "" && err != nil {
				o = err.Error()
			}
			return errors.New(tr("نصب کلید روی سرور نشد: ", "Couldn't install the key on the server: ") + o)
		}
		b, e := os.ReadFile(privPath)
		if e != nil {
			return nil
		}
		signer, e := ssh.ParsePrivateKey(b)
		if e != nil {
			return nil // passphrase-protected: can't verify, assume ok
		}
		c2, e := dialChecked(s, []ssh.AuthMethod{ssh.PublicKeys(signer)})
		if e != nil {
			return errVerify
		}
		c2.Close()
		return nil
	}

	err = tryKind("ed25519")
	if err == errVerify {
		// old Dropbear / old firmware: no ed25519 support -> fall back to RSA
		if err = tryKind("rsa"); err == nil {
			return true, nil
		}
	}
	if err == errVerify {
		return false, errors.New(tr("کلید روی دستگاه نصب شد ولی ورود با کلید هنوز کار نمی‌کنه؛ احتمالاً این دستگاه ورود با کلید رو پشتیبانی نمی‌کنه یا بسته", "The key was installed but key login still fails; this device probably doesn't support or has disabled key login"))
	}
	return dropbear, err
}

// ---------------------------------------------------------------- ping

func pingOne(s server) int {
	t0 := time.Now()
	c, err := net.DialTimeout("tcp", addrOf(s), 3*time.Second)
	if err != nil {
		return -1
	}
	c.Close()
	ms := int(time.Since(t0).Milliseconds())
	if ms < 1 {
		ms = 1
	}
	return ms
}
