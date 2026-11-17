//go:build windows

package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// startShellSrv runs an SSH server whose "shell" echoes every byte back and exits on 'q'.
// It records the pty size it was given and every window-change.
func startShellSrv(t *testing.T, sizes *[]string, smu *sync.Mutex) int {
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
	t.Cleanup(func() { l.Close() })
	rec := func(s string) { smu.Lock(); *sizes = append(*sizes, s); smu.Unlock() }
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
							switch req.Type {
							case "pty-req":
								// string term, uint32 cols, rows, ...
								n := binary.BigEndian.Uint32(req.Payload)
								p := req.Payload[4+n:]
								rec(fmt.Sprintf("pty %dx%d", binary.BigEndian.Uint32(p), binary.BigEndian.Uint32(p[4:])))
								req.Reply(true, nil)
							case "window-change":
								rec(fmt.Sprintf("win %dx%d", binary.BigEndian.Uint32(req.Payload), binary.BigEndian.Uint32(req.Payload[4:])))
							case "shell":
								req.Reply(true, nil)
								go func() {
									ch.Write([]byte("welcome ✓\r\n"))
									buf := make([]byte, 1024)
									for {
										n, err := ch.Read(buf)
										if n > 0 {
											if i := strings.IndexByte(string(buf[:n]), 'q'); i >= 0 {
												ch.Write(buf[:i])
												ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
												ch.Close()
												return
											}
											ch.Write(buf[:n])
										}
										if err != nil {
											return
										}
									}
								}()
							default:
								req.Reply(req.WantReply && req.Type == "env", nil)
							}
						}
					}()
				}
			}(nc)
		}
	}()
	return l.Addr().(*net.TCPAddr).Port
}

func termOut(fw *fakeWV, tid int) (data string, states []string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	pre := fmt.Sprintf("window.onTermData(%d,\"", tid)
	for _, e := range fw.evals {
		if i := strings.Index(e, pre); i >= 0 {
			b64 := e[i+len(pre):]
			b64 = b64[:strings.IndexByte(b64, '"')]
			b, _ := base64.StdEncoding.DecodeString(b64)
			data += string(b)
		}
		if i := strings.Index(e, fmt.Sprintf("window.onTermState(%d,", tid)); i >= 0 {
			states = append(states, e[i:])
		}
	}
	return
}

func TestTerminal(t *testing.T) {
	home := t.TempDir()
	os.Setenv("USERPROFILE", home)
	fw := &fakeWV{}
	w = fw
	var sizes []string
	var smu sync.Mutex
	port := startShellSrv(t, &sizes, &smu)
	s := server{ID: "t1", Host: "127.0.0.1", Port: port, User: "u"}

	ts := &termSess{tid: 7, srvID: "t1"}
	if err := termStart(ts, s, "", 120, 40); !errors.Is(err, errNeedPass) {
		t.Fatalf("no key, no password: want need-pass, got %v", err)
	}
	if err := termStart(ts, s, "wrong", 120, 40); err == nil || !strings.Contains(err.Error(), "unable to authenticate") {
		t.Fatalf("want auth error, got %v", err)
	}
	if err := termStart(ts, s, "pw", 120, 40); err != nil {
		t.Fatal(err)
	}
	terms[7] = ts

	// keystrokes arrive in order even when sent quickly one by one
	var want strings.Builder
	for i := 0; i < 300; i++ {
		c := fmt.Sprintf("%d,", i)
		want.WriteString(c)
		ts.mu.Lock()
		in := ts.stdin
		ts.mu.Unlock()
		in <- []byte(c)
	}
	ts.mu.Lock()
	ts.sess.WindowChange(50, 160)
	ts.mu.Unlock()

	deadline := time.Now().Add(5 * time.Second)
	for {
		d, _ := termOut(fw, 7)
		if strings.Contains(d, want.String()) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("echo not complete/in order: %q", d)
		}
		time.Sleep(20 * time.Millisecond)
	}
	d, _ := termOut(fw, 7)
	if !strings.HasPrefix(d, "welcome ✓\r\n") {
		t.Fatalf("utf-8 banner broken: %q", d[:min(len(d), 40)])
	}

	ts.mu.Lock()
	ts.stdin <- []byte("bye q")
	ts.mu.Unlock()
	deadline = time.Now().Add(5 * time.Second)
	for {
		_, st := termOut(fw, 7)
		if len(st) > 0 {
			if !strings.Contains(st[0], `"closed",""`) {
				t.Fatalf("want clean close, got %s", st[0])
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("no closed state after the shell exited")
		}
		time.Sleep(20 * time.Millisecond)
	}
	smu.Lock()
	got := strings.Join(sizes, "; ")
	smu.Unlock()
	if !strings.Contains(got, "pty 120x40") || !strings.Contains(got, "win 160x50") {
		t.Fatalf("pty sizes: %s", got)
	}

	// a closed tab must not report anything after shutdown
	ts2 := &termSess{tid: 8, srvID: "t1"}
	if err := termStart(ts2, s, "pw", 80, 24); err != nil {
		t.Fatal(err)
	}
	ts2.shutdown()
	time.Sleep(200 * time.Millisecond)
	if _, st := termOut(fw, 8); len(st) > 0 {
		t.Fatalf("closed tab reported state: %v", st)
	}
}
