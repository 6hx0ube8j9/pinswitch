package winapi

import (
	"strings"
	"syscall"
	"unsafe"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procRegisterHotKey           = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey         = user32.NewProc("UnregisterHotKey")
	procGetMessage               = user32.NewProc("GetMessageW")
	
	advapi32                     = syscall.NewLazyDLL("advapi32.dll")
	procRegOpenKeyEx             = advapi32.NewProc("RegOpenKeyExW")
	procRegCloseKey              = advapi32.NewProc("RegCloseKey")
	procRegQueryValueEx          = advapi32.NewProc("RegQueryValueExW")
	procRegSetKeyValue           = advapi32.NewProc("RegSetKeyValueW")
	procRegDeleteValue           = advapi32.NewProc("RegDeleteValueW")
	procRegNotifyChangeKeyValue  = advapi32.NewProc("RegNotifyChangeKeyValue")

	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = kernel32.NewProc("Process32FirstW")
	procProcess32Next            = kernel32.NewProc("Process32NextW")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procTerminateProcess         = kernel32.NewProc("TerminateProcess")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
	procCreateEvent              = kernel32.NewProc("CreateEventW")
	procWaitForSingleObject      = kernel32.NewProc("WaitForSingleObject")
	procResetEvent               = kernel32.NewProc("ResetEvent")
)

const (
	HKEY_CURRENT_USER          = 0x80000001
	KEY_QUERY_VALUE            = 0x0001
	KEY_SET_VALUE              = 0x0002
	KEY_NOTIFY                 = 0x0010
	REG_SZ                     = 1
	REG_DWORD                  = 4
	REG_NOTIFY_CHANGE_LAST_SET = 0x00000004
	WAIT_OBJECT_0              = 0x00000000
	INFINITE                   = 0xFFFFFFFF
	TH32CS_SNAPPROCESS         = 0x00000002
	PROCESS_TERMINATE          = 0x0001
)

type TagMSG struct {
	Hwnd    uint32
	Message uint32
	Wparam  uintptr
	Lparam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type TagPROCESSENTRY32W struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

func RegisterHotKey(id int, modifiers, vk uint32) bool {
	r1, _, _ := procRegisterHotKey.Call(0, uintptr(id), uintptr(modifiers), uintptr(vk))
	return r1 != 0
}

func UnregisterHotKey(id int) {
	procUnregisterHotKey.Call(0, uintptr(id))
}

func GetMessage(msg *TagMSG) int32 {
	r1, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(msg)), 0, 0, 0)
	return int32(r1)
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

func KillOldInstances(targetExeName string, currentPID uint32) {
	snapshot, _, _ := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if syscall.Handle(snapshot) == syscall.InvalidHandle {
		return
	}
	defer procCloseHandle.Call(snapshot)

	var entry TagPROCESSENTRY32W
	entry.Size = uint32(unsafe.Sizeof(entry))

	ret, _, _ := procProcess32First.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	for ret != 0 {
		exeName := syscall.UTF16ToString(entry.ExeFile[:])
		if strings.EqualFold(exeName, targetExeName) && entry.ProcessID != currentPID {
			hProcess, _, _ := procOpenProcess.Call(PROCESS_TERMINATE, 0, uintptr(entry.ProcessID))
			if hProcess != 0 {
				procTerminateProcess.Call(hProcess, 0)
				procCloseHandle.Call(hProcess)
			}
		}
		ret, _, _ = procProcess32Next.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	}
}

func RegOpenKeyEx(hKey uintptr, path string, samDesired uint32) (syscall.Handle, error) {
	var out syscall.Handle
	pathPtr, _ := syscall.UTF16PtrFromString(path)
	r1, _, _ := procRegOpenKeyEx.Call(hKey, uintptr(unsafe.Pointer(pathPtr)), 0, uintptr(samDesired), uintptr(unsafe.Pointer(&out)))
	if r1 != 0 {
		return 0, syscall.Errno(r1)
	}
	return out, nil
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

func RegSetKeyValueDWORD(path, valueName string, value uint32) {
	pathPtr, _ := syscall.UTF16PtrFromString(path)
	valuePtr, _ := syscall.UTF16PtrFromString(valueName)
	procRegSetKeyValue.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(pathPtr)), uintptr(unsafe.Pointer(valuePtr)), REG_DWORD, uintptr(unsafe.Pointer(&value)), 4)
}

func RegNotifyChangeKeyValue(hKey syscall.Handle, hEvent syscall.Handle) {
	procRegNotifyChangeKeyValue.Call(uintptr(hKey), 0, REG_NOTIFY_CHANGE_LAST_SET, uintptr(hEvent), 1)
}

func RegDeleteValue(hKey syscall.Handle, valueName string) {
	valuePtr, _ := syscall.UTF16PtrFromString(valueName)
	procRegDeleteValue.Call(uintptr(hKey), uintptr(unsafe.Pointer(valuePtr)))
}

func IsAutoStartEnabled(path, key string) bool {
	hKey, err := RegOpenKeyEx(HKEY_CURRENT_USER, path, KEY_QUERY_VALUE)
	if err != nil {
		return false
	}
	defer RegCloseKey(hKey)

	valuePtr, _ := syscall.UTF16PtrFromString(key)
	var size uint32 = 520
	buf := make([]uint16, 260)
	r1, _, _ := procRegQueryValueEx.Call(uintptr(hKey), uintptr(unsafe.Pointer(valuePtr)), 0, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	return r1 == 0
}

func EnableAutoStart(path, key, exePath string) {
	pathPtr, _ := syscall.UTF16PtrFromString(path)
	valuePtr, _ := syscall.UTF16PtrFromString(key)
	exePathStr := `"` + exePath + `"`
	exePathPtr, _ := syscall.UTF16PtrFromString(exePathStr)
	size := uint32((len(exePathStr) + 1) * 2)

	procRegSetKeyValue.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(pathPtr)), uintptr(unsafe.Pointer(valuePtr)), REG_SZ, uintptr(unsafe.Pointer(exePathPtr)), uintptr(size))
}
