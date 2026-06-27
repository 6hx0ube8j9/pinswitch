//go:build windows

package main

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/energye/systray"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	RegPathInput   = `SOFTWARE\Microsoft\InputMethod\Settings\CHS`
	RegValInput    = "Enable Double Pinyin"
	RegPathRun     = `Software\Microsoft\Windows\CurrentVersion\Run`
	RegValRun      = "PinswitchAutoStart"
	RegPathApp     = `Software\Pinswitch`
	RegValHideTray = "HideTrayIcon"

	HotkeyToggleMode = 1
	HotkeyToggleHide = 2
)

type SwitchBrain struct {
	hwnd           uintptr
	lastToggle     time.Time
	lastToggleHide time.Time
	toggleMu       sync.Mutex
	isTogglingHide bool
}

func NewSwitchBrain() *SwitchBrain {
	return &SwitchBrain{
		lastToggleHide: time.Now(), 
	}
}

func (b *SwitchBrain) GetIMEMode() uint32 {
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

func (b *SwitchBrain) SetIMEMode(mode uint32) bool {
	if b.GetIMEMode() == mode {
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

func (b *SwitchBrain) ToggleMode() {
	b.toggleMu.Lock()
	defer b.toggleMu.Unlock()

	if time.Since(b.lastToggle) < 200*time.Millisecond {
		return
	}
	b.lastToggle = time.Now()

	current := b.GetIMEMode()
	if b.SetIMEMode(1 - current) {
		AsyncRefreshActiveWindowIME()
	}
}

func (b *SwitchBrain) IsAutoStart() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, RegPathRun, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	_, _, err = k.GetStringValue(RegValRun)
	return err == nil
}

func (b *SwitchBrain) ToggleAutoStart() {
	k, err := registry.OpenKey(registry.CURRENT_USER, RegPathRun, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()

	if b.IsAutoStart() {
		k.DeleteValue(RegValRun)
	} else {
		exePath, err := os.Executable()
		if err == nil {
			safePath := `"` + exePath + `"`
			k.SetStringValue(RegValRun, safePath)
		}
	}
}

func (b *SwitchBrain) IsTrayHidden() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, RegPathApp, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	val, _, err := k.GetIntegerValue(RegValHideTray)
	return err == nil && val == 1
}

func (b *SwitchBrain) SetTrayHidden(hide bool) {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, RegPathApp, registry.ALL_ACCESS)
	if err != nil {
		return
	}
	defer k.Close()

	if hide {
		k.SetDWordValue(RegValHideTray, 1)
	} else {
		k.SetDWordValue(RegValHideTray, 0)
	}
}

func (b *SwitchBrain) ToggleHide() {
	b.toggleMu.Lock()
	if b.isTogglingHide || time.Since(b.lastToggleHide) < 1000*time.Millisecond {
		b.toggleMu.Unlock()
		return
	}
	b.isTogglingHide = true
	b.lastToggleHide = time.Now()
	b.toggleMu.Unlock()

	isHidden := b.IsTrayHidden()
	b.SetTrayHidden(!isHidden)

	exePath, _ := os.Executable()
	cmd := exec.Command(exePath, "-restart")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000008}
	cmd.Start()

	if !isHidden {
		systray.Quit()
	} else {
		if b.hwnd != 0 {
			PostMessage(b.hwnd, WM_CLOSE, 0, 0)
		}
	}
}

func (b *SwitchBrain) StartHotkeyListener() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	className := "PinswitchHotkeyWindow_Unique_Class"
	RegisterClass(className, func(hwnd uintptr, msg uint32, wparam uintptr, lparam uintptr) uintptr {
		switch msg {
		case WM_HOTKEY:
			switch int(wparam) {
			case HotkeyToggleMode:
				b.ToggleMode()
			case HotkeyToggleHide:
				b.ToggleHide()
			}
			return 0
		case WM_USER + 777:
			b.ToggleMode()
			return 0
		case WM_USER + 778:
			b.ToggleHide()
			return 0
		case WM_CLOSE:
			UnregisterHotKey(hwnd, HotkeyToggleMode)
			UnregisterHotKey(hwnd, HotkeyToggleHide)
			DestroyWindow(hwnd)
			PostQuitMessage(0)
			return 0
		}
		return DefWindowProc(hwnd, msg, wparam, lparam)
	})

	hwnd := CreateWindowEx(0, className, "PinswitchHotkey", 0, 0, 0, 0, 0, 0, 0, 0, 0)
	if hwnd == 0 {
		return
	}
	b.hwnd = hwnd

	RegisterHotKey(hwnd, HotkeyToggleMode, 0x0002|0x0004, 0x59)
	RegisterHotKey(hwnd, HotkeyToggleHide, 0x0004|0x0002|0x0008, 0x59)

	var msg Msg
	for {
		ret := GetMessage(&msg, 0, 0, 0)
		if ret == 0 || ret == -1 {
			break
		}
		TranslateMessage(&msg)
		DispatchMessage(&msg)
	}
}

func (b *SwitchBrain) Close() {
	if b.hwnd != 0 {
		PostMessage(b.hwnd, WM_CLOSE, 0, 0)
	}
}

func (b *SwitchBrain) WatchRegistry(ctx context.Context, onChanged func()) {
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
		err = windows.RegNotifyChangeKeyValue(windows.Handle(k), false, windows.REG_NOTIFY_CHANGE_LAST_SET, regEvent, true)
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

func main() {
	isRestart := len(os.Args) > 1 && os.Args[1] == "-restart"

	var ret uintptr
	var err error

	if isRestart {
		for i := 0; i < 30; i++ {
			ret, err = CreateMutex("Local\\PinswitchUniqueMutexSecure")
			if err != syscall.Errno(183) && ret != 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	} else {
		ret, err = CreateMutex("Local\\PinswitchUniqueMutexSecure")
	}

   if err == syscall.Errno(183) {
		if isRestart {
			return
		}

		oldHwnd := FindWindow("PinswitchHotkeyWindow_Unique_Class")
		if oldHwnd != 0 {
			if GetAsyncKeyState(0x10) { 
				PostMessage(oldHwnd, WM_USER+778, 0, 0)
			} else { 
				PostMessage(oldHwnd, WM_USER+777, 0, 0)
			}
		}

		return
		
	} else if ret == 0 {
		return
	}

	defer func() {
		if ret != 0 {
			CloseHandle(syscall.Handle(ret))
		}
	}()

	brain := NewSwitchBrain()

	if brain.IsTrayHidden() {
		brain.StartHotkeyListener()
	} else {
		go brain.StartHotkeyListener()
		tray := NewTrayUI(brain)
		tray.Start()
	}
}
