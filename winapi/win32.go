package winapi

import (
	"syscall"
	"unsafe"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procRegisterHotKey           = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey         = user32.NewProc("UnregisterHotKey")
	procGetMessage               = user32.NewProc("GetMessageW")
	procTranslateMessage         = user32.NewProc("TranslateMessage")
	procDispatchMessage          = user32.NewProc("DispatchMessageW")
	procDefWindowProc            = user32.NewProc("DefWindowProcW")
	procRegisterClass            = user32.NewProc("RegisterClassW")
	procCreateWindowEx           = user32.NewProc("CreateWindowExW")
	procDestroyWindow            = user32.NewProc("DestroyWindow")
	procPostQuitMessage          = user32.NewProc("PostQuitMessage")
	procFindWindowW              = user32.NewProc("FindWindowW")
	procSendMessageW             = user32.NewProc("SendMessageW")
	
	advapi32                     = syscall.NewLazyDLL("advapi32.dll")
	procRegOpenKeyEx             = advapi32.NewProc("RegOpenKeyExW")
	procRegCloseKey              = advapi32.NewProc("RegCloseKey")
	procRegQueryValueEx          = advapi32.NewProc("RegQueryValueExW")
	procRegSetKeyValue           = advapi32.NewProc("RegSetKeyValueW")
	procRegDeleteKeyValue        = advapi32.NewProc("RegDeleteKeyValueW")
	procRegNotifyChangeKeyValue  = advapi32.NewProc("RegNotifyChangeKeyValue")

	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex              = kernel32.NewProc("CreateMutexW")
	procCreateEvent              = kernel32.NewProc("CreateEventW")
	procWaitForSingleObject      = kernel32.NewProc("WaitForSingleObject")
	procResetEvent               = kernel32.NewProc("ResetEvent")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
)

const (
	HKEY_CURRENT_USER          = 0x80000001
	KEY_QUERY_VALUE            = 0x0001
	KEY_NOTIFY                 = 0x0010
	REG_DWORD                  = 4
	REG_SZ                     = 1
	REG_NOTIFY_CHANGE_LAST_SET = 0x00000004
	WAIT_OBJECT_0              = 0x00000000
	HWND_MESSAGE               = ^uintptr(2)
	WM_CLOSE                   = 0x0010
	WM_HOTKEY                  = 0x0312
	WM_DESTROY                 = 0x0002
)

type TagMSG struct {
	Hwnd    uintptr
	Message uint32
	Wparam  uintptr
	Lparam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type TagWNDCLASS struct {
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
}

func CreateMutex(name string) (uintptr, error) {
	namePtr, _ := syscall.UTF16PtrFromString(name)
	ret, _, _ := procCreateMutex.Call(0, 1, uintptr(unsafe.Pointer(namePtr)))
	return ret, syscall.GetLastError()
}

func FindWindow(className string) uintptr {
	namePtr, _ := syscall.UTF16PtrFromString(className)
	ret, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(namePtr)), 0)
	return ret
}

func SendMessage(hwnd uintptr, msg uint32, wparam, lparam uintptr) uintptr {
	ret, _, _ := procSendMessageW.Call(hwnd, uintptr(msg), wparam, lparam)
	return ret
}

func RegisterClass(className string, wndProc func(hwnd uintptr, msg uint32, wparam uintptr, lparam uintptr) uintptr) {
	namePtr, _ := syscall.UTF16PtrFromString(className)
	hInstance, _, _ := procCloseHandle.Call(0)
	wc := TagWNDCLASS{
		LpfnWndProc:   syscall.NewCallback(wndProc),
		HInstance:     hInstance,
		LpszClassName: namePtr,
	}
	procRegisterClass.Call(uintptr(unsafe.Pointer(&wc)))
}

func CreateWindowEx(className string) uintptr {
	namePtr, _ := syscall.UTF16PtrFromString(className)
	hInstance, _, _ := procCloseHandle.Call(0)
	hwnd, _, _ := procCreateWindowEx.Call(0, uintptr(unsafe.Pointer(namePtr)), uintptr(unsafe.Pointer(namePtr)), 0, 0, 0, 0, 0, HWND_MESSAGE, 0, hInstance, 0)
	return hwnd
}

func DestroyWindow(hwnd uintptr) {
	procDestroyWindow.Call(hwnd)
}

func PostQuitMessage(exitCode int32) {
	procPostQuitMessage.Call(uintptr(exitCode))
}

func DefWindowProc(hwnd uintptr, msg uint32, wparam, lparam uintptr) uintptr {
	r1, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wparam, lparam)
	return r1
}

func RegisterHotKey(hwnd uintptr, id int, modifiers, vk uint32) bool {
	r1, _, _ := procRegisterHotKey.Call(hwnd, uintptr(id), uintptr(modifiers), uintptr(vk))
	return r1 != 0
}

func UnregisterHotKey(hwnd uintptr, id int) {
	procUnregisterHotKey.Call(hwnd, uintptr(id))
}

func GetMessage(msg *TagMSG) int32 {
	r1, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(msg)), 0, 0, 0)
	return int32(r1)
}

func TranslateMessage(msg *TagMSG) {
	procTranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
}

func DispatchMessage(msg *TagMSG) {
	procDispatchMessage.Call(uintptr(unsafe.Pointer(msg)))
}

func CreateEvent() syscall.Handle {
	h, _, _ := procCreateEvent.Call(0, 1, 0, 0)
	return syscall.Handle(h)
}

func CloseHandle(h syscall.Handle) {
	procCloseHandle.Call(uintptr(h))
}

func WaitForSingleObject(h syscall.Handle, ms uint32) uint32 {
	r1, _, _ := procWaitForSingleObject.Call(uintptr(h), uintptr(ms))
	return uint32(r1)
}

func ResetEvent(h syscall.Handle) {
	procResetEvent.Call(uintptr(h))
}

func RegOpenKeyEx(hKey uintptr, path string, samDesired uint32) (syscall.Handle, error) {
	var out uintptr
	pathPtr, _ := syscall.UTF16PtrFromString(path)
	r1, _, _ := procRegOpenKeyEx.Call(hKey, uintptr(unsafe.Pointer(pathPtr)), 0, uintptr(samDesired), uintptr(unsafe.Pointer(&out)))
	if r1 != 0 {
		return 0, syscall.Errno(r1)
	}
	return syscall.Handle(out), nil
}

func RegCloseKey(hKey syscall.Handle) {
	procRegCloseKey.Call(uintptr(hKey))
}

func RegQueryValueExDWORD(hKey syscall.Handle, valueName string) (uint32, error) {
	valuePtr, _ := syscall.UTF16PtrFromString(valueName)
	var value, size uint32 = 0, 4
	r1, _, _ := procRegQueryValueEx.Call(uintptr(hKey), uintptr(unsafe.Pointer(valuePtr)), 0, 0, uintptr(unsafe.Pointer(&value)), uintptr(unsafe.Pointer(&size)))
	if r1 != 0 {
		return 0, syscall.Errno(r1)
	}
	return value, nil
}

func RegQueryValueExSZ(hKey syscall.Handle, valueName string) bool {
	valuePtr, _ := syscall.UTF16PtrFromString(valueName)
	var size uint32 = 520
	buf := make([]uint16, 260)
	r1, _, _ := procRegQueryValueEx.Call(uintptr(hKey), uintptr(unsafe.Pointer(valuePtr)), 0, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	return r1 == 0
}

func RegSetKeyValueDWORD(path, valueName string, value uint32) {
	pathPtr, _ := syscall.UTF16PtrFromString(path)
	valuePtr, _ := syscall.UTF16PtrFromString(valueName)
	procRegSetKeyValue.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(pathPtr)), uintptr(unsafe.Pointer(valuePtr)), REG_DWORD, uintptr(unsafe.Pointer(&value)), 4)
}

func RegSetKeyValueSZ(path, valueName, valueStr string) {
	pathPtr, _ := syscall.UTF16PtrFromString(path)
	valuePtr, _ := syscall.UTF16PtrFromString(valueName)
	valStr := `"` + valueStr + `"`
	valPtr, _ := syscall.UTF16PtrFromString(valStr)
	size := uint32((len(valStr) + 1) * 2)
	procRegSetKeyValue.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(pathPtr)), uintptr(unsafe.Pointer(valuePtr)), REG_SZ, uintptr(unsafe.Pointer(valPtr)), uintptr(size))
}

func RegDeleteKeyValue(path, valueName string) {
	pathPtr, _ := syscall.UTF16PtrFromString(path)
	valuePtr, _ := syscall.UTF16PtrFromString(valueName)
	procRegDeleteKeyValue.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(pathPtr)), uintptr(unsafe.Pointer(valuePtr)))
}

func RegNotifyChangeKeyValue(hKey syscall.Handle, hEvent syscall.Handle) {
	procRegNotifyChangeKeyValue.Call(uintptr(hKey), 0, REG_NOTIFY_CHANGE_LAST_SET, uintptr(hEvent), 1)
}
