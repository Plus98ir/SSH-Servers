//go:build windows

// SSH Servers - cards for SSH servers; opens them straight in Windows Terminal.
// Config: <exe folder>\servers.json
package main

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	webview2 "github.com/jchv/go-webview2"
)

//go:embed ui.html
var uiHTML string

//go:embed fonts/Vazirmatn-Regular.woff2
var fontRegular []byte

//go:embed fonts/Vazirmatn-Medium.woff2
var fontMedium []byte

//go:embed fonts/Vazirmatn-Bold.woff2
var fontBold []byte

const appTitle = "SSH Servers"

// ---------------------------------------------------------------- config

type server struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	User      string `json:"user"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Color     string `json:"color"`
	KeyOK     bool   `json:"keyOK"`
	Legacy    bool   `json:"legacy,omitempty"`    // router/modem: allow old ssh-rsa & kex in the ssh client
	AutoTrust bool   `json:"autoTrust,omitempty"` // accept a changed host key (rebuilt server) without asking
	U         int64  `json:"u,omitempty"`         // last edit (ms), for sync
}

type config struct {
	Servers    []server         `json:"servers"`
	SameWindow bool             `json:"sameWindow"`
	AutoClose  bool             `json:"autoClose"`
	Lang       string           `json:"lang,omitempty"`
	Deleted    map[string]int64 `json:"deleted,omitempty"` // id -> delete time (ms), for sync
	OrderU     int64            `json:"orderU,omitempty"`  // last reorder (ms)
	Sync       *syncCfg         `json:"sync,omitempty"`
}

var (
	mu      sync.Mutex
	cfg     = config{SameWindow: true, AutoClose: true, Lang: "fa"}
	cfgPath string
	w       webview2.WebView
)

func loadConfig() {
	exe, _ := os.Executable()
	cfgPath = filepath.Join(filepath.Dir(exe), "servers.json")
	if b, err := os.ReadFile(cfgPath); err == nil {
		json.Unmarshal(b, &cfg)
	}
	if cfg.Servers == nil {
		cfg.Servers = []server{}
	}
	if cfg.Lang == "" {
		cfg.Lang = "fa"
	}
}

// saveConfig writes atomically; call with mu held.
func saveConfig() error {
	b, _ := json.MarshalIndent(cfg, "", "  ")
	tmp := cfgPath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, cfgPath)
}

// find returns index and pointer; call with mu held.
func find(id string) (int, *server) {
	for i := range cfg.Servers {
		if cfg.Servers[i].ID == id {
			return i, &cfg.Servers[i]
		}
	}
	return -1, nil
}

// get returns a copy of a server.
func get(id string) (server, bool) {
	mu.Lock()
	defer mu.Unlock()
	_, p := find(id)
	if p == nil {
		return server{}, false
	}
	return *p, true
}

func update(id string, f func(*server)) {
	mu.Lock()
	defer mu.Unlock()
	if _, p := find(id); p != nil {
		f(p)
		saveConfig()
	}
}

func tr(fa, en string) string {
	mu.Lock()
	l := cfg.Lang
	mu.Unlock()
	if l == "en" {
		return en
	}
	return fa
}

// ---------------------------------------------------------------- JS bridge

func js(fn string, args ...interface{}) {
	parts := make([]string, len(args))
	for i, a := range args {
		b, _ := json.Marshal(a)
		parts[i] = string(b)
	}
	code := "window." + fn + "&&window." + fn + "(" + strings.Join(parts, ",") + ")"
	w.Dispatch(func() { w.Eval(code) })
}

func toastErr(msg string) { js("onToast", msg, "err") }

func uiPage() string {
	b64 := base64.StdEncoding.EncodeToString
	fonts := fmt.Sprintf(
		"@font-face{font-family:Vazirmatn;font-weight:400;src:url(data:font/woff2;base64,%s) format('woff2')}"+
			"@font-face{font-family:Vazirmatn;font-weight:500;src:url(data:font/woff2;base64,%s) format('woff2')}"+
			"@font-face{font-family:Vazirmatn;font-weight:700;src:url(data:font/woff2;base64,%s) format('woff2')}",
		b64(fontRegular), b64(fontMedium), b64(fontBold))
	return strings.Replace(uiHTML, "/*FONTS*/", fonts, 1)
}

// ---------------------------------------------------------------- main

var (
	user32       = syscall.NewLazyDLL("user32.dll")
	pMessageBoxW = user32.NewProc("MessageBoxW")
)

func msgBox(text string) {
	t, _ := syscall.UTF16PtrFromString(text)
	c, _ := syscall.UTF16PtrFromString(appTitle)
	pMessageBoxW.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), 0x10)
}

func main() {
	loadConfig()
	dataDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "SSHLauncher")
	w = webview2.NewWithOptions(webview2.WebViewOptions{
		DataPath:  dataDir,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  appTitle,
			Width:  800,
			Height: 576,
			IconId: 2,
			Center: true,
		},
	})
	if w == nil {
		msgBox("برای اجرای این برنامه «Microsoft Edge WebView2 Runtime» لازمه.\nاز سایت مایکروسافت نصبش کن و دوباره امتحان کن.\n\nThis app needs the Microsoft Edge WebView2 Runtime. Please install it and try again.")
		return
	}
	defer w.Destroy()
	hwnd := uintptr(w.Window())
	makeFrameless(hwnd)
	bindAll(hwnd)
	w.SetHtml(uiPage())
	w.Run()
}
