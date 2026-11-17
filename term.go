//go:build windows

package main

// Built-in terminal: one SSH shell per tab (tid, picked by the page). Output is batched and
// sent to the page as base64 (onTermData), the page sends keystrokes back with termInput.

import (
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/crypto/ssh"
)

type termSess struct {
	tid   int
	srvID string

	mu      sync.Mutex
	sc      *ssh.Client
	sess    *ssh.Session
	stdin   chan []byte // keystrokes, written in order by one goroutine
	out     []byte
	pending bool // a flush is scheduled
	closed  bool
}

var (
	termMu   sync.Mutex
	terms    = map[int]*termSess{}
	termPass = map[string]string{} // server id -> password that worked (memory only, for more tabs / reconnects)
)

const termFlushEvery = 12 * time.Millisecond

func (t *termSess) write(b []byte) {
	t.mu.Lock()
	t.out = append(t.out, b...)
	if !t.pending {
		t.pending = true
		time.AfterFunc(termFlushEvery, t.flush)
	}
	t.mu.Unlock()
}

func (t *termSess) flush() {
	t.mu.Lock()
	b := t.out
	t.out, t.pending = nil, false
	t.mu.Unlock()
	for len(b) > 0 { // keep each message reasonable
		n := min(len(b), 512<<10)
		js("onTermData", t.tid, base64.StdEncoding.EncodeToString(b[:n]))
		b = b[n:]
	}
}

func (t *termSess) shutdown() {
	t.mu.Lock()
	t.closed = true
	sess, sc := t.sess, t.sc
	t.sess, t.sc, t.stdin = nil, nil, nil
	t.mu.Unlock()
	if sess != nil {
		sess.Close()
	}
	if sc != nil {
		sc.Close()
	}
}

func termGet(tid int) *termSess {
	termMu.Lock()
	defer termMu.Unlock()
	return terms[tid]
}

// termStart dials and starts a login shell. Errors: *hostKeyChanged, errNeedPass, auth errors, net errors.
func termStart(t *termSess, s server, pass string, cols, rows int) error {
	auth := keyAuth()
	if pass != "" {
		auth = append(auth, passAuth(pass)...)
	}
	if len(auth) == 0 {
		return errNeedPass
	}
	sc, err := dialChecked(s, auth)
	if err != nil {
		return err
	}
	sess, err := sc.NewSession()
	if err != nil {
		sc.Close()
		return err
	}
	modes := ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 115200, ssh.TTY_OP_OSPEED: 115200}
	if cols < 10 || rows < 3 {
		cols, rows = 100, 30
	}
	if err := sess.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		sess.Close()
		sc.Close()
		return err
	}
	sess.Setenv("COLORTERM", "truecolor") // most servers ignore it (AcceptEnv); harmless
	stdin, err := sess.StdinPipe()
	if err != nil {
		sess.Close()
		sc.Close()
		return err
	}
	stdout, _ := sess.StdoutPipe()
	stderr, _ := sess.StderrPipe()
	if err := sess.Shell(); err != nil {
		sess.Close()
		sc.Close()
		return err
	}
	inCh, done := make(chan []byte, 4096), make(chan struct{})
	t.mu.Lock()
	if t.closed { // tab closed while connecting
		t.mu.Unlock()
		sess.Close()
		sc.Close()
		return nil
	}
	t.sc, t.sess, t.stdin = sc, sess, inCh
	t.mu.Unlock()
	go func() {
		for {
			select {
			case b := <-inCh:
				if _, err := stdin.Write(b); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	pump := func(r io.Reader) {
		buf := make([]byte, 32<<10)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				t.write(append([]byte(nil), buf[:n]...))
			}
			if err != nil {
				return
			}
		}
	}
	go pump(stdout)
	go pump(stderr)
	go func() { // keepalive: notices a dead link (sleep, NAT timeout) instead of hanging forever
		tk := time.NewTicker(25 * time.Second)
		defer tk.Stop()
		for {
			select {
			case <-done:
				return
			case <-tk.C:
			}
			if _, _, err := sc.SendRequest("keepalive@openssh.com", true, nil); err != nil {
				sc.Close()
				return
			}
		}
	}()
	go func() {
		err := sess.Wait()
		close(done)
		time.Sleep(50 * time.Millisecond) // let the last output through
		t.flush()
		t.mu.Lock()
		mine := t.sess == sess
		closed := t.closed
		if mine {
			t.sess, t.sc, t.stdin = nil, nil, nil
		}
		t.mu.Unlock()
		sc.Close()
		if !mine || closed {
			return
		}
		msg := ""
		var ee *ssh.ExitError
		if err != nil && !errors.As(err, &ee) {
			msg = tr("اتصال قطع شد", "Connection lost")
		}
		js("onTermState", t.tid, "closed", msg)
	}()
	return nil
}

func bindTerm() {
	// termOpen connects tab tid to a server. pass is optional (asked by the page after "pass").
	// Answers onTermState(tid, state, msg): "open" | "pass" | "closed" | "error".
	w.Bind("termOpen", func(tid int, id, pass string, saveKey bool, cols, rows int) error {
		s, ok := get(id)
		if !ok {
			return errors.New(tr("سرور پیدا نشد", "Server not found"))
		}
		termMu.Lock()
		if old := terms[tid]; old != nil {
			termMu.Unlock()
			old.shutdown()
			termMu.Lock()
		}
		t := &termSess{tid: tid, srvID: id}
		terms[tid] = t
		if pass == "" {
			pass = termPass[id]
		}
		termMu.Unlock()
		go func() {
			err := termStart(t, s, pass, cols, rows)
			var hc *hostKeyChanged
			if errors.As(err, &hc) && s.AutoTrust {
				if err = acceptNewKey(s, hc.Key); err == nil {
					err = termStart(t, s, pass, cols, rows)
				}
			}
			switch {
			case err == nil:
				if pass != "" {
					termMu.Lock()
					termPass[id] = pass
					termMu.Unlock()
				}
				js("onTermState", tid, "open", "")
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
				askTrust("term", []string{id}, []server{s}, []ssh.PublicKey{hc.Key})
			case errors.Is(err, errNeedPass) || strings.Contains(err.Error(), "unable to authenticate") || strings.Contains(err.Error(), "no supported methods"):
				msg := ""
				if pass != "" {
					msg = tr("رمز یا یوزرنیم اشتباهه", "Wrong password or username")
					termMu.Lock()
					delete(termPass, id)
					termMu.Unlock()
				}
				js("onTermState", tid, "pass", msg)
			default:
				js("onTermState", tid, "error", friendlyDialErr(err).Error())
			}
		}()
		return nil
	})

	w.Bind("termInput", func(tid int, data string) {
		if t := termGet(tid); t != nil {
			t.mu.Lock()
			in := t.stdin
			t.mu.Unlock()
			if in != nil {
				select { // never block the UI thread on a slow link
				case in <- []byte(data):
				default:
				}
			}
		}
	})

	w.Bind("termResize", func(tid, cols, rows int) {
		if t := termGet(tid); t != nil && cols > 0 && rows > 0 {
			t.mu.Lock()
			sess := t.sess
			t.mu.Unlock()
			if sess != nil {
				go sess.WindowChange(rows, cols)
			}
		}
	})

	w.Bind("termClose", func(tid int) {
		termMu.Lock()
		t := terms[tid]
		delete(terms, tid)
		termMu.Unlock()
		if t != nil {
			go t.shutdown()
		}
	})

	w.Bind("openURL", func(u string) { // Ctrl+click on a link in the terminal
		if strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://") {
			shellOpen(u)
		}
	})
	w.Bind("clipGet", clipGet)
	w.Bind("clipSet", clipSet)
}

// ---------------------------------------------------------------- clipboard (Win32)

var (
	pOpenClipboard    = user32.NewProc("OpenClipboard")
	pCloseClipboard   = user32.NewProc("CloseClipboard")
	pEmptyClipboard   = user32.NewProc("EmptyClipboard")
	pGetClipboardData = user32.NewProc("GetClipboardData")
	pSetClipboardData = user32.NewProc("SetClipboardData")
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	pGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	pGlobalFree       = kernel32.NewProc("GlobalFree")
	pGlobalLock       = kernel32.NewProc("GlobalLock")
	pGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	pLstrlenW         = kernel32.NewProc("lstrlenW")
)

const cfUnicodeText = 13

func openClip() bool {
	for i := 0; i < 10; i++ { // another app may hold it for a moment
		if r, _, _ := pOpenClipboard.Call(mainHwnd); r != 0 {
			return true
		}
		time.Sleep(15 * time.Millisecond)
	}
	return false
}

func clipGet() string {
	if !openClip() {
		return ""
	}
	defer pCloseClipboard.Call()
	h, _, _ := pGetClipboardData.Call(cfUnicodeText)
	if h == 0 {
		return ""
	}
	p, _, _ := pGlobalLock.Call(h)
	if p == 0 {
		return ""
	}
	defer pGlobalUnlock.Call(h)
	n, _, _ := pLstrlenW.Call(p)
	return syscall.UTF16ToString(unsafe.Slice(*(**uint16)(unsafe.Pointer(&p)), int(n)))
}

func clipSet(s string) bool {
	u, err := syscall.UTF16FromString(strings.ReplaceAll(s, "\x00", ""))
	if err != nil {
		return false
	}
	if !openClip() {
		return false
	}
	defer pCloseClipboard.Call()
	pEmptyClipboard.Call()
	h, _, _ := pGlobalAlloc.Call(0x0002, uintptr(len(u)*2)) // GMEM_MOVEABLE
	if h == 0 {
		return false
	}
	p, _, _ := pGlobalLock.Call(h)
	if p == 0 {
		pGlobalFree.Call(h)
		return false
	}
	copy(unsafe.Slice(*(**uint16)(unsafe.Pointer(&p)), len(u)), u)
	pGlobalUnlock.Call(h)
	if r, _, _ := pSetClipboardData.Call(cfUnicodeText, h); r == 0 {
		pGlobalFree.Call(h)
		return false
	}
	return true
}
