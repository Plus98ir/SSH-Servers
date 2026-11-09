//go:build windows

// Server-list sync between devices through a secret GitHub Gist.
// The gist holds one file, encrypted with AES-256-GCM; the key never leaves the
// devices (it travels only inside the sync link / QR code).
// The phone side is the web app in docs/app/ (same format, same merge rules).
package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

const (
	syncPage  = "https://corenetbase.com/SSH-Servers/app/" // phone web app
	gistFile  = "ssh-servers.json"
	gistDesc  = "SSH Servers sync (encrypted)"
	tombLife  = 180 * 24 * time.Hour
	tokenPage = "https://github.com/settings/tokens/new?scopes=gist&description=SSH%20Servers%20sync"
)

type syncCfg struct {
	Token string `json:"token"`
	Gist  string `json:"gist"`
	Key   string `json:"key"` // base64url, 32 bytes
	Last  int64  `json:"last,omitempty"`
}

// what goes into the gist (no device-only fields like keyOK / autoTrust)
type syncSrv struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	User   string `json:"user"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Color  string `json:"color"`
	Legacy bool   `json:"legacy,omitempty"`
	U      int64  `json:"u"`
}

type syncDoc struct {
	V       int              `json:"v"`
	Servers []syncSrv        `json:"servers"`
	Deleted map[string]int64 `json:"deleted"`
	OrderU  int64            `json:"orderU"`
}

type envelope struct {
	App string `json:"app"`
	V   int    `json:"v"`
	IV  string `json:"iv"`
	CT  string `json:"ct"`
}

type syncInfo struct {
	On       bool   `json:"on"`
	Link     string `json:"link,omitempty"`
	QR       string `json:"qr,omitempty"`
	Last     int64  `json:"last,omitempty"`
	TokenURL string `json:"tokenURL"`
	Login    bool   `json:"login"` // "Sign in with GitHub" available
}

func nowMs() int64 { return time.Now().UnixMilli() }

// ---------------------------------------------------------------- merge (same rules as docs/app/sync.js)

func localDoc() syncDoc { // call with mu held
	d := syncDoc{V: 1, Servers: []syncSrv{}, Deleted: map[string]int64{}, OrderU: cfg.OrderU}
	for k, v := range cfg.Deleted {
		d.Deleted[k] = v
	}
	for _, s := range cfg.Servers {
		d.Servers = append(d.Servers, syncSrv{s.ID, s.Name, s.User, s.Host, s.Port, s.Color, s.Legacy, s.U})
	}
	return d
}

// merge: newest edit of each server wins, deletions win over older edits,
// order comes from the side that reordered last. Ties go to a.
func merge(a, b syncDoc) syncDoc {
	del := map[string]int64{}
	for _, m := range []map[string]int64{a.Deleted, b.Deleted} {
		for k, v := range m {
			if v > del[k] {
				del[k] = v
			}
		}
	}
	win := map[string]syncSrv{}
	for _, list := range [][]syncSrv{a.Servers, b.Servers} {
		for _, s := range list {
			if cur, ok := win[s.ID]; !ok || s.U > cur.U {
				win[s.ID] = s
			}
		}
	}
	first, second := a, b
	if b.OrderU > a.OrderU {
		first, second = b, a
	}
	out := []syncSrv{}
	seen := map[string]bool{}
	for _, list := range [][]syncSrv{first.Servers, second.Servers} {
		for _, s := range list {
			if seen[s.ID] {
				continue
			}
			seen[s.ID] = true
			w := win[s.ID]
			if d, ok := del[s.ID]; ok && d >= w.U {
				continue
			}
			out = append(out, w)
		}
	}
	cut := nowMs() - tombLife.Milliseconds()
	for k, v := range del {
		if v < cut {
			delete(del, k)
		}
	}
	ou := a.OrderU
	if b.OrderU > ou {
		ou = b.OrderU
	}
	return syncDoc{V: 1, Servers: out, Deleted: del, OrderU: ou}
}

func sameDoc(a, b syncDoc) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return bytes.Equal(x, y)
}

// applyDoc replaces the local list with the merged one, keeping device-only fields. Call with mu held.
func applyDoc(d syncDoc) {
	old := map[string]server{}
	for _, s := range cfg.Servers {
		old[s.ID] = s
	}
	list := make([]server, 0, len(d.Servers))
	for _, r := range d.Servers {
		s := server{ID: r.ID, Name: r.Name, User: r.User, Host: r.Host, Port: r.Port, Color: r.Color, Legacy: r.Legacy, U: r.U}
		if o, ok := old[r.ID]; ok {
			s.AutoTrust = o.AutoTrust
			s.KeyOK = o.KeyOK && o.Host == s.Host && o.User == s.User && o.Port == s.Port
		}
		list = append(list, s)
	}
	cfg.Servers = list
	cfg.Deleted = d.Deleted
	cfg.OrderU = d.OrderU
}

// ---------------------------------------------------------------- crypto

func syncKey(k string) ([]byte, error) {
	b, err := base64.RawURLEncoding.DecodeString(k)
	if err != nil || len(b) != 32 {
		return nil, errors.New("bad key")
	}
	return b, nil
}

func seal(key []byte, d syncDoc) (string, error) {
	plain, _ := json.Marshal(d)
	blk, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	g, _ := cipher.NewGCM(blk)
	iv := make([]byte, 12)
	rand.Read(iv)
	env := envelope{App: "ssh-servers", V: 1,
		IV: base64.StdEncoding.EncodeToString(iv),
		CT: base64.StdEncoding.EncodeToString(g.Seal(nil, iv, plain, nil))}
	b, _ := json.MarshalIndent(env, "", " ")
	return string(b), nil
}

var errBadKey = errors.New("badkey")

func unseal(key []byte, content string) (syncDoc, error) {
	var env envelope
	var d syncDoc
	if err := json.Unmarshal([]byte(content), &env); err != nil || env.App != "ssh-servers" {
		return d, errBadKey
	}
	iv, e1 := base64.StdEncoding.DecodeString(env.IV)
	ct, e2 := base64.StdEncoding.DecodeString(env.CT)
	if e1 != nil || e2 != nil {
		return d, errBadKey
	}
	blk, _ := aes.NewCipher(key)
	g, _ := cipher.NewGCM(blk)
	plain, err := g.Open(nil, iv, ct, nil)
	if err != nil {
		return d, errBadKey
	}
	if err := json.Unmarshal(plain, &d); err != nil {
		return d, errBadKey
	}
	if d.Deleted == nil {
		d.Deleted = map[string]int64{}
	}
	if d.Servers == nil {
		d.Servers = []syncSrv{}
	}
	return d, nil
}

// ---------------------------------------------------------------- GitHub API

var (
	errNotFound = errors.New("notfound")
	errAuth     = errors.New("auth")
	httpc       = &http.Client{Timeout: 25 * time.Second}
)

func gh(method, path, token string, body, out interface{}) error {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, "https://api.github.com"+path, rd)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "SSH-Servers")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := httpc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == 401 || res.StatusCode == 403:
		return errAuth
	case res.StatusCode == 404:
		return errNotFound
	case res.StatusCode >= 300:
		return fmt.Errorf("GitHub: %s", res.Status)
	}
	if out != nil {
		return json.NewDecoder(res.Body).Decode(out)
	}
	return nil
}

type gistFileT struct {
	Content string `json:"content"`
}
type gistT struct {
	ID    string               `json:"id"`
	Files map[string]gistFileT `json:"files"`
}

func gistRead(sc syncCfg) (string, error) {
	var g gistT
	if err := gh("GET", "/gists/"+sc.Gist, sc.Token, nil, &g); err != nil {
		return "", err
	}
	f, ok := g.Files[gistFile]
	if !ok {
		return "", errNotFound
	}
	return f.Content, nil
}

func gistWrite(sc syncCfg, content string) error {
	return gh("PATCH", "/gists/"+sc.Gist, sc.Token,
		map[string]interface{}{"files": map[string]gistFileT{gistFile: {content}}}, nil)
}

func gistCreate(token, content string) (string, error) {
	var g gistT
	err := gh("POST", "/gists", token, map[string]interface{}{
		"description": gistDesc, "public": false,
		"files": map[string]gistFileT{gistFile: {content}},
	}, &g)
	return g.ID, err
}

func gistFind(token string) (string, error) {
	var list []gistT
	if err := gh("GET", "/gists?per_page=100", token, nil, &list); err != nil {
		return "", err
	}
	for _, g := range list {
		if _, ok := g.Files[gistFile]; ok {
			return g.ID, nil
		}
	}
	return "", nil
}

func syncErr(err error) error {
	switch {
	case errors.Is(err, errAuth):
		return errors.New(tr("توکن GitHub قبول نشد (اشتباهه، منقضی شده یا دسترسی gist نداره)", "GitHub didn't accept the token (wrong, expired or without gist access)"))
	case errors.Is(err, errNotFound):
		return errors.New(tr("فایل سینک روی GitHub پیدا نشد", "The sync file wasn't found on GitHub"))
	case errors.Is(err, errBadKey):
		return errors.New(tr("کلید سینک با این فایل نمی‌خونه؛ لینک سینک رو دوباره از دستگاه دیگه بیار", "The sync key doesn't match this file; get the sync link again from your other device"))
	}
	return errors.New(tr("به GitHub وصل نشد: ", "Couldn't reach GitHub: ") + err.Error())
}

// ---------------------------------------------------------------- link

func syncLink(sc syncCfg) string {
	return syncPage + "#g=" + url.QueryEscape(sc.Gist) + "&k=" + url.QueryEscape(sc.Key) + "&t=" + url.QueryEscape(sc.Token)
}

func parseLink(s string) (syncCfg, bool) {
	i := strings.Index(s, "#")
	if i < 0 {
		return syncCfg{}, false
	}
	q, err := url.ParseQuery(s[i+1:])
	if err != nil {
		return syncCfg{}, false
	}
	sc := syncCfg{Gist: q.Get("g"), Key: q.Get("k"), Token: q.Get("t")}
	if sc.Gist == "" || sc.Token == "" {
		return syncCfg{}, false
	}
	if _, err := syncKey(sc.Key); err != nil {
		return syncCfg{}, false
	}
	return sc, true
}

// ---------------------------------------------------------------- run

var (
	syncMu    sync.Mutex // one sync at a time
	syncTimer *time.Timer
	timerMu   sync.Mutex
)

type syncStatus struct {
	OK      bool   `json:"ok"`
	Msg     string `json:"msg,omitempty"`
	Last    int64  `json:"last,omitempty"`
	Changed bool   `json:"changed"`
}

// runSync pulls, merges and pushes. Returns whether the local list changed.
func runSync() (bool, error) {
	syncMu.Lock()
	defer syncMu.Unlock()
	mu.Lock()
	if cfg.Sync == nil {
		mu.Unlock()
		return false, nil
	}
	sc := *cfg.Sync
	mu.Unlock()
	key, err := syncKey(sc.Key)
	if err != nil {
		return false, errBadKey
	}
	content, err := gistRead(sc)
	if err != nil {
		return false, err
	}
	remote, err := unseal(key, content)
	if err != nil {
		return false, err
	}
	mu.Lock()
	local := localDoc()
	merged := merge(local, remote)
	changed := !sameDoc(merged, local)
	if changed {
		applyDoc(merged)
		saveConfig()
	}
	mu.Unlock()
	if !sameDoc(merged, remote) {
		out, err := seal(key, merged)
		if err != nil {
			return changed, err
		}
		if err := gistWrite(sc, out); err != nil {
			return changed, err
		}
	}
	mu.Lock()
	if cfg.Sync != nil {
		cfg.Sync.Last = nowMs()
		saveConfig()
	}
	mu.Unlock()
	return changed, nil
}

func syncAndReport() {
	js("onSync", syncStatus{OK: false, Msg: "busy"})
	changed, err := runSync()
	st := syncStatus{OK: err == nil, Changed: changed}
	if err != nil {
		st.Msg = syncErr(err).Error()
	}
	mu.Lock()
	if cfg.Sync != nil {
		st.Last = cfg.Sync.Last
	}
	mu.Unlock()
	js("onSync", st)
}

// syncSoon runs a sync shortly after the last local change.
func syncSoon() {
	mu.Lock()
	on := cfg.Sync != nil
	mu.Unlock()
	if !on {
		return
	}
	timerMu.Lock()
	defer timerMu.Unlock()
	if syncTimer != nil {
		syncTimer.Stop()
	}
	syncTimer = time.AfterFunc(1200*time.Millisecond, syncAndReport)
}

func getSyncInfo() syncInfo {
	mu.Lock()
	defer mu.Unlock()
	in := syncInfo{TokenURL: tokenPage, Login: githubClientID != ""}
	if cfg.Sync == nil {
		return in
	}
	in.On, in.Last = true, cfg.Sync.Last
	in.Link = syncLink(*cfg.Sync)
	if png, err := qrcode.Encode(in.Link, qrcode.Medium, 320); err == nil {
		in.QR = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	}
	return in
}

// setupSync: input is either a sync link from another device or a GitHub token.
// result: "ok", or "exists" (a sync already lives on this account: ask for the link or replace=true).
func setupSync(input string, replace bool) (string, error) {
	input = strings.TrimSpace(input)
	loginMu.Lock()
	if input == "" && replace {
		input = loginToken // "start over" after Sign in with GitHub
	}
	loginMu.Unlock()
	if input == "" {
		return "", errors.New(tr("توکن یا لینک سینک رو وارد کن", "Enter a token or a sync link"))
	}
	if sc, ok := parseLink(input); ok {
		key, _ := syncKey(sc.Key)
		content, err := gistRead(sc)
		if err != nil {
			return "", syncErr(err)
		}
		if _, err := unseal(key, content); err != nil {
			return "", syncErr(err)
		}
		mu.Lock()
		cfg.Sync = &sc
		saveConfig()
		mu.Unlock()
		_, err = runSync()
		if err != nil {
			return "ok", syncErr(err)
		}
		return "ok", nil
	}
	token := input
	if strings.ContainsAny(token, " #/") {
		return "", errors.New(tr("این نه توکن GitHub هست نه لینک سینک", "That's neither a GitHub token nor a sync link"))
	}
	id, err := gistFind(token)
	if err != nil {
		return "", syncErr(err)
	}
	if id != "" && !replace {
		return "exists", nil
	}
	kb := make([]byte, 32)
	rand.Read(kb)
	sc := syncCfg{Token: token, Key: base64.RawURLEncoding.EncodeToString(kb)}
	mu.Lock()
	doc := localDoc()
	mu.Unlock()
	content, _ := seal(kb, doc)
	if id != "" {
		sc.Gist = id
		err = gistWrite(sc, content)
	} else {
		sc.Gist, err = gistCreate(token, content)
	}
	if err != nil {
		return "", syncErr(err)
	}
	sc.Last = nowMs()
	mu.Lock()
	cfg.Sync = &sc
	saveConfig()
	mu.Unlock()
	return "ok", nil
}

func openBrowser(u string) {
	c := exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	c.Start()
}

// ---------------------------------------------------------------- "Sign in with GitHub" (OAuth device flow, no secret needed)

// githubClientID is the public Client ID of the OAuth App (device flow enabled). Empty = button hidden.
const githubClientID = "Ov23lixhLgGnIlMRameI"

var (
	loginMu    sync.Mutex
	loginGen   int
	loginToken string
)

type deviceCode struct {
	DeviceCode string `json:"device_code"`
	UserCode   string `json:"user_code"`
	URI        string `json:"verification_uri"`
	Interval   int    `json:"interval"`
	ExpiresIn  int    `json:"expires_in"`
}

func ghForm(u string, v url.Values, out interface{}) error {
	req, _ := http.NewRequest("POST", u, strings.NewReader(v.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "SSH-Servers")
	res, err := httpc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("GitHub: %s", res.Status)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// loginGitHub shows a code, opens github.com/login/device and waits for the OK, then sets up sync.
func loginGitHub() {
	loginMu.Lock()
	loginGen++
	gen := loginGen
	loginMu.Unlock()
	alive := func() bool { loginMu.Lock(); defer loginMu.Unlock(); return gen == loginGen }

	var dc deviceCode
	if err := ghForm("https://github.com/login/device/code", url.Values{"client_id": {githubClientID}, "scope": {"gist"}}, &dc); err != nil || dc.DeviceCode == "" {
		if err == nil {
			err = errors.New("no device code")
		}
		js("onSetup", "", syncErr(err).Error())
		return
	}
	js("onDevice", dc.UserCode, dc.URI)
	openBrowser(dc.URI)
	iv := time.Duration(dc.Interval) * time.Second
	if iv < 5*time.Second {
		iv = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(dc.ExpiresIn) * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(iv)
		if !alive() {
			return
		}
		var r struct {
			Token string `json:"access_token"`
			Err   string `json:"error"`
		}
		if err := ghForm("https://github.com/login/oauth/access_token", url.Values{
			"client_id": {githubClientID}, "device_code": {dc.DeviceCode},
			"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}}, &r); err != nil {
			continue
		}
		switch {
		case r.Token != "":
			loginMu.Lock()
			loginToken = r.Token
			loginMu.Unlock()
			res, err := setupSync(r.Token, false)
			msg := ""
			if err != nil {
				msg = err.Error()
			}
			js("onSetup", res, msg)
			return
		case r.Err == "slow_down":
			iv += 5 * time.Second
		case r.Err == "authorization_pending":
		default: // access_denied, expired_token, ...
			js("onSetup", "", tr("ورود با GitHub لغو شد", "GitHub sign-in was cancelled"))
			return
		}
	}
	js("onSetup", "", tr("وقت ورود تموم شد؛ دوباره امتحان کن", "Sign-in timed out; try again"))
}

func cancelLogin() { loginMu.Lock(); loginGen++; loginMu.Unlock() }
