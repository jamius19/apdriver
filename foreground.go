//go:build windows

package main

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procSetWinEventHook          = user32.NewProc("SetWinEventHook")
	procUnhookWinEvent           = user32.NewProc("UnhookWinEvent")
	procGetMessageW              = user32.NewProc("GetMessageW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetAncestor              = user32.NewProc("GetAncestor")
	procGetLastActivePopup       = user32.NewProc("GetLastActivePopup")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procGetWindowInfo            = user32.NewProc("GetWindowInfo")

	procOpenProcess               = kernel32.NewProc("OpenProcess")
	procCloseHandle               = kernel32.NewProc("CloseHandle")
	procQueryFullProcessImageName = kernel32.NewProc("QueryFullProcessImageNameW")
)

const (
	EVENT_SYSTEM_FOREGROUND           = 0x0003
	EVENT_SYSTEM_SWITCHSTART          = 0x0014
	EVENT_SYSTEM_SWITCHEND            = 0x0015
	WINEVENT_OUTOFCONTEXT             = 0x0000
	WINEVENT_SKIPOWNPROCESS           = 0x0002
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000

	GA_ROOTOWNER     = 3
	WS_EX_TOOLWINDOW = 0x00000080
)

type rect struct{ Left, Top, Right, Bottom int32 }

// WINDOWINFO per Win32 (no padding changes; packed)
type windowInfo struct {
	CbSize          uint32
	RcWindow        rect
	RcClient        rect
	DwStyle         uint32
	DwExStyle       uint32
	DwWindowStatus  uint32
	CxWindowBorders uint32
	CyWindowBorders uint32
	AtomWindowType  uint16
	WCreatorVersion uint16
}

var (
	callbackPtr uintptr
	hook        uintptr
)

var ForegroundChanged = make(chan string, 8)

func isToolWindow(hwnd uintptr) bool {
	var wi windowInfo
	wi.CbSize = uint32(unsafe.Sizeof(wi))
	ok, _, _ := procGetWindowInfo.Call(hwnd, uintptr(unsafe.Pointer(&wi)))
	if ok == 0 {
		return false
	}
	return (wi.DwExStyle & WS_EX_TOOLWINDOW) != 0
}

func normalizeTopLevel(hwnd uintptr) uintptr {
	if hwnd == 0 {
		return 0
	}
	root, _, _ := procGetAncestor.Call(hwnd, GA_ROOTOWNER)
	if root == 0 {
		root = hwnd
	}
	for {
		last, _, _ := procGetLastActivePopup.Call(root)
		if last == 0 || last == root {
			return root
		}
		vis, _, _ := procIsWindowVisible.Call(last)
		if vis != 0 && !isToolWindow(last) {
			return last
		}
		root = last
	}
}

func getForegroundNormalized() uintptr {
	h, _, _ := procGetForegroundWindow.Call()
	return normalizeTopLevel(h)
}

func exeFromHWND(hwnd uintptr) (string, error) {
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return "", fmt.Errorf("no pid")
	}
	h, _, err := procOpenProcess.Call(PROCESS_QUERY_LIMITED_INFORMATION, 0, uintptr(pid))
	if h == 0 {
		return "", fmt.Errorf("OpenProcess: %v", err)
	}
	defer procCloseHandle.Call(h)

	buf := make([]uint16, 32768)
	sz := uint32(len(buf))
	r, _, qerr := procQueryFullProcessImageName.Call(h, 0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&sz)))
	if r == 0 {
		return "", fmt.Errorf("QueryFullProcessImageName: %v", qerr)
	}
	full := syscall.UTF16ToString(buf[:sz])
	return full, nil
}

// VOID CALLBACK WinEventProc(HWINEVENTHOOK, DWORD, HWND, LONG, LONG, DWORD, DWORD);
func winEventProc(hHook, event, hwnd, idObject, idChild, idEventThread, dwmsEventTime uintptr) uintptr {
	switch event {
	case EVENT_SYSTEM_FOREGROUND:
		// Direct click: usually final already; ignore Explorer & tool windows.
		h := normalizeTopLevel(hwnd)
		if h != 0 && !isToolWindow(h) {
			if name, err := exeFromHWND(h); err == nil && name != "" {
				select {
				case ForegroundChanged <- name:
				default:
				}
			}
		}
	case EVENT_SYSTEM_SWITCHEND:
		// Alt+Tab finished; sample the real foreground now.
		h := getForegroundNormalized()
		if h != 0 && !isToolWindow(h) {
			if name, err := exeFromHWND(h); err == nil && name != "" {
				select {
				case ForegroundChanged <- name:
				default:
				}
			}
		}
	}
	return 0
}

func startForegroundListener() (func(), error) {
	errc := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		callbackPtr = syscall.NewCallback(winEventProc)
		r, _, e := procSetWinEventHook.Call(
			EVENT_SYSTEM_FOREGROUND, EVENT_SYSTEM_SWITCHEND, // range covers both events
			0, callbackPtr, 0, 0,
			WINEVENT_OUTOFCONTEXT|WINEVENT_SKIPOWNPROCESS,
		)
		if r == 0 {
			errc <- fmt.Errorf("SetWinEventHook: %v", e)
			return
		}
		hook = r
		errc <- nil

		var msg struct {
			hwnd    uintptr
			message uint32
			wparam  uintptr
			lparam  uintptr
			time    uint32
			pt      struct{ x, y int32 }
		}
		for {
			ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if int32(ret) <= 0 {
				break
			}
		}
	}()
	if err := <-errc; err != nil {
		return nil, err
	}
	stop := func() {
		if hook != 0 {
			procUnhookWinEvent.Call(hook)
			hook = 0
		}
	}
	return stop, nil
}

func HandleForeground(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	stopListener, err := startForegroundListener()
	if err != nil {
		panic(err)
	}
	defer stopListener()

	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}
