package winapi

import (
	"syscall"
	"unsafe"
)

const (
	WM_CLOSE  = 0x0010
	WM_HOTKEY = 0x0312
	WM_USER   = 0x0400
	
	WM_SETTINGCHANGE  = 0x001A
	WM_IME_CONTROL    = 0x0283
	IMC_GETOPENSTATUS = 0x0005
	IMC_SETOPENSTATUS = 0x0006
	HWND_BROADCAST    = 0xFFFF
	SMTO_ABORTIFHUNG  = 0x0002

	HotkeyToggleMode = 1
	HotkeyToggleHide = 2
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	imm32                = syscall.NewLazyDLL("imm32.dll")

	procRegisterHotKey      = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey    = user32.NewProc("UnregisterHotKey")
	procGetMessage          = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessage     = user32.NewProc("DispatchMessageW")
	procDefWindowProc       = user32.NewProc("DefWindowProcW")
	procRegisterClassEx     = user32.NewProc("RegisterClassExW")
	procCreateWindowEx      = user32.NewProc("CreateWindowExW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procFindWindowW         = user32.NewProc("FindWindowW")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procGetAsyncKeyState      = user32.NewProc("GetAsyncKeyState")
	procMessageBoxW         = user32.NewProc("MessageBoxW")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procSendMessageW        = user32.NewProc("SendMessageW")       
	procSendMessageTimeoutW = user32.NewProc("SendMessageTimeoutW")

	procImmGetDefaultIMEWnd = imm32.NewProc("ImmGetDefaultIMEWnd")

	procCreateMutexW        = kernel32.NewProc("CreateMutexW")
	procCloseHandle         = kernel32.NewProc("CloseHandle")
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
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
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

func GetAsyncKeyState(vKey int) bool {
	ret, _, _ := procGetAsyncKeyState.Call(uintptr(vKey))
	return int16(ret) < 0
}

func MessageBox(hwnd uintptr, text, caption string, flags uint32) int {
	textPtr, _ := syscall.UTF16PtrFromString(text)
	captionPtr, _ := syscall.UTF16PtrFromString(caption)
	ret, _, _ := procMessageBoxW.Call(
		hwnd,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(captionPtr)),
		uintptr(flags),
	)
	return int(ret)
}

func GetForegroundWindow() uintptr {
	ret, _, _ := procGetForegroundWindow.Call()
	return ret
}

func SendMessage(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	ret, _, _ := procSendMessageW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func SendMessageTimeout(hwnd uintptr, msg uint32, wParam, lParam uintptr, flags uint32, timeout uint32) uintptr {
	var result uintptr
	procSendMessageTimeoutW.Call(
		hwnd,
		uintptr(msg),
		wParam,
		lParam,
		uintptr(flags),
		uintptr(timeout),
		uintptr(unsafe.Pointer(&result)),
	)
	return result
}

func ImmGetDefaultIMEWnd(hwnd uintptr) uintptr {
	ret, _, _ := procImmGetDefaultIMEWnd.Call(hwnd)
	return ret
}

func RefreshActiveWindowIME() {
	time.Sleep(50 * time.Millisecond)

	fg := GetForegroundWindow()
	if fg != 0 {
		PostMessage(fg, WM_SETTINGCHANGE, 0, 0)

		imeWnd := ImmGetDefaultIMEWnd(fg)
		if imeWnd != 0 {
			status := SendMessageTimeout(imeWnd, WM_IME_CONTROL, IMC_GETOPENSTATUS, 0, SMTO_ABORTIFHUNG, 50)
			
			if status != 0 {
				PostMessage(imeWnd, WM_IME_CONTROL, IMC_SETOPENSTATUS, 0)
				PostMessage(imeWnd, WM_IME_CONTROL, IMC_SETOPENSTATUS, 1)
			} else {
				PostMessage(imeWnd, WM_IME_CONTROL, IMC_SETOPENSTATUS, 1)
				PostMessage(imeWnd, WM_IME_CONTROL, IMC_SETOPENSTATUS, 0)
			}
		}
	}
	SendMessageTimeout(HWND_BROADCAST, WM_SETTINGCHANGE, 0, 0, SMTO_ABORTIFHUNG, 100)
}
