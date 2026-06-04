package winapi

import (
	"syscall"
	"unsafe"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	
	procRegisterHotKey   = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
	procGetMessage       = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessage  = user32.NewProc("DispatchMessageW")
	procDefWindowProc    = user32.NewProc("DefWindowProcW")
	procRegisterClass    = user32.NewProc("RegisterClassW")
	procCreateWindowEx   = user32.NewProc("CreateWindowExW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procFindWindowW      = user32.NewProc("FindWindowW")
	procPostMessageW     = user32.NewProc("PostMessageW")

	procCreateMutexW     = kernel32.NewProc("CreateMutexW")
	procCloseHandle      = kernel32.NewProc("CloseHandle")
)

type Msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type WndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

func CreateMutex(name string) (uintptr, error) {
	namePtr, _ := syscall.UTF16PtrFromString(name)
	ret, _, err := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(namePtr)))
	if ret != 0 && err == syscall.Errno(183) {
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
	wc := WndClassEx{
		CbSize:        uint32(unsafe.Sizeof(WndClassEx{})),
		LpfnWndProc:   syscall.NewCallback(wndProc),
		LpszClassName: classNamePtr,
	}
	procRegisterClass.Call(uintptr(unsafe.Pointer(&wc)))
}

func CreateWindowEx(dwExStyle uint32, lpClassName, lpWindowName string, dwStyle uint32, x, y, nWidth, nHeight int32, hWndParent, hMenu, hInstance, lpParam uintptr) uintptr {
	classNamePtr, _ := syscall.UTF16PtrFromString(lpClassName)
	windowNamePtr, _ := syscall.UTF16PtrFromString(lpWindowName)
	ret, _, _ := procCreateWindowEx.Call(
		uintptr(dwExStyle),
		uintptr(unsafe.Pointer(classNamePtr)),
		uintptr(unsafe.Pointer(windowNamePtr)),
		uintptr(dwStyle),
		uintptr(x), uintptr(y), uintptr(nWidth), uintptr(nHeight),
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

func FindWindow(className string) uintptr {
	classNamePtr, _ := syscall.UTF16PtrFromString(className)
	ret, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(classNamePtr)), 0)
	return ret
}

func PostMessage(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	ret, _, _ := procPostMessageW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}
