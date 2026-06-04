package core

import (
	"context"
	"os"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	RegPathInput = `SOFTWARE\Microsoft\InputMethod\Settings\CHS`
	RegValInput  = "Enable Double Pinyin"
	RegPathRun   = `Software\Microsoft\Windows\CurrentVersion\Run`
	RegValRun    = "PinswitchAutoStart"
)

type SwitchEngine struct{}

func NewSwitchEngine() *SwitchEngine {
	return &SwitchEngine{}
}

func (e *SwitchEngine) GetIMEMode() uint32 {
	k, err := registry.OpenKey(registry.CURRENT_USER, RegPathInput, registry.QUERY_VALUE)
	if err != nil {
		return 0
	}
	defer k.Close()

	val, _, err := k.GetIntegerValue(RegValInput)
	if err != nil {
		return 0
	}
	return uint32(val)
}

func (e *SwitchEngine) SetIMEMode(mode uint32) bool {
	if e.GetIMEMode() == mode {
		return false
	}
	
	k, err := registry.OpenKey(registry.CURRENT_USER, RegPathInput, registry.SET_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	err = k.SetDWordValue(RegValInput, mode)
	return err == nil
}

func (e *SwitchEngine) IsAutoStart() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, RegPathRun, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	_, _, err = k.GetStringValue(RegValRun)
	return err == nil
}

func (e *SwitchEngine) ToggleAutoStart() {
	k, err := registry.OpenKey(registry.CURRENT_USER, RegPathRun, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()

	if e.IsAutoStart() {
		k.DeleteValue(RegValRun)
	} else {
		exePath, err := os.Executable()
		if err == nil {
			k.SetStringValue(RegValRun, exePath)
		}
	}
}

func (e *SwitchEngine) WatchRegistry(ctx context.Context, onChanged func()) {
	k, err := registry.OpenKey(registry.CURRENT_USER, RegPathInput, registry.NOTIFY)
	if err != nil {
		return
	}
	defer k.Close()

	regEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return
	}
	defer windows.CloseHandle(regEvent)

	quitEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return
	}
	defer windows.CloseHandle(quitEvent)

	go func() {
		<-ctx.Done()
		windows.SetEvent(quitEvent)
	}()

	events := []windows.Handle{regEvent, quitEvent}

	for {
		err = windows.RegNotifyChangeKeyValue(
			windows.Handle(k),
			false,
			windows.REG_NOTIFY_CHANGE_LAST_SET,
			regEvent,
			true,
		)
		if err != nil {
			return
		}

		s, err := windows.WaitForMultipleObjects(events, false, windows.INFINITE)
		if err != nil {
			return
		}

		switch s {
		case windows.WAIT_OBJECT_0:
			onChanged()
		case windows.WAIT_OBJECT_0 + 1:
			return
		}
	}
}
