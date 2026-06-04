package core

import (
	"context"
	"os"
	"sync/atomic"
	"syscall"
	"pinswitch/winapi"
)

const (
	RegPathInput = `SOFTWARE\Microsoft\InputMethod\Settings\CHS`
	RegValInput  = "Enable Double Pinyin"
	RegPathRun   = `Software\Microsoft\Windows\CurrentVersion\Run`
	RegValRun    = "PinswitchAutoStart"
)

type SwitchEngine struct {
	IsWriting int32
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

func (e *SwitchEngine) SetIMEMode(mode uint32) bool {
	if e.GetIMEMode() == mode {
		return false
	}
	if !atomic.CompareAndSwapInt32(&e.IsWriting, 0, 1) {
		return false
	}
	defer atomic.StoreInt32(&e.IsWriting, 0)

	winapi.RegSetKeyValueDWORD(RegPathInput, RegValInput, mode)
	return true
}

func (e *SwitchEngine) IsAutoStart() bool {
	hKey, err := winapi.RegOpenKeyEx(winapi.HKEY_CURRENT_USER, RegPathRun, winapi.KEY_QUERY_VALUE)
	if err != nil {
		return false
	}
	defer winapi.RegCloseKey(hKey)
	return winapi.RegQueryValueExSZ(hKey, RegValRun)
}

func (e *SwitchEngine) ToggleAutoStart() {
	if e.IsAutoStart() {
		winapi.RegDeleteKeyValue(RegPathRun, RegValRun)
	} else {
		exePath, err := os.Executable()
		if err == nil {
			winapi.RegSetKeyValueSZ(RegPathRun, RegValRun, exePath)
		}
	}
}

func (e *SwitchEngine) WatchRegistry(ctx context.Context, onChanged func()) {
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
		select {
		case <-ctx.Done():
			return
		default:
			winapi.RegNotifyChangeKeyValue(hKey, hEvent)
			res := winapi.WaitForSingleObject(hEvent, 100)
			if res == winapi.WAIT_OBJECT_0 {
				winapi.ResetEvent(hEvent)
				if atomic.LoadInt32(&e.IsWriting) == 1 {
					continue
				}
				onChanged()
			}
		}
	}
}
