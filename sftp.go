//go:build windows

package main

// Built-in SFTP file manager: one SFTP session per server, file operations, uploads streamed
// from the page in chunks, downloads written straight to the Downloads folder.

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// ---------------------------------------------------------------- session

type fmSess struct {
	mu   sync.Mutex
	srv  server
	pass string // kept in memory only while the window is open (for reconnects)
	sc   *ssh.Client
	c    *sftp.Client
	home string
	gen  int
}

var (
	fmMu       sync.Mutex
	fmSessions = map[string]*fmSess{}
)

func passAuth(pw string) []ssh.AuthMethod {
	return []ssh.AuthMethod{
		ssh.Password(pw),
		ssh.KeyboardInteractive(func(_, _ string, qs []string, _ []bool) ([]string, error) {
			ans := make([]string, len(qs))
			for i := range ans {
				ans[i] = pw
			}
			return ans, nil
		}),
	}
}

// dial opens ssh + sftp. Call with f.mu held.
func (f *fmSess) dial() error {
	auth := keyAuth()
	if f.pass != "" {
		auth = append(auth, passAuth(f.pass)...)
	}
	if len(auth) == 0 {
		return errNeedPass
	}
	sc, err := dialChecked(f.srv, auth)
	if err != nil {
		return err
	}
	c, err := sftp.NewClient(sc, sftp.UseConcurrentWrites(true), sftp.UseConcurrentReads(true), sftp.MaxConcurrentRequestsPerFile(64))
	if err != nil {
		sc.Close()
		return errNoSFTP
	}
	f.sc, f.c = sc, c
	f.gen++
	if f.home == "" {
		if h, err := c.Getwd(); err == nil && h != "" {
			f.home = h
		} else {
			f.home = "/"
		}
	}
	go f.keepAlive(sc, f.gen)
	return nil
}

func (f *fmSess) keepAlive(sc *ssh.Client, gen int) {
	t := time.NewTicker(25 * time.Second)
	defer t.Stop()
	for range t.C {
		f.mu.Lock()
		cur := f.gen == gen && f.sc == sc
		f.mu.Unlock()
		if !cur {
			return
		}
		if _, _, err := sc.SendRequest("keepalive@openssh.com", true, nil); err != nil {
			f.mu.Lock()
			if f.sc == sc {
				f.drop()
			}
			f.mu.Unlock()
			return
		}
	}
}

// drop closes the connection. Call with f.mu held.
func (f *fmSess) drop() {
	if f.c != nil {
		f.c.Close()
	}
	if f.sc != nil {
		f.sc.Close()
	}
	f.c, f.sc = nil, nil
}

func (f *fmSess) client() (*sftp.Client, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.c == nil {
		if err := f.dial(); err != nil {
			return nil, err
		}
	}
	return f.c, nil
}

func connLost(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, sftp.ErrSSHFxConnectionLost) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	m := err.Error()
	return strings.Contains(m, "connection lost") || strings.Contains(m, "closed network connection") ||
		strings.Contains(m, "connection reset") || strings.Contains(m, "broken pipe") || strings.Contains(m, "forcibly closed")
}

// do runs fn, reconnecting once if the connection dropped (laptop sleep, NAT timeout...).
func (f *fmSess) do(fn func(c *sftp.Client) error) error {
	c, err := f.client()
	if err != nil {
		return err
	}
	err = fn(c)
	if connLost(err) {
		f.mu.Lock()
		if f.c == c {
			f.drop()
		}
		f.mu.Unlock()
		if c, err = f.client(); err == nil {
			err = fn(c)
		}
	}
	return err
}

func getSess(id string) *fmSess {
	fmMu.Lock()
	defer fmMu.Unlock()
	return fmSessions[id]
}

var (
	errNeedPass = errors.New("need-pass")
	errNoSFTP   = errors.New("no-sftp")
	errCancel   = errors.New("cancel")
)

// fsErr turns sftp/ssh errors into short messages for the user.
func fsErr(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, errCancel) {
		return "cancel"
	}
	var se *sftp.StatusError
	if errors.As(err, &se) {
		switch se.FxCode() {
		case sftp.ErrSSHFxPermissionDenied:
			return tr("دسترسی نداری (Permission denied)", "Permission denied")
		case sftp.ErrSSHFxNoSuchFile:
			return tr("همچین فایل یا پوشه‌ای نیست", "No such file or folder")
		case sftp.ErrSSHFxFailure:
			return tr("سرور قبول نکرد (شاید از قبل وجود داره یا پوشه خالی نیست)", "The server refused (maybe it already exists or the folder isn't empty)")
		}
	}
	if os.IsNotExist(err) {
		return tr("همچین فایل یا پوشه‌ای نیست", "No such file or folder")
	}
	if os.IsPermission(err) {
		return tr("دسترسی نداری (Permission denied)", "Permission denied")
	}
	if connLost(err) {
		return tr("اتصال قطع شد", "Connection lost")
	}
	return err.Error()
}

// ---------------------------------------------------------------- listing

type fsItem struct {
	N string `json:"n"`           // name
	D bool   `json:"d"`           // directory (or link to one)
	L bool   `json:"l,omitempty"` // symlink
	S int64  `json:"s"`           // size
	T int64  `json:"t"`           // modified (ms)
	M string `json:"m"`           // mode string
	O uint32 `json:"o"`           // permission bits
}

func remoteJoin(dir, name string) string { return path.Join(dir, name) }

func (f *fmSess) resolve(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "~" {
		return f.home
	}
	if strings.HasPrefix(p, "~/") {
		return path.Join(f.home, p[2:])
	}
	if !strings.HasPrefix(p, "/") {
		return path.Join(f.home, p)
	}
	return path.Clean(p)
}

func (f *fmSess) list(p string) (map[string]interface{}, error) {
	dir := f.resolve(p)
	var items []fsItem
	err := f.do(func(c *sftp.Client) error {
		infos, err := c.ReadDir(dir)
		if err != nil {
			return err
		}
		items = make([]fsItem, 0, len(infos))
		for _, fi := range infos {
			n := fi.Name()
			if n == "." || n == ".." {
				continue
			}
			it := fsItem{N: n, D: fi.IsDir(), S: fi.Size(), T: fi.ModTime().UnixMilli(), M: fi.Mode().String(), O: uint32(fi.Mode().Perm())}
			if fi.Mode()&os.ModeSymlink != 0 {
				it.L = true
				if t, err := c.Stat(remoteJoin(dir, n)); err == nil {
					it.D = t.IsDir()
					if !it.D {
						it.S = t.Size()
					}
				}
			}
			items = append(items, it)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return strings.ToLower(items[i].N) < strings.ToLower(items[j].N) })
	return map[string]interface{}{"path": dir, "home": f.home, "items": items}, nil
}

func removeAny(c *sftp.Client, p string) error {
	fi, err := c.Lstat(p)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return c.RemoveAll(p)
	}
	return c.Remove(p)
}

const maxEdit = 5 << 20

// fsOp runs one file operation; the answer goes back through onFs.
func (f *fmSess) fsOp(op string, a []string) (interface{}, error) {
	arg := func(i int) string {
		if i < len(a) {
			return a[i]
		}
		return ""
	}
	switch op {
	case "list":
		return f.list(arg(0))
	case "mkdir":
		return nil, f.do(func(c *sftp.Client) error { return c.Mkdir(f.resolve(arg(0))) })
	case "touch":
		return nil, f.do(func(c *sftp.Client) error {
			fl, err := c.OpenFile(f.resolve(arg(0)), os.O_WRONLY|os.O_CREATE|os.O_EXCL)
			if err != nil {
				return err
			}
			return fl.Close()
		})
	case "rename":
		return nil, f.do(func(c *sftp.Client) error { return c.Rename(f.resolve(arg(0)), f.resolve(arg(1))) })
	case "remove":
		var errs []string
		for _, p := range a {
			if err := f.do(func(c *sftp.Client) error { return removeAny(c, f.resolve(p)) }); err != nil {
				errs = append(errs, path.Base(p)+": "+fsErr(err))
			}
		}
		if len(errs) > 0 {
			return nil, errors.New(strings.Join(errs, "\n"))
		}
		return nil, nil
	case "chmod":
		m, err := strconv.ParseUint(arg(1), 8, 32)
		if err != nil || m > 0o7777 {
			return nil, errors.New(tr("دسترسی رو درست بنویس (مثلاً 644)", "Enter valid permissions (e.g. 644)"))
		}
		var errs []string
		if len(a) < 3 {
			return nil, nil
		}
		for _, p := range a[2:] {
			if err := f.do(func(c *sftp.Client) error { return c.Chmod(f.resolve(p), os.FileMode(m)) }); err != nil {
				errs = append(errs, path.Base(p)+": "+fsErr(err))
			}
		}
		if len(errs) > 0 {
			return nil, errors.New(strings.Join(errs, "\n"))
		}
		return nil, nil
	case "read":
		p := f.resolve(arg(0))
		var out map[string]interface{}
		err := f.do(func(c *sftp.Client) error {
			fi, err := c.Stat(p)
			if err != nil {
				return err
			}
			if fi.IsDir() {
				return errors.New("dir")
			}
			if fi.Size() > maxEdit {
				return errors.New("big")
			}
			fl, err := c.Open(p)
			if err != nil {
				return err
			}
			defer fl.Close()
			b, err := io.ReadAll(io.LimitReader(fl, maxEdit+1))
			if err != nil {
				return err
			}
			head := b
			if len(head) > 8192 {
				head = head[:8192]
			}
			if strings.IndexByte(string(head), 0) >= 0 || !utf8.Valid(b) {
				return errors.New("binary")
			}
			s := string(b)
			out = map[string]interface{}{"text": strings.ReplaceAll(s, "\r\n", "\n"), "crlf": strings.Contains(s, "\r\n"), "path": p}
			return nil
		})
		return out, err
	case "write":
		p := f.resolve(arg(0))
		text := strings.ReplaceAll(arg(1), "\r\n", "\n")
		if arg(2) == "1" {
			text = strings.ReplaceAll(text, "\n", "\r\n")
		}
		return nil, f.do(func(c *sftp.Client) error {
			fl, err := c.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
			if err != nil {
				return err
			}
			if _, err = fl.Write([]byte(text)); err != nil {
				fl.Close()
				return err
			}
			return fl.Close()
		})
	}
	return nil, errors.New("unknown op " + op)
}

// ---------------------------------------------------------------- transfers

type xfer struct {
	id     string
	sid    string
	kind   string // up | down
	name   string
	total  int64
	done   atomic.Int64
	cancel atomic.Bool
	ch     chan []byte // uploads: chunks from the page
	abort  atomic.Bool
	last   time.Time
	local  string
}

var (
	xfMu  sync.Mutex
	xfers = map[string]*xfer{}
	xfSeq atomic.Int64
)

func (x *xfer) report(state, msg string, force bool) {
	if !force && time.Since(x.last) < 150*time.Millisecond {
		return
	}
	x.last = time.Now()
	js("onXfer", map[string]interface{}{"x": x.id, "sid": x.sid, "k": x.kind, "n": x.name, "done": x.done.Load(), "total": x.total, "st": state, "e": msg, "p": x.local})
}

func (x *xfer) finish(err error) {
	xfMu.Lock()
	delete(xfers, x.id)
	xfMu.Unlock()
	switch {
	case err == nil:
		x.report("ok", "", true)
	case errors.Is(err, errCancel) || x.cancel.Load():
		x.report("cancel", "", true)
	default:
		x.report("err", fsErr(err), true)
	}
}

type countW struct {
	w io.Writer
	x *xfer
}

func (c countW) Write(b []byte) (int, error) {
	if c.x.cancel.Load() {
		return 0, errCancel
	}
	n, err := c.w.Write(b)
	c.x.done.Add(int64(n))
	c.x.report("run", "", false)
	return n, err
}

func downloadsDir() string {
	d := filepath.Join(os.Getenv("USERPROFILE"), "Downloads")
	if fi, err := os.Stat(d); err == nil && fi.IsDir() {
		return d
	}
	return os.Getenv("USERPROFILE")
}

var winReserved = map[string]bool{"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true}

// winName makes a remote name safe as a Windows file name.
func winName(n string) string {
	n = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, n)
	n = strings.TrimRight(n, ". ")
	if n == "" {
		n = "_"
	}
	base := strings.ToUpper(strings.SplitN(n, ".", 2)[0])
	if winReserved[base] {
		n = "_" + n
	}
	return n
}

func uniqueLocal(dir, name string) string {
	p := filepath.Join(dir, name)
	if _, err := os.Lstat(p); err != nil {
		return p
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 1; ; i++ {
		p = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", stem, i, ext))
		if _, err := os.Lstat(p); err != nil {
			return p
		}
	}
}

func downloadFile(c *sftp.Client, remote, local string, x *xfer) error {
	rf, err := c.Open(remote)
	if err != nil {
		return err
	}
	defer rf.Close()
	lf, err := os.Create(local)
	if err != nil {
		return err
	}
	_, err = rf.WriteTo(countW{lf, x})
	if cerr := lf.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		if fi, e := rf.Stat(); e == nil {
			os.Chtimes(local, time.Now(), fi.ModTime())
		}
	}
	return err
}

// download copies remote files/folders to Downloads, one transfer row per item.
func (f *fmSess) download(paths []string) {
	dst := downloadsDir()
	for _, p := range paths {
		p = f.resolve(p)
		x := &xfer{id: "d" + strconv.FormatInt(xfSeq.Add(1), 10), sid: f.srv.ID, kind: "down", name: path.Base(p)}
		xfMu.Lock()
		xfers[x.id] = x
		xfMu.Unlock()
		x.report("run", "", true)
		var local string
		err := f.do(func(c *sftp.Client) error {
			x.done.Store(0)
			fi, err := c.Stat(p)
			if err != nil {
				return err
			}
			if !fi.IsDir() {
				x.total = fi.Size()
				local = uniqueLocal(dst, winName(path.Base(p)))
				x.local = local
				err = downloadFile(c, p, local, x)
				if err != nil {
					os.Remove(local)
				}
				return err
			}
			// folder: size first, then copy the tree
			type ent struct {
				rel string
				dir bool
			}
			var list []ent
			var total int64
			w := c.Walk(p)
			for w.Step() {
				if w.Err() != nil {
					continue
				}
				rel := strings.TrimPrefix(strings.TrimPrefix(w.Path(), p), "/")
				st := w.Stat()
				if st.Mode()&os.ModeSymlink != 0 {
					continue
				}
				if rel == "" {
					continue
				}
				list = append(list, ent{rel, st.IsDir()})
				if !st.IsDir() {
					total += st.Size()
				}
			}
			x.total = total
			local = uniqueLocal(dst, winName(path.Base(p)))
			x.local = local
			if err := os.MkdirAll(local, 0o755); err != nil {
				return err
			}
			for _, e := range list {
				if x.cancel.Load() {
					return errCancel
				}
				parts := strings.Split(e.rel, "/")
				for i := range parts {
					parts[i] = winName(parts[i])
				}
				lp := filepath.Join(append([]string{local}, parts...)...)
				if e.dir {
					os.MkdirAll(lp, 0o755)
					continue
				}
				os.MkdirAll(filepath.Dir(lp), 0o755)
				if err := downloadFile(c, p+"/"+e.rel, lp, x); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil && x.cancel.Load() && local != "" {
			os.RemoveAll(local)
		}
		x.finish(err)
	}
}

// upload receives chunks from the page and writes them to the server.
func (f *fmSess) upload(x *xfer, remote string, ch chan []byte) {
	var rf *sftp.File
	var werr error
	c, err := f.client()
	if err == nil {
		if d := path.Dir(remote); d != "/" && d != "." {
			c.MkdirAll(d)
		}
		rf, err = c.OpenFile(remote, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	}
	werr = err
	for b := range ch {
		if werr == nil && x.cancel.Load() {
			werr = errCancel
		}
		if werr == nil {
			_, werr = rf.Write(b)
			x.done.Add(int64(len(b)))
			x.report("run", "", false)
		}
		js("onUpAck", x.id, fsErr(werr))
	}
	if rf != nil {
		if cerr := rf.Close(); werr == nil {
			werr = cerr
		}
	}
	if werr == nil && x.abort.Load() {
		werr = errCancel
	}
	if werr != nil && rf != nil && c != nil {
		c.Remove(remote)
	}
	x.finish(werr)
}

// ---------------------------------------------------------------- opening local files

var (
	shell32       = syscall.NewLazyDLL("shell32.dll")
	pShellExecute = shell32.NewProc("ShellExecuteW")
)

func shellOpen(p string) {
	verb, _ := syscall.UTF16PtrFromString("open")
	file, _ := syscall.UTF16PtrFromString(p)
	pShellExecute.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)), 0, 0, 1)
}

func showInFolder(p string) {
	c := exec.Command("explorer.exe")
	c.SysProcAttr = &syscall.SysProcAttr{CmdLine: `explorer.exe /select,"` + p + `"`}
	c.Start()
}

// ---------------------------------------------------------------- bindings

func bindFiles() {
	// fsOpen connects the file manager to a server. pass is optional (used when key login doesn't work).
	// Answers onFsOpen(id, info, err) with err = "" | "pass" | "nosftp" | message.
	w.Bind("fsOpen", func(id, pass string, saveKey bool) error {
		s, ok := get(id)
		if !ok {
			return errors.New(tr("سرور پیدا نشد", "Server not found"))
		}
		go func() {
			fmMu.Lock()
			f := fmSessions[id]
			if f == nil {
				f = &fmSess{}
				fmSessions[id] = f
			}
			fmMu.Unlock()
			f.mu.Lock()
			f.drop()
			f.srv, f.pass = s, pass
			err := f.dial()
			var hc *hostKeyChanged
			if errors.As(err, &hc) && s.AutoTrust {
				if err = acceptNewKey(s, hc.Key); err == nil {
					err = f.dial()
				}
			}
			home := f.home
			f.mu.Unlock()
			switch {
			case err == nil:
				js("onFsOpen", id, map[string]string{"home": home}, "")
				if pass != "" && saveKey {
					go func() {
						legacy, err := installKey(s, pass)
						if err != nil {
							toastErr(tr("نصب کلید نشد: ", "Couldn't install the key: ") + err.Error())
							return
						}
						update(id, func(p *server) {
							p.KeyOK = true
							if legacy {
								p.Legacy = true
							}
						})
						js("onToast", tr("ورود بدون رمز فعال شد", "Passwordless login is set up"), "ok")
					}()
				}
			case errors.As(err, &hc):
				askTrust("sftp", []string{id}, []server{s}, []ssh.PublicKey{hc.Key})
			case errors.Is(err, errNeedPass) || strings.Contains(err.Error(), "unable to authenticate") || strings.Contains(err.Error(), "no supported methods"):
				msg := ""
				if pass != "" {
					msg = tr("رمز یا یوزرنیم اشتباهه", "Wrong password or username")
				}
				js("onFsOpen", id, nil, "pass", msg)
			case errors.Is(err, errNoSFTP):
				js("onFsOpen", id, nil, "nosftp")
			default:
				js("onFsOpen", id, nil, friendlyDialErr(err).Error())
			}
		}()
		return nil
	})

	w.Bind("fsClose", func(id string) {
		fmMu.Lock()
		f := fmSessions[id]
		delete(fmSessions, id)
		fmMu.Unlock()
		xfMu.Lock()
		for _, x := range xfers {
			if x.sid == id {
				x.cancel.Store(true)
			}
		}
		xfMu.Unlock()
		if f != nil {
			go func() {
				time.Sleep(time.Second) // let running transfers notice the cancel
				f.mu.Lock()
				f.drop()
				f.pass = ""
				f.mu.Unlock()
			}()
		}
	})

	w.Bind("fsDo", func(req int, id, op string, a []string) error {
		f := getSess(id)
		if f == nil {
			return errors.New(tr("اتصال بسته شده", "The connection is closed"))
		}
		go func() {
			data, err := f.fsOp(op, a)
			msg := ""
			if err != nil {
				switch err.Error() {
				case "big", "binary", "dir":
					msg = err.Error()
				default:
					msg = fsErr(err)
				}
			}
			js("onFs", req, data, msg)
		}()
		return nil
	})

	w.Bind("fsDownload", func(id string, paths []string) error {
		f := getSess(id)
		if f == nil {
			return errors.New(tr("اتصال بسته شده", "The connection is closed"))
		}
		go f.download(paths)
		return nil
	})

	// uploads: upStart → upChunk… (each answered by onUpAck) → upEnd
	w.Bind("upStart", func(xid, id, remote string, size int64) error {
		f := getSess(id)
		if f == nil {
			return errors.New(tr("اتصال بسته شده", "The connection is closed"))
		}
		remote = f.resolve(remote)
		x := &xfer{id: xid, sid: id, kind: "up", name: path.Base(remote), total: size, ch: make(chan []byte, 8)}
		xfMu.Lock()
		xfers[xid] = x
		xfMu.Unlock()
		x.report("run", "", true)
		go f.upload(x, remote, x.ch)
		return nil
	})
	w.Bind("upChunk", func(xid, b64 string) error {
		xfMu.Lock()
		x := xfers[xid]
		xfMu.Unlock()
		if x == nil {
			return errors.New("cancel")
		}
		b, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return err
		}
		select {
		case x.ch <- b:
			return nil
		default:
			return errors.New("busy")
		}
	})
	w.Bind("upEnd", func(xid string, abort bool) {
		xfMu.Lock()
		x := xfers[xid]
		xfMu.Unlock()
		if x != nil && x.ch != nil {
			if abort {
				x.abort.Store(true)
			}
			close(x.ch)
			x.ch = nil
		}
	})

	w.Bind("xferCancel", func(xid string) {
		xfMu.Lock()
		if x := xfers[xid]; x != nil {
			x.cancel.Store(true)
		}
		xfMu.Unlock()
	})
	w.Bind("openLocal", func(p string) {
		if p != "" && fileExists(p) {
			shellOpen(p)
		}
	})
	w.Bind("showLocal", func(p string) {
		if p != "" && fileExists(p) {
			showInFolder(p)
		}
	})
	w.Bind("openDownloads", func() { shellOpen(downloadsDir()) })
}
