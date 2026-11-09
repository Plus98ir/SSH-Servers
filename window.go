//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	dwmapi                 = syscall.NewLazyDLL("dwmapi.dll")
	pDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	pGetWindowLongPtrW     = user32.NewProc("GetWindowLongPtrW")
	pSetWindowLongPtrW     = user32.NewProc("SetWindowLongPtrW")
	pCallWindowProcW       = user32.NewProc("CallWindowProcW")
	pSetWindowPos          = user32.NewProc("SetWindowPos")
	pGetWindowRect         = user32.NewProc("GetWindowRect")
	pShowWindow            = user32.NewProc("ShowWindow")
	pPostMessageW          = user32.NewProc("PostMessageW")
	pGetDpiForWindow       = user32.NewProc("GetDpiForWindow")
	pMonitorFromWindow     = user32.NewProc("MonitorFromWindow")
	pGetMonitorInfoW       = user32.NewProc("GetMonitorInfoW")
)

const (
	gwlStyle    = ^uintptr(15) // -16
	gwlpWndProc = ^uintptr(3)  // -4

	wsThickFrame  = 0x00040000
	wsMaximizeBox = 0x00010000

	wmNCCalcSize = 0x0083
	wmClose      = 0x0010

	swpNoSize       = 0x0001
	swpNoMove       = 0x0002
	swpNoZOrder     = 0x0004
	swpNoActivate   = 0x0010
	swpFrameChanged = 0x0020
)

type rect struct{ L, T, R, B int32 }

type monitorInfo struct {
	Size    uint32
	Monitor rect
	Work    rect
	Flags   uint32
}

var (
	oldWndProc uintptr
	mainHwnd   uintptr
	firstFit   = true
	dragOrigin rect
)

func wndProc(hwnd, msg, wp, lp uintptr) uintptr {
	if msg == wmNCCalcSize && wp != 0 {
		return 0 // whole window is client area: no title bar or border, keeps DWM shadow
	}
	r, _, _ := pCallWindowProcW.Call(oldWndProc, hwnd, msg, wp, lp)
	return r
}

func makeFrameless(hwnd uintptr) {
	mainHwnd = hwnd
	style, _, _ := pGetWindowLongPtrW.Call(hwnd, gwlStyle)
	style &^= wsThickFrame | wsMaximizeBox
	pSetWindowLongPtrW.Call(hwnd, gwlStyle, style)
	oldWndProc, _, _ = pSetWindowLongPtrW.Call(hwnd, gwlpWndProc, syscall.NewCallback(wndProc))
	pref := uint32(2) // DWMWCP_ROUND (Windows 11)
	pDwmSetWindowAttribute.Call(hwnd, 33, uintptr(unsafe.Pointer(&pref)), 4)
	dark := uint32(1) // DWMWA_USE_IMMERSIVE_DARK_MODE
	pDwmSetWindowAttribute.Call(hwnd, 20, uintptr(unsafe.Pointer(&dark)), 4)
	pSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoZOrder|swpNoActivate|swpFrameChanged)
}

func windowRect() rect {
	var r rect
	pGetWindowRect.Call(mainHwnd, uintptr(unsafe.Pointer(&r)))
	return r
}

func workArea() rect {
	mon, _, _ := pMonitorFromWindow.Call(mainHwnd, 2) // MONITOR_DEFAULTTONEAREST
	mi := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	pGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	return mi.Work
}

func dpiScale(dpr float64) float64 {
	if d, _, _ := pGetDpiForWindow.Call(mainHwnd); d > 0 {
		return float64(d) / 96
	}
	if dpr > 0 {
		return dpr
	}
	return 1
}

// fitWindow resizes to the page's size (CSS px), keeps it on screen, centers it the first time.
func fitWindow(cw, ch int, dpr float64) {
	s := dpiScale(dpr)
	wa := workArea()
	ww, wh := int32(float64(cw)*s+0.5), int32(float64(ch)*s+0.5)
	if wh > wa.B-wa.T {
		wh = wa.B - wa.T
	}
	if ww > wa.R-wa.L {
		ww = wa.R - wa.L
	}
	r := windowRect()
	x, y := r.L, r.T
	if firstFit {
		firstFit = false
		x = wa.L + (wa.R-wa.L-ww)/2
		y = wa.T + (wa.B-wa.T-wh)/2
	}
	if y+wh > wa.B {
		y = wa.B - wh
	}
	if y < wa.T {
		y = wa.T
	}
	if x+ww > wa.R {
		x = wa.R - ww
	}
	if x < wa.L {
		x = wa.L
	}
	pSetWindowPos.Call(mainHwnd, 0, uintptr(x), uintptr(y), uintptr(ww), uintptr(wh), swpNoZOrder|swpNoActivate)
}

func dragStart() { dragOrigin = windowRect() }

func dragMove(dx, dy, dpr float64) {
	s := dpr
	if s <= 0 {
		s = 1
	}
	x := dragOrigin.L + int32(dx*s)
	y := dragOrigin.T + int32(dy*s)
	pSetWindowPos.Call(mainHwnd, 0, uintptr(x), uintptr(y), 0, 0, swpNoSize|swpNoZOrder|swpNoActivate)
}

func minimizeWindow() { pShowWindow.Call(mainHwnd, 6) } // SW_MINIMIZE

func closeWindow() { pPostMessageW.Call(mainHwnd, wmClose, 0, 0) }
