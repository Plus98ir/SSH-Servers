//go:build windows

// Update check + self-update from GitHub releases (tags v*; the Android app uses android-v*).
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	appVersion  = "1.1.0"
	releasesAPI = "https://api.github.com/repos/Plus98ir/SSH-Servers/releases?per_page=30"
	exeAsset    = "SSH.Servers.exe"
)

type updateInfo struct {
	Current string `json:"current"`
	Latest  string `json:"latest"`
	Newer   bool   `json:"newer"`
	Notes   string `json:"notes,omitempty"`
	URL     string `json:"url,omitempty"` // exe download
	Page    string `json:"page,omitempty"`
	Err     string `json:"err,omitempty"`
}

// verParts "1.2.3" -> [1 2 3]
func verParts(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	out := []int{}
	for _, p := range strings.Split(v, ".") {
		n, _ := strconv.Atoi(strings.TrimFunc(p, func(r rune) bool { return r < '0' || r > '9' }))
		out = append(out, n)
	}
	return out
}

func newerVer(a, b string) bool { // a > b
	x, y := verParts(a), verParts(b)
	for i := 0; i < len(x) || i < len(y); i++ {
		var p, q int
		if i < len(x) {
			p = x[i]
		}
		if i < len(y) {
			q = y[i]
		}
		if p != q {
			return p > q
		}
	}
	return false
}

func checkUpdate() updateInfo {
	in := updateInfo{Current: appVersion}
	req, _ := http.NewRequest("GET", releasesAPI, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "SSH-Servers")
	res, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		in.Err = err.Error()
		return in
	}
	defer res.Body.Close()
	var rels []struct {
		Tag        string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
		Body       string `json:"body"`
		Page       string `json:"html_url"`
		Assets     []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(res.Body).Decode(&rels); err != nil {
		in.Err = err.Error()
		return in
	}
	for _, r := range rels {
		if r.Draft || r.Prerelease || !strings.HasPrefix(r.Tag, "v") {
			continue
		}
		in.Latest = strings.TrimPrefix(r.Tag, "v")
		in.Notes, in.Page = r.Body, r.Page
		for _, a := range r.Assets {
			if a.Name == exeAsset {
				in.URL = a.URL
			}
		}
		in.Newer = newerVer(in.Latest, appVersion) && in.URL != ""
		break
	}
	return in
}

// selfUpdate downloads the new exe, swaps it in (a running exe can be renamed) and restarts.
func selfUpdate(url string) error {
	if !strings.HasPrefix(url, "https://github.com/Plus98ir/SSH-Servers/") {
		return errors.New("bad url")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	res, err := (&http.Client{Timeout: 5 * time.Minute}).Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("download: %s", res.Status)
	}
	tmp := exe + ".new"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, res.Body)
	f.Close()
	if err != nil || n < 1<<20 {
		os.Remove(tmp)
		if err == nil {
			err = errors.New("download too small")
		}
		return err
	}
	old := exe + ".old"
	os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, exe); err != nil {
		os.Rename(old, exe)
		return err
	}
	c := exec.Command(exe)
	c.Dir = filepath.Dir(exe)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: false}
	if err := c.Start(); err != nil {
		return err
	}
	w.Dispatch(closeWindow)
	return nil
}

// cleanupOld removes the previous exe left by a self-update.
func cleanupOld() {
	if exe, err := os.Executable(); err == nil {
		os.Remove(exe + ".old")
	}
}

// autoCheck runs at start: asks GitHub at most every 12 hours, otherwise uses the last answer.
func autoCheck() {
	mu.Lock()
	last, cached := cfg.UpdateCheck, cfg.Update
	mu.Unlock()
	if time.Since(time.UnixMilli(last)) < 12*time.Hour {
		if cached != nil && newerVer(cached.Latest, appVersion) && cached.URL != "" {
			c := *cached
			c.Current, c.Newer = appVersion, true
			time.Sleep(1500 * time.Millisecond)
			js("onUpdate", c, false)
		}
		return
	}
	time.Sleep(3 * time.Second)
	in := checkUpdate()
	if in.Err != "" {
		return
	}
	mu.Lock()
	cfg.UpdateCheck, cfg.Update = nowMs(), &in
	saveConfig()
	mu.Unlock()
	if in.Newer {
		js("onUpdate", in, false)
	}
}
