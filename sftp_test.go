//go:build windows

package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	webview2 "github.com/jchv/go-webview2"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type fakeWV struct {
	webview2.WebView
	mu    sync.Mutex
	evals []string
}

func (f *fakeWV) Dispatch(fn func()) { fn() }
func (f *fakeWV) Eval(s string)      { f.mu.Lock(); f.evals = append(f.evals, s); f.mu.Unlock() }

func startSrv(t *testing.T) int {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer, _ := ssh.NewSignerFromKey(priv)
	conf := &ssh.ServerConfig{PasswordCallback: func(c ssh.ConnMetadata, p []byte) (*ssh.Permissions, error) {
		if string(p) == "pw" {
			return nil, nil
		}
		return nil, errors.New("bad")
	}}
	conf.AddHostKey(signer)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	h := sftp.InMemHandler()
	go func() {
		for {
			nc, err := l.Accept()
			if err != nil {
				return
			}
			go func(nc net.Conn) {
				_, chans, reqs, err := ssh.NewServerConn(nc, conf)
				if err != nil {
					return
				}
				go ssh.DiscardRequests(reqs)
				for nch := range chans {
					ch, in, err := nch.Accept()
					if err != nil {
						continue
					}
					go func() {
						for req := range in {
							ok := req.Type == "subsystem" && len(req.Payload) > 4 && string(req.Payload[4:]) == "sftp"
							req.Reply(ok, nil)
							if ok {
								rs := sftp.NewRequestServer(ch, h)
								rs.Serve()
								ch.Close()
							}
						}
					}()
				}
			}(nc)
		}
	}()
	return l.Addr().(*net.TCPAddr).Port
}

func TestSFTP(t *testing.T) {
	home := t.TempDir()
	os.Setenv("USERPROFILE", home)
	os.MkdirAll(filepath.Join(home, "Downloads"), 0o755)
	fw := &fakeWV{}
	w = fw
	port := startSrv(t)
	s := server{ID: "t1", Host: "127.0.0.1", Port: port, User: "u"}
	f := &fmSess{srv: s}
	fmSessions["t1"] = f

	f.mu.Lock()
	err := f.dial()
	f.mu.Unlock()
	if !errors.Is(err, errNeedPass) {
		t.Fatalf("want need-pass, got %v", err)
	}
	f.pass = "wrong"
	f.mu.Lock()
	err = f.dial()
	f.mu.Unlock()
	if err == nil || !strings.Contains(err.Error(), "unable to authenticate") {
		t.Fatalf("want auth error, got %v", err)
	}
	f.pass = "pw"
	f.mu.Lock()
	err = f.dial()
	f.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("home:", f.home)

	must := func(op string, a ...string) interface{} {
		t.Helper()
		d, err := f.fsOp(op, a)
		if err != nil {
			t.Fatalf("%s %v: %v", op, a, err)
		}
		return d
	}
	must("mkdir", "/a")
	must("touch", "/a/x.txt")
	if _, err := f.fsOp("touch", []string{"/a/x.txt"}); err == nil {
		t.Fatal("touch existing should fail")
	}
	must("write", "/a/x.txt", "hello\nwörld\n", "1")
	d := must("read", "/a/x.txt").(map[string]interface{})
	if d["text"] != "hello\nwörld\n" || d["crlf"] != true {
		t.Fatalf("read: %#v", d)
	}
	// binary detection
	must("write", "/a/bin", "ab\x00cd", "0")
	if _, err := f.fsOp("read", []string{"/a/bin"}); err == nil || err.Error() != "binary" {
		t.Fatalf("want binary, got %v", err)
	}

	// upload 2.5 MB in chunks into a new sub folder
	payload := make([]byte, 2_500_000)
	rand.Read(payload)
	x := &xfer{id: "u1", sid: "t1", kind: "up", name: "big.bin", total: int64(len(payload)), ch: make(chan []byte, 8)}
	xfMu.Lock()
	xfers["u1"] = x
	xfMu.Unlock()
	go f.upload(x, "/a/sub/big.bin", x.ch)
	for off := 0; off < len(payload); off += 1 << 20 {
		end := off + 1<<20
		if end > len(payload) {
			end = len(payload)
		}
		x.ch <- payload[off:end]
	}
	close(x.ch)
	waitGone(t, "u1")

	l := must("list", "/a").(map[string]interface{})
	items := l["items"].([]fsItem)
	names := []string{}
	for _, it := range items {
		names = append(names, it.N)
	}
	t.Log("list /a:", names)
	if len(items) != 3 {
		t.Fatalf("want 3 items, got %v", names)
	}

	// download the folder
	f.download([]string{"/a"})
	got, err := os.ReadFile(filepath.Join(home, "Downloads", "a", "sub", "big.bin"))
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("download mismatch: %v len=%d", err, len(got))
	}
	txt, _ := os.ReadFile(filepath.Join(home, "Downloads", "a", "x.txt"))
	if string(txt) != "hello\r\nwörld\r\n" {
		t.Fatalf("x.txt = %q", txt)
	}
	// second download of same file → unique name
	f.download([]string{"/a/x.txt"})
	if !fileExists(filepath.Join(home, "Downloads", "x.txt")) {
		t.Fatal("single download missing")
	}
	f.download([]string{"/a/x.txt"})
	if !fileExists(filepath.Join(home, "Downloads", "x (1).txt")) {
		t.Fatal("unique name missing")
	}

	// connection drops underneath → transparent reconnect
	f.mu.Lock()
	f.sc.Close()
	f.mu.Unlock()
	time.Sleep(100 * time.Millisecond)
	must("rename", "/a/x.txt", "/a/y.txt")
	must("chmod", "", "600", "/a/y.txt")
	must("remove", "/a")
	l = must("list", "/").(map[string]interface{})
	if n := len(l["items"].([]fsItem)); n != 0 {
		t.Fatalf("root not empty: %d", n)
	}
	if _, err := f.fsOp("list", []string{"/nope"}); err == nil {
		t.Fatal("want error")
	} else {
		t.Log("missing dir msg:", fsErr(err))
	}
	fw.mu.Lock()
	n := len(fw.evals)
	last := fw.evals[n-1]
	fw.mu.Unlock()
	t.Log("js events:", n, "last:", last[:min(len(last), 160)])
	if winName("CON.txt") != "_CON.txt" || winName(`a:b?.`) != "a_b_" {
		t.Fatal("winName", winName("CON.txt"), winName(`a:b?.`))
	}
}

func waitGone(t *testing.T, id string) {
	for i := 0; i < 200; i++ {
		xfMu.Lock()
		_, ok := xfers[id]
		xfMu.Unlock()
		if !ok {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("transfer did not finish")
}
