//go:build tray && windows

// The Windows notification-area icon: left click shows the window, right
// click offers Open and Quit, and the tooltip shows gateway health. It runs
// its own hidden window and message loop on a dedicated OS thread, so the
// webview's loop on the main thread is untouched. Pure syscalls through
// x/sys/windows; no new dependency (spec D21).
package main

import (
	_ "embed"
	"errors"
	"fmt"
	"log"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

//go:embed trayicon_windows.png
var trayIconPNG []byte

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	shell32                      = windows.NewLazySystemDLL("shell32.dll")
	procRegisterClassExW         = user32.NewProc("RegisterClassExW")
	procCreateWindowExW          = user32.NewProc("CreateWindowExW")
	procDefWindowProcW           = user32.NewProc("DefWindowProcW")
	procDestroyWindow            = user32.NewProc("DestroyWindow")
	procPostQuitMessage          = user32.NewProc("PostQuitMessage")
	procGetMessageW              = user32.NewProc("GetMessageW")
	procTranslateMessage         = user32.NewProc("TranslateMessage")
	procDispatchMessageW         = user32.NewProc("DispatchMessageW")
	procPostMessageW             = user32.NewProc("PostMessageW")
	procCreatePopupMenu          = user32.NewProc("CreatePopupMenu")
	procAppendMenuW              = user32.NewProc("AppendMenuW")
	procTrackPopupMenu           = user32.NewProc("TrackPopupMenu")
	procDestroyMenu              = user32.NewProc("DestroyMenu")
	procGetCursorPos             = user32.NewProc("GetCursorPos")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procShowWindow               = user32.NewProc("ShowWindow")
	procRegisterWindowMessageW   = user32.NewProc("RegisterWindowMessageW")
	procCreateIconFromResourceEx = user32.NewProc("CreateIconFromResourceEx")
	procLoadIconW                = user32.NewProc("LoadIconW")
	procShellNotifyIconW         = shell32.NewProc("Shell_NotifyIconW")
)

const (
	wmNull         = 0x0000
	wmClose        = 0x0010
	wmDestroy      = 0x0002
	wmContextMenu  = 0x007B
	wmLButtonUp    = 0x0202
	wmRButtonUp    = 0x0205
	wmApp          = 0x8000
	wmTrayCallback = wmApp + 1
	nimAdd         = 0
	nimModify      = 1
	nimDelete      = 2
	nifMessage     = 0x1
	nifIcon        = 0x2
	nifTip         = 0x4
	mfString       = 0x0
	tpmRightButton = 0x0002
	tpmReturnCmd   = 0x0100
	swRestore      = 9
	idiApplication = 32512
)

// notifyIconData is NOTIFYICONDATAW.
type notifyIconData struct {
	CbSize           uint32
	HWnd             windows.HWND
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            windows.Handle
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         windows.GUID
	HBalloonIcon     windows.Handle
}

// wndClassEx is WNDCLASSEXW.
type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

type point struct{ X, Y int32 }

// msg is MSG.
type msg struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
	Private uint32
}

var trayIcon struct {
	mu             sync.Mutex
	hwnd           uintptr // our hidden window
	main           uintptr // the webview window
	data           notifyIconData
	added          bool
	taskbarCreated uint32
}

// The window class is registered once per process: a second registration
// fails, and every windows.NewCallback uses up one of a fixed number of slots.
var (
	trayClassOnce sync.Once
	trayClassErr  error
)

// installStatusItem adds the notification-area icon for the webview window.
func installStatusItem(win unsafe.Pointer) {
	trayIcon.mu.Lock()
	trayIcon.main = uintptr(win)
	trayIcon.mu.Unlock()
	ready := make(chan struct{})
	go runTrayLoop(ready)
	<-ready
}

func runTrayLoop(ready chan<- struct{}) {
	// The window belongs to this thread and only this thread's GetMessage
	// sees its messages; the goroutine ends with the loop and takes the
	// locked thread with it.
	runtime.LockOSThread()
	hwnd, err := createTrayWindow()
	if err != nil {
		log.Printf("jumpgate: tray icon: %v", err)
		close(ready)
		return
	}
	trayIcon.mu.Lock()
	trayIcon.hwnd = hwnd
	trayIcon.mu.Unlock()
	addTrayIcon(hwnd)
	close(ready)
	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// registerTrayClass registers the hidden window's class, once.
func registerTrayClass(inst windows.Handle, className *uint16) error {
	trayClassOnce.Do(func() {
		wc := wndClassEx{LpfnWndProc: windows.NewCallback(trayWndProc), HInstance: inst, LpszClassName: className}
		wc.CbSize = uint32(unsafe.Sizeof(wc))
		if r, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 && !errors.Is(err, windows.ERROR_CLASS_ALREADY_EXISTS) {
			trayClassErr = fmt.Errorf("RegisterClassExW: %w", err)
		}
	})
	return trayClassErr
}

// createTrayWindow makes a hidden top-level window to receive the icon's
// messages. Not a message-only window: those miss the TaskbarCreated
// broadcast that says Explorer restarted and the icon must be re-added.
func createTrayWindow() (uintptr, error) {
	className, _ := windows.UTF16PtrFromString("JumpgateTray")
	var inst windows.Handle
	if err := windows.GetModuleHandleEx(0, nil, &inst); err != nil {
		return 0, err
	}
	if err := registerTrayClass(inst, className); err != nil {
		return 0, err
	}
	title, _ := windows.UTF16PtrFromString("Jumpgate")
	hwnd, _, err := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)),
		0, 0, 0, 0, 0, 0, 0, uintptr(inst), 0)
	if hwnd == 0 {
		return 0, fmt.Errorf("CreateWindowExW: %w", err)
	}
	tb, _ := windows.UTF16PtrFromString("TaskbarCreated")
	r, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(tb)))
	trayIcon.mu.Lock()
	trayIcon.taskbarCreated = uint32(r)
	trayIcon.mu.Unlock()
	return hwnd, nil
}

// loadTrayIcon builds the icon from the embedded PNG (Vista and later accept
// PNG icon data), falling back to the stock application icon.
func loadTrayIcon() windows.Handle {
	if len(trayIconPNG) > 0 {
		h, _, _ := procCreateIconFromResourceEx.Call(uintptr(unsafe.Pointer(&trayIconPNG[0])), uintptr(len(trayIconPNG)), 1, 0x00030000, 32, 32, 0)
		if h != 0 {
			return windows.Handle(h)
		}
	}
	h, _, _ := procLoadIconW.Call(0, idiApplication)
	return windows.Handle(h)
}

func setTip(d *notifyIconData, s string) {
	u, _ := windows.UTF16FromString(s)
	n := copy(d.SzTip[:len(d.SzTip)-1], u)
	d.SzTip[n] = 0
}

func addTrayIcon(hwnd uintptr) {
	trayIcon.mu.Lock()
	defer trayIcon.mu.Unlock()
	d := &trayIcon.data
	d.CbSize = uint32(unsafe.Sizeof(*d))
	d.HWnd = windows.HWND(hwnd)
	d.UID = 1
	d.UFlags = nifMessage | nifIcon | nifTip
	d.UCallbackMessage = wmTrayCallback
	if d.HIcon == 0 {
		d.HIcon = loadTrayIcon()
	}
	if d.SzTip[0] == 0 {
		setTip(d, healthTooltip(healthOff))
	}
	r, _, _ := procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(d)))
	trayIcon.added = r != 0
}

// setHealth updates the tooltip. Shell_NotifyIconW may be called from any
// thread.
func setHealth(k healthKind) {
	trayIcon.mu.Lock()
	defer trayIcon.mu.Unlock()
	setTip(&trayIcon.data, healthTooltip(k))
	if trayIcon.added {
		procShellNotifyIconW.Call(nimModify, uintptr(unsafe.Pointer(&trayIcon.data)))
	}
}

// removeStatusItem deletes the icon and ends the tray thread's loop, so no
// ghost icon stays in the notification area after the app quits.
func removeStatusItem() {
	trayIcon.mu.Lock()
	defer trayIcon.mu.Unlock()
	if trayIcon.added {
		procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&trayIcon.data)))
		trayIcon.added = false
	}
	if trayIcon.hwnd != 0 {
		procPostMessageW.Call(trayIcon.hwnd, wmClose, 0, 0)
		trayIcon.hwnd = 0
	}
}

func trayWndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	switch uint32(message) {
	case wmTrayCallback:
		switch uint32(lParam) & 0xFFFF {
		case wmLButtonUp:
			runTrayAction(trayOpen)
		case wmRButtonUp, wmContextMenu:
			runTrayAction(trayActionFor(showTrayMenu(hwnd)))
		}
		return 0
	case wmClose:
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	trayIcon.mu.Lock()
	tb := trayIcon.taskbarCreated
	trayIcon.mu.Unlock()
	if tb != 0 && uint32(message) == tb {
		addTrayIcon(hwnd)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return r
}

// showTrayMenu shows Open / Quit at the cursor and returns the chosen id.
func showTrayMenu(hwnd uintptr) int {
	menu, _, _ := procCreatePopupMenu.Call()
	defer procDestroyMenu.Call(menu)
	open, _ := windows.UTF16PtrFromString("Open Jumpgate")
	quit, _ := windows.UTF16PtrFromString("Quit Jumpgate")
	procAppendMenuW.Call(menu, mfString, 1, uintptr(unsafe.Pointer(open)))
	procAppendMenuW.Call(menu, mfString, 2, uintptr(unsafe.Pointer(quit)))
	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	// Without this the menu stays up when the user clicks elsewhere.
	procSetForegroundWindow.Call(hwnd)
	cmd, _, _ := procTrackPopupMenu.Call(menu, tpmReturnCmd|tpmRightButton, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
	// And without this a second right click can open no menu: the documented
	// companion to SetForegroundWindow above (TrackPopupMenu remarks).
	procPostMessageW.Call(hwnd, wmNull, 0, 0)
	return int(cmd)
}

func runTrayAction(a trayAction) {
	switch a {
	case trayOpen:
		trayIcon.mu.Lock()
		main := trayIcon.main
		trayIcon.mu.Unlock()
		if main != 0 {
			procShowWindow.Call(main, swRestore)
			procSetForegroundWindow.Call(main)
		}
	case trayQuit:
		requestQuit()
	}
}
