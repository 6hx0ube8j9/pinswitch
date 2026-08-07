//go:build windows

package main

import (
	"encoding/binary"
	"syscall"
	"time"
	"unsafe"
)

const (
	WM_NULL            = 0x0000
	WM_CLOSE           = 0x0010
	WM_COMMAND         = 0x0111
	WM_HOTKEY          = 0x0312
	WM_USER            = 0x0400
	WM_SETTINGCHANGE   = 0x001A
	WM_POWERBROADCAST  = 0x0218
	WM_TRAYICON        = WM_USER + 100

	PBT_APMRESUMEAUTOMATIC = 0x0012
	PBT_APMRESUMESUSPEND   = 0x0007

	SMTO_ABORTIFHUNG   = 0x0002
	SM_CXSMICON        = 49
	SM_CYSMICON        = 50

	NIM_ADD            = 0x00000000
	NIM_MODIFY         = 0x00000001
	NIM_DELETE         = 0x00000002
	NIF_MESSAGE        = 0x00000001
	NIF_ICON           = 0x00000002
	NIF_TIP            = 0x00000004

	WM_LBUTTONUP       = 0x0202
	WM_RBUTTONUP       = 0x0205

	MF_STRING          = 0x00000000
	MF_GRAYED          = 0x00000001
	MF_CHECKED         = 0x00000008
	MF_SEPARATOR       = 0x00000800

	TPM_LEFTALIGN      = 0x0000
	TPM_BOTTOMALIGN    = 0x0020
	TPM_RIGHTBUTTON    = 0x0002
)

type Msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      Point
}

type Point struct {
	X, Y int32
}

type WndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor        uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type NotifyIconData struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
}

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")

	procRegisterHotKey           = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey         = user32.NewProc("UnregisterHotKey")
	procGetMessage               = user32.NewProc("GetMessageW")
	procTranslateMessage         = user32.NewProc("TranslateMessage")
	procDispatchMessage          = user32.NewProc("DispatchMessageW")
	procDefWindowProc            = user32.NewProc("DefWindowProcW")
	procRegisterClassEx          = user32.NewProc("RegisterClassExW")
	procCreateWindowEx           = user32.NewProc("CreateWindowExW")
	procDestroyWindow            = user32.NewProc("DestroyWindow")
	procPostQuitMessage          = user32.NewProc("PostQuitMessage")
	procFindWindowExW            = user32.NewProc("FindWindowExW")
	procPostMessageW             = user32.NewProc("PostMessageW")
	procGetAsyncKeyState           = user32.NewProc("GetAsyncKeyState")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procSendMessageTimeoutW      = user32.NewProc("SendMessageTimeoutW")
	procRegisterWindowMessageW   = user32.NewProc("RegisterWindowMessageW")
	procGetSystemMetrics         = user32.NewProc("GetSystemMetrics")
	procMessageBoxW              = user32.NewProc("MessageBoxW")

	procShellNotifyIconW         = shell32.NewProc("Shell_NotifyIconW")
	procCreatePopupMenu          = user32.NewProc("CreatePopupMenu")
	procAppendMenuW              = user32.NewProc("AppendMenuW")
	procTrackPopupMenu           = user32.NewProc("TrackPopupMenu")
	procGetCursorPos             = user32.NewProc("GetCursorPos")
	procDestroyMenu              = user32.NewProc("DestroyMenu")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procCreateIconFromResourceEx = user32.NewProc("CreateIconFromResourceEx")
	procDestroyIcon              = user32.NewProc("DestroyIcon")

	procCreateMutexW = kernel32.NewProc("CreateMutexW")
	procCloseHandle  = kernel32.NewProc("CloseHandle")

	globalWndProcCallback uintptr
)

var imeRefreshChan = make(chan struct{}, 1)

func init() {
	go startIMEMonitorLoop()
}

func AsyncRefreshActiveWindowIME() {
	select {
	case imeRefreshChan <- struct{}{}:
	default:
	}
}

func startIMEMonitorLoop() {
	strPtr, _ := syscall.UTF16PtrFromString("Control Panel\\Input Method")
	strVal := uintptr(unsafe.Pointer(strPtr))

	for range imeRefreshChan {
		time.Sleep(300 * time.Millisecond)
		for len(imeRefreshChan) > 0 {
			<-imeRefreshChan
		}
		fg, _, _ := procGetForegroundWindow.Call()
		if fg != 0 {
			var dwResult uintptr
			procSendMessageTimeoutW.Call(
				fg, WM_SETTINGCHANGE, 0, strVal, SMTO_ABORTIFHUNG, 50, uintptr(unsafe.Pointer(&dwResult)),
			)
		}
	}
}

func RegisterWindowMessage(name string) uint32 {
	strPtr, _ := syscall.UTF16PtrFromString(name)
	ret, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(strPtr)))
	return uint32(ret)
}

func GetSystemMetrics(index int) int {
	ret, _, _ := procGetSystemMetrics.Call(uintptr(index))
	return int(ret)
}

func MessageBox(hwnd uintptr, text, caption string, uType uint32) int {
	textPtr, _ := syscall.UTF16PtrFromString(text)
	captionPtr, _ := syscall.UTF16PtrFromString(caption)
	ret, _, _ := procMessageBoxW.Call(hwnd, uintptr(unsafe.Pointer(textPtr)), uintptr(unsafe.Pointer(captionPtr)), uintptr(uType))
	return int(ret)
}

func CreateMutex(name string) (uintptr, error) {
	namePtr, _ := syscall.UTF16PtrFromString(name)
	ret, _, err := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(namePtr)))
	if ret != 0 && err == syscall.ERROR_ALREADY_EXISTS {
		return ret, err
	}
	if ret == 0 {
		return 0, err
	}
	return ret, nil
}

func CloseHandle(handle syscall.Handle) {
	procCloseHandle.Call(uintptr(handle))
}

func RegisterHotKey(hwnd uintptr, id int, fsModifiers, vk uint32) bool {
	ret, _, _ := procRegisterHotKey.Call(hwnd, uintptr(id), uintptr(fsModifiers), uintptr(vk))
	return ret != 0
}

func UnregisterHotKey(hwnd uintptr, id int) bool {
	ret, _, _ := procUnregisterHotKey.Call(hwnd, uintptr(id))
	return ret != 0
}

func GetMessage(msg *Msg, hwnd uintptr, msgFilterMin, msgFilterMax uint32) int {
	ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(msg)), hwnd, uintptr(msgFilterMin), uintptr(msgFilterMax))
	return int(ret)
}

func TranslateMessage(msg *Msg) bool {
	ret, _, _ := procTranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
	return ret != 0
}

func DispatchMessage(msg *Msg) uintptr {
	ret, _, _ := procDispatchMessage.Call(uintptr(unsafe.Pointer(msg)))
	return ret
}

func DefWindowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	ret, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func RegisterClass(className string, wndProc func(hwnd uintptr, msg uint32, wparam uintptr, lparam uintptr) uintptr) {
	classNamePtr, _ := syscall.UTF16PtrFromString(className)
	globalWndProcCallback = syscall.NewCallback(wndProc)

	wc := WndClassEx{
		CbSize:        uint32(unsafe.Sizeof(WndClassEx{})),
		LpfnWndProc:   globalWndProcCallback,
		LpszClassName: classNamePtr,
	}
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
}

func CreateWindowEx(dwExStyle uint32, lpClassName, lpWindowName string, dwStyle uint32, x, y, nWidth, nHeight int32, hWndParent, hMenu, hInstance, lpParam uintptr) uintptr {
	classNamePtr, _ := syscall.UTF16PtrFromString(lpClassName)
	windowNamePtr, _ := syscall.UTF16PtrFromString(lpWindowName)
	ret, _, _ := procCreateWindowEx.Call(
		uintptr(dwExStyle), uintptr(unsafe.Pointer(classNamePtr)), uintptr(unsafe.Pointer(windowNamePtr)),
		uintptr(dwStyle), uintptr(x), uintptr(y), uintptr(nWidth), uintptr(nHeight),
		hWndParent, hMenu, hInstance, lpParam,
	)
	return ret
}

func DestroyWindow(hwnd uintptr) bool {
	ret, _, _ := procDestroyWindow.Call(hwnd)
	return ret != 0
}

func PostQuitMessage(exitCode int32) {
	procPostQuitMessage.Call(uintptr(exitCode))
}

func PostMessage(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	ret, _, _ := procPostMessageW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func GetAsyncKeyState(vKey int) bool {
	ret, _, _ := procGetAsyncKeyState.Call(uintptr(vKey))
	return int16(ret) < 0
}

func FindWindow(className string) uintptr {
	classNamePtr, _ := syscall.UTF16PtrFromString(className)
	ret, _, _ := procFindWindowExW.Call(0, 0, uintptr(unsafe.Pointer(classNamePtr)), 0)
	return ret
}

func ShellNotifyIcon(dwMessage uint32, lpData *NotifyIconData) bool {
	ret, _, _ := procShellNotifyIconW.Call(uintptr(dwMessage), uintptr(unsafe.Pointer(lpData)))
	return ret != 0
}

func CreatePopupMenu() uintptr {
	ret, _, _ := procCreatePopupMenu.Call()
	return ret
}

func AppendMenu(hMenu uintptr, uFlags uint32, uIDNewItem uintptr, lpNewItem string) {
	textPtr, _ := syscall.UTF16PtrFromString(lpNewItem)
	procAppendMenuW.Call(hMenu, uintptr(uFlags), uIDNewItem, uintptr(unsafe.Pointer(textPtr)))
}

func TrackPopupMenu(hMenu uintptr, uFlags uint32, x, y int32, nReserved int, hWnd uintptr, prcRect uintptr) {
	procTrackPopupMenu.Call(hMenu, uintptr(uFlags), uintptr(x), uintptr(y), uintptr(nReserved), hWnd, prcRect)
}

func GetCursorPos() Point {
	var pt Point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	return pt
}

func SetForegroundWindow(hwnd uintptr) {
	procSetForegroundWindow.Call(hwnd)
}

func DestroyMenu(hMenu uintptr) {
	procDestroyMenu.Call(hMenu)
}

func HICONFromICOBytes(data []byte) uintptr {
	if len(data) < 22 {
		return 0
	}

	cx, _, _ := procGetSystemMetrics.Call(SM_CXSMICON)
	cy, _, _ := procGetSystemMetrics.Call(SM_CYSMICON)
	if cx == 0 {
		cx = 16
	}
	if cy == 0 {
		cy = 16
	}

	count := binary.LittleEndian.Uint16(data[4:6])
	if count == 0 {
		return 0
	}

	var bestIndex int = -1
	var bestDiff int = 99999

	for i := 0; i < int(count); i++ {
		offset := 6 + i*16
		if offset+16 > len(data) {
			break
		}

		w := int(data[offset])
		h := int(data[offset+1])
		if w == 0 {
			w = 256
		}
		if h == 0 {
			h = 256
		}

		diff := (w-int(cx))*(w-int(cx)) + (h-int(cy))*(h-int(cy))
		if diff < bestDiff {
			bestDiff = diff
			bestIndex = i
		}
	}

	if bestIndex == -1 {
		return 0
	}

	entryOffset := 6 + bestIndex*16
	bytesInRes := binary.LittleEndian.Uint32(data[entryOffset+8 : entryOffset+12])
	imageOffset := binary.LittleEndian.Uint32(data[entryOffset+12 : entryOffset+16])

	if imageOffset+bytesInRes > uint32(len(data)) {
		return 0
	}

	imageData := data[imageOffset : imageOffset+bytesInRes]

	if len(imageData) == 0 {
		return 0
	}

	ret, _, _ := procCreateIconFromResourceEx.Call(
		uintptr(unsafe.Pointer(&imageData[0])),
		uintptr(len(imageData)),
		1,
		0x00030000,
		cx,
		cy,
		0,
	)
	return ret
}

func abs32(n int32) int32 {
	if n < 0 {
		return -n
	}
	return n
}

func DestroyIcon(hIcon uintptr) bool {
	if hIcon == 0 {
		return false
	}
	ret, _, _ := procDestroyIcon.Call(hIcon)
	return ret != 0
}
