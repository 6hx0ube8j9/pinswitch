package core

import (
	"os"
	"runtime"
	"pinswitch/winapi"
)

const (
	RegPathInput = `SOFTWARE\Microsoft\InputMethod\Settings\CHS`
	RegValInput  = "Enable Double Pinyin"
	RegPathRun   = `Software\Microsoft\Windows\CurrentVersion\Run`
	RegValRun    = "PinswitchAutoStart" // 内部注册表键名也同步扁平化
)

type SwitchEngine struct {
	IsWriting bool
}

func NewSwitchEngine() *SwitchEngine {
	return &SwitchEngine{}
}

func (e *SwitchEngine) GetIMEMode() uint32 {
	hKey, err := winapi.RegOpenKeyEx(winapi.HKEY_CURRENT_USER, RegPathInput, winapi.KEY_QUERY_VALUE)
	if err != nil {
		return 0
	}
	defer winapi.RegCloseKey(hKey)
	
	val, err := winapi.RegQueryValueExDWORD(hKey, RegValInput)
	if err != nil {
		return 0
	}
	return val
}

func (e *SwitchEngine) SetIMEMode(mode uint32) {
	e.IsWriting = true
	winapi.RegSetKeyValueDWORD(RegPathInput, RegValInput, mode)
	e.IsWriting = false
}

func (e *SwitchEngine) IsAutoStart() bool {
	return winapi.IsAutoStartEnabled(RegPathRun, RegValRun)
}

func (e *SwitchEngine) ToggleAutoStart() {
	if e.IsAutoStart() {
		hKey, err := winapi.RegOpenKeyEx(winapi.HKEY_CURRENT_USER, RegPathRun, winapi.KEY_SET_VALUE)
		if err == nil {
			winapi.RegDeleteValue(hKey, RegValRun)
			winapi.RegCloseKey(hKey)
		}
	} else {
		exePath, err := os.Executable()
		if err == nil {
			winapi.EnableAutoStart(RegPathRun, RegValRun, exePath)
		}
	}
}

func (e *SwitchEngine) WatchRegistry(onChanged func()) {
	hKey, err := winapi.RegOpenKeyEx(winapi.HKEY_CURRENT_USER, RegPathInput, winapi.KEY_NOTIFY|winapi.KEY_QUERY_VALUE)
	if err != nil {
		return
	}
	defer winapi.RegCloseKey(hKey)

	hEvent := winapi.CreateEvent()
	if hEvent == 0 {
		return
	}
	defer winapi.CloseHandle(hEvent)

	for {
		winapi.RegNotifyChangeKeyValue(hKey, hEvent)
		res := winapi.WaitForSingleObject(hEvent, winapi.INFINITE)
		if res == winapi.WAIT_OBJECT_0 {
			winapi.ResetEvent(hEvent)
			if e.IsWriting {
				continue
			}
			onChanged()
		}
	}
	// 内存安全保障：确保整个进程生命周期内外部 API 句柄不被 Go 意外释放
	runtime.KeepAlive(hKey)
}
