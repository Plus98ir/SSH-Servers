//go:build windows

package main

import (
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func wtPath() string {
	if p, err := exec.LookPath("wt.exe"); err == nil {
		return p
	}
	p := filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "WindowsApps", "wt.exe")
	if fileExists(p) {
		return p
	}
	return ""
}

// sshArgs is the ssh command line for one server.
func sshArgs(s server) []string {
	a := []string{"ssh", "-o", "StrictHostKeyChecking=accept-new"}
	if s.Legacy {
		a = append(a, "-o", "HostKeyAlgorithms=+ssh-rsa", "-o", "PubkeyAcceptedAlgorithms=+ssh-rsa",
			"-o", "KexAlgorithms=+diffie-hellman-group14-sha1,diffie-hellman-group1-sha1")
	}
	if s.Port != 0 && s.Port != 22 {
		a = append(a, "-p", strconv.Itoa(s.Port))
	}
	if s.User != "" {
		a = append(a, s.User+"@"+s.Host)
	} else {
		a = append(a, s.Host)
	}
	return a
}

// wtTitle keeps ';' out of titles (Windows Terminal uses it to separate commands).
func wtTitle(s server) string {
	n := strings.NewReplacer(";", ",", "\"", "'").Replace(s.Name)
	if n == "" {
		n = s.Host
	}
	return n
}

func paneArgs(s server) []string {
	a := []string{"--title", wtTitle(s), "--suppressApplicationTitle"}
	if s.Color != "" {
		a = append(a, "--tabColor", s.Color)
	}
	return append(a, sshArgs(s)...)
}

func startDetached(name string, args ...string) error {
	c := exec.Command(name, args...)
	return c.Start()
}

// fallback without Windows Terminal: one classic console window per server.
func openClassic(list []server) error {
	for _, s := range list {
		args := append([]string{"/c", "start", wtTitle(s)}, sshArgs(s)...)
		if err := startDetached("cmd.exe", args...); err != nil {
			return err
		}
	}
	return nil
}

func launchOne(s server, sameWindow bool) error {
	wt := wtPath()
	if wt == "" {
		return openClassic([]server{s})
	}
	win := "new"
	if sameWindow {
		win = "0"
	}
	args := append([]string{"-w", win, "nt"}, paneArgs(s)...)
	return startDetached(wt, args...)
}

func launchTabs(list []server) error {
	if len(list) == 0 {
		return errors.New("no servers")
	}
	wt := wtPath()
	if wt == "" {
		return openClassic(list)
	}
	args := []string{"-w", "new"}
	for i, s := range list {
		if i > 0 {
			args = append(args, ";")
		}
		args = append(args, "nt")
		args = append(args, paneArgs(s)...)
	}
	args = append(args, ";", "ft", "-t", "0")
	return startDetached(wt, args...)
}

func frac(f float64) string { return strconv.FormatFloat(f, 'f', 3, 64) }

// launchSplit opens the servers as a grid of panes in one maximized Terminal tab.
func launchSplit(list []server) error {
	n := len(list)
	if n == 0 {
		return errors.New("no servers")
	}
	wt := wtPath()
	if wt == "" {
		return openClassic(list)
	}
	cols := int(math.Ceil(math.Sqrt(float64(n))))
	// spread servers over columns, earlier columns get the extra ones
	grid := make([][]server, cols)
	for i, s := range list {
		c := i * cols / n
		grid[c] = append(grid[c], s)
	}
	args := []string{"-w", "new", "--maximized", "nt"}
	args = append(args, paneArgs(grid[0][0])...)
	// columns: each split takes its share of the remaining space on the right
	for j := 1; j < cols; j++ {
		left := cols - j + 1
		args = append(args, ";", "sp", "-V", "-s", frac(float64(left-1)/float64(left)))
		args = append(args, paneArgs(grid[j][0])...)
	}
	// rows inside each column, top to bottom
	args = append(args, ";", "mf", "first")
	for j := 0; j < cols; j++ {
		rows := len(grid[j])
		for r := 1; r < rows; r++ {
			left := rows - r + 1
			args = append(args, ";", "sp", "-H", "-s", frac(float64(left-1)/float64(left)))
			args = append(args, paneArgs(grid[j][r])...)
		}
		if j < cols-1 {
			args = append(args, ";", "mf", "nextInOrder")
		}
	}
	args = append(args, ";", "mf", "first")
	return startDetached(wt, args...)
}
