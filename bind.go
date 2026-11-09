//go:build windows

package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type uiState struct {
	Servers    []server `json:"servers"`
	SameWindow bool     `json:"sameWindow"`
	AutoClose  bool     `json:"autoClose"`
	WTFound    bool     `json:"wtFound"`
	Lang       string   `json:"lang"`
}

type changedItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Addr string `json:"addr"`
	FP   string `json:"fp"`
}

var (
	pendMu    sync.Mutex
	pending   = map[string]ssh.PublicKey{} // new host keys waiting for the user's OK
	retrusted = map[string]bool{}          // servers whose host key was just replaced (rebuilt)
)

func newID() string {
	const abc = "0123456789abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 12)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(abc))))
		b[i] = abc[n.Int64()]
	}
	return string(b)
}

func displayAddr(s server) string {
	a := s.Host
	if s.User != "" {
		a = s.User + "@" + a
	}
	if s.Port != 0 && s.Port != 22 {
		a += fmt.Sprintf(":%d", s.Port)
	}
	return a
}

// acceptNewKey saves a server's new host key (rebuilt server) and remembers that it happened.
func acceptNewKey(s server, key ssh.PublicKey) error {
	if err := replaceHostKey(s, key); err != nil {
		return errors.New(tr("ذخیرهٔ کلید جدید توی known_hosts نشد: ", "Couldn't save the new key in known_hosts: ") + err.Error())
	}
	pendMu.Lock()
	delete(pending, s.ID)
	retrusted[s.ID] = true
	pendMu.Unlock()
	return nil
}

// askTrust shows the "host key changed" dialog; then = what to do after the user accepts.
func askTrust(then string, ids []string, changed []server, keys []ssh.PublicKey) {
	items := make([]changedItem, len(changed))
	pendMu.Lock()
	for i, s := range changed {
		pending[s.ID] = keys[i]
		items[i] = changedItem{ID: s.ID, Name: s.Name, Addr: displayAddr(s), FP: fingerprint(keys[i])}
	}
	pendMu.Unlock()
	js("onHostChanged", map[string]interface{}{"then": then, "ids": ids, "items": items})
}

// prepare checks servers before opening terminals. Changed host keys are accepted automatically
// for auto-trust servers; the rest are returned for the user to confirm.
// needKey is a server whose key login stopped working right after its host key was replaced.
func prepare(list []server) (changed []server, keys []ssh.PublicKey, needKey string) {
	type res struct {
		i int
		r probeResult
	}
	ch := make(chan res, len(list))
	for i, s := range list {
		go func(i int, s server) { ch <- res{i, probe(s)} }(i, s)
	}
	results := make([]probeResult, len(list))
	for range list {
		r := <-ch
		results[r.i] = r.r
	}
	for i, s := range list {
		r := results[i]
		if r.Changed != nil {
			if !s.AutoTrust {
				changed = append(changed, s)
				keys = append(keys, r.Changed.Key)
				continue
			}
			if err := acceptNewKey(s, r.Changed.Key); err != nil {
				toastErr(err.Error())
				continue
			}
			r = probe(s)
		}
		pendMu.Lock()
		was := retrusted[s.ID]
		delete(retrusted, s.ID)
		pendMu.Unlock()
		if was && s.KeyOK && r.Changed == nil && r.NetErr == nil && !r.AuthOK && len(list) == 1 {
			update(s.ID, func(p *server) { p.KeyOK = false })
			needKey = s.ID
		}
	}
	return
}

func idsOf(list []server) []string {
	out := make([]string, len(list))
	for i, s := range list {
		out[i] = s.ID
	}
	return out
}

func serversFor(ids []string) []server {
	mu.Lock()
	defer mu.Unlock()
	var out []server
	for _, id := range ids {
		if _, p := find(id); p != nil {
			out = append(out, *p)
		}
	}
	return out
}

func afterLaunch(err error) {
	if err != nil {
		toastErr(tr("باز کردن Terminal نشد: ", "Couldn't open Terminal: ") + err.Error())
		return
	}
	mu.Lock()
	ac := cfg.AutoClose
	mu.Unlock()
	if ac {
		time.Sleep(400 * time.Millisecond)
		w.Dispatch(closeWindow)
	}
}

func bindAll(hwnd uintptr) {
	w.Bind("getState", func() uiState {
		mu.Lock()
		defer mu.Unlock()
		list := make([]server, len(cfg.Servers))
		copy(list, cfg.Servers)
		return uiState{Servers: list, SameWindow: cfg.SameWindow, AutoClose: cfg.AutoClose, WTFound: wtPath() != "", Lang: cfg.Lang}
	})

	w.Bind("setLang", func(l string) {
		if l != "fa" && l != "en" {
			return
		}
		mu.Lock()
		cfg.Lang = l
		saveConfig()
		mu.Unlock()
	})

	w.Bind("setOption", func(name string, v bool) {
		mu.Lock()
		defer mu.Unlock()
		switch name {
		case "sameWindow":
			cfg.SameWindow = v
		case "autoClose":
			cfg.AutoClose = v
		default:
			return
		}
		saveConfig()
	})

	w.Bind("pingAll", func() {
		mu.Lock()
		list := make([]server, len(cfg.Servers))
		copy(list, cfg.Servers)
		mu.Unlock()
		for _, s := range list {
			go func(s server) { js("onPing", s.ID, pingOne(s)) }(s)
		}
	})

	// connect checks the server first (changed host key, broken key login) and then opens Terminal.
	// skipCheck opens Terminal straight away.
	w.Bind("connect", func(id string, skipCheck bool) error {
		s, ok := get(id)
		if !ok {
			return errors.New(tr("سرور پیدا نشد", "Server not found"))
		}
		go func() {
			if !skipCheck {
				changed, keys, needKey := prepare([]server{s})
				if len(changed) > 0 {
					askTrust("connect", []string{id}, changed, keys)
					return
				}
				if needKey != "" {
					js("onNeedKey", needKey)
					return
				}
				s, _ = get(id)
			}
			mu.Lock()
			same := cfg.SameWindow
			mu.Unlock()
			afterLaunch(launchOne(s, same))
		}()
		return nil
	})

	openMany := func(then string, ids []string, launch func([]server) error) error {
		list := serversFor(ids)
		if len(list) == 0 {
			return errors.New(tr("حداقل یه سرور انتخاب کن", "Select at least one server"))
		}
		go func() {
			changed, keys, _ := prepare(list)
			if len(changed) > 0 {
				askTrust(then, idsOf(list), changed, keys)
				return
			}
			afterLaunch(launch(serversFor(ids)))
		}()
		return nil
	}
	w.Bind("openSplit", func(ids []string) error { return openMany("split", ids, launchSplit) })
	w.Bind("openTabs", func(ids []string) error { return openMany("tabs", ids, launchTabs) })

	// trustHost saves the new host keys the user just accepted.
	w.Bind("trustHost", func(ids []string, always bool) error {
		for _, id := range ids {
			s, ok := get(id)
			if !ok {
				continue
			}
			pendMu.Lock()
			key := pending[id]
			pendMu.Unlock()
			if key == nil {
				continue
			}
			if err := acceptNewKey(s, key); err != nil {
				return err
			}
			update(id, func(p *server) { p.AutoTrust = always })
		}
		return nil
	})

	w.Bind("saveServer", func(in server) (server, error) {
		in.Name = strings.TrimSpace(in.Name)
		in.User = strings.TrimSpace(in.User)
		in.Host = strings.TrimSpace(in.Host)
		if in.Host == "" || strings.ContainsAny(in.Host, " @;\"'") {
			return server{}, errors.New(tr("آدرس سرور رو درست بنویس", "Enter a valid server address"))
		}
		if strings.ContainsAny(in.User, " @;\"'") {
			return server{}, errors.New(tr("یوزر نباید فاصله یا @ داشته باشه", "The user can't contain spaces or @"))
		}
		if in.Port <= 0 || in.Port > 65535 {
			in.Port = 22
		}
		if in.Name == "" {
			in.Name = in.Host
		}
		mu.Lock()
		defer mu.Unlock()
		if in.ID != "" {
			if _, old := find(in.ID); old != nil {
				same := old.Host == in.Host && old.User == in.User && old.Port == in.Port
				in.KeyOK = old.KeyOK && same
				in.Legacy = old.Legacy && same
				in.U = nowMs()
				*old = in
				defer syncSoon()
				return in, saveConfig()
			}
		}
		in.ID = newID()
		in.KeyOK, in.Legacy = false, false
		in.U = nowMs()
		cfg.Servers = append(cfg.Servers, in)
		defer syncSoon()
		return in, saveConfig()
	})

	w.Bind("deleteServer", func(id string) error {
		mu.Lock()
		defer mu.Unlock()
		i, _ := find(id)
		if i < 0 {
			return nil
		}
		cfg.Servers = append(cfg.Servers[:i], cfg.Servers[i+1:]...)
		if cfg.Deleted == nil {
			cfg.Deleted = map[string]int64{}
		}
		cfg.Deleted[id] = nowMs()
		defer syncSoon()
		return saveConfig()
	})

	w.Bind("moveServer", func(id string, d int) error {
		mu.Lock()
		defer mu.Unlock()
		i, _ := find(id)
		j := i + d
		if i < 0 || j < 0 || j >= len(cfg.Servers) {
			return nil
		}
		cfg.Servers[i], cfg.Servers[j] = cfg.Servers[j], cfg.Servers[i]
		cfg.OrderU = nowMs()
		defer syncSoon()
		return saveConfig()
	})

	// password -> ssh key on the server (password is used once, never stored)
	w.Bind("installKey", func(id, password string) error {
		s, ok := get(id)
		if !ok {
			return errors.New(tr("سرور پیدا نشد", "Server not found"))
		}
		if password == "" {
			return errors.New(tr("رمز سرور رو بنویس", "Enter the server password"))
		}
		go func() {
			legacy, err := installKey(s, password)
			var hc *hostKeyChanged
			if errors.As(err, &hc) {
				if !s.AutoTrust {
					askTrust("key", []string{id}, []server{s}, []ssh.PublicKey{hc.Key})
					return
				}
				if err = acceptNewKey(s, hc.Key); err == nil {
					legacy, err = installKey(s, password)
				}
			}
			pendMu.Lock()
			delete(retrusted, id)
			pendMu.Unlock()
			if err == nil {
				update(id, func(p *server) {
					p.KeyOK = true
					if legacy {
						p.Legacy = true
					}
				})
				js("onKey", id, true, "")
				return
			}
			if errors.As(err, &hc) {
				err = errors.New(errHostKeyChangedMsg())
			}
			js("onKey", id, false, err.Error())
		}()
		return nil
	})

	// runMulti runs one command on several servers at once (key login only).
	w.Bind("runMulti", func(runID int, ids []string, cmd string) error {
		list := serversFor(ids)
		if len(list) == 0 || strings.TrimSpace(cmd) == "" {
			return errors.New(tr("حداقل یه سرور انتخاب کن", "Select at least one server"))
		}
		for _, s := range list {
			go func(s server) {
				t0 := time.Now()
				res := map[string]interface{}{"run": runID, "id": s.ID, "code": 0, "out": "", "err": ""}
				defer func() {
					res["ms"] = time.Since(t0).Milliseconds()
					js("onRun", res)
				}()
				auth := keyAuth()
				if auth == nil {
					res["err"] = tr("هنوز کلید SSH نساختی؛ اول «ورود بدون رمز» رو برای این سرور فعال کن", "No SSH key yet; set up “Passwordless login” for this server first")
					return
				}
				c, err := dialChecked(s, auth)
				var hc *hostKeyChanged
				if errors.As(err, &hc) && s.AutoTrust {
					if err = acceptNewKey(s, hc.Key); err == nil {
						c, err = dialChecked(s, auth)
					}
				}
				if err != nil {
					switch {
					case errors.As(err, &hc):
						res["err"] = errHostKeyChangedMsg() + tr("؛ یه بار روی کارتش بزن تا کلید جدید رو قبول کنی.", ". Click its card once to accept the new key.")
					case strings.Contains(err.Error(), "unable to authenticate"):
						res["err"] = tr("ورود با کلید روی این سرور کار نمی‌کنه؛ «ورود بدون رمز» رو براش فعال کن", "Key login doesn't work on this server; set up “Passwordless login” for it")
					default:
						res["err"] = friendlyDialErr(err).Error()
					}
					return
				}
				defer c.Close()
				out, code, err := run(c, cmd, 90*time.Second)
				res["out"], res["code"] = out, code
				if err != nil {
					res["err"] = err.Error()
				}
			}(s)
		}
		return nil
	})

	w.Bind("getSync", func() syncInfo { return getSyncInfo() })
	w.Bind("syncNow", func() { go syncAndReport() })
	w.Bind("setupSync", func(input string, replace bool) {
		go func() {
			res, err := setupSync(input, replace)
			msg := ""
			if err != nil {
				msg = err.Error()
			}
			js("onSetup", res, msg)
		}()
	})
	w.Bind("loginGitHub", func() { go loginGitHub() })
	w.Bind("cancelLogin", func() { cancelLogin() })
	w.Bind("stopSync", func() {
		mu.Lock()
		cfg.Sync = nil
		saveConfig()
		mu.Unlock()
	})
	w.Bind("openURL", func(u string) {
		if strings.HasPrefix(u, "https://") {
			openBrowser(u)
		}
	})

	w.Bind("appVersion", func() string { return appVersion })
	w.Bind("checkUpdate", func() {
		go func() {
			in := checkUpdate()
			if in.Err == "" {
				mu.Lock()
				cfg.UpdateCheck, cfg.Update = nowMs(), &in
				saveConfig()
				mu.Unlock()
			}
			js("onUpdate", in, true)
		}()
	})
	w.Bind("doUpdate", func(url string) {
		go func() {
			if err := selfUpdate(url); err != nil {
				js("onUpdateFail", err.Error())
			}
		}()
	})

	w.Bind("fitWindow", func(cw, ch int, dpr float64) { fitWindow(cw, ch, dpr) })
	w.Bind("dragStart", func() { dragStart() })
	w.Bind("dragMove", func(dx, dy, dpr float64) { dragMove(dx, dy, dpr) })
	w.Bind("minimize", func() { minimizeWindow() })
	w.Bind("closeApp", func() { closeWindow() })
}
