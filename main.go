//go:build windows

package main

import (
	"context"
	"os"
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
	OnTrayToggle   func(hide bool) 
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
	if time.Since(b.lastToggleHide) < 500*time.Millisecond {
		b.toggleMu.Unlock()
		return
	}
	b.lastToggleHide = time.Now()
	b.toggleMu.Unlock()

	isHidden := !b.IsTrayHidden()
	b.SetTrayHidden(isHidden)

	if b.OnTrayToggle != nil {
		b.OnTrayToggle(isHidden)
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

	hwnd := CreateWindowEx(0, className, "PinswitchHotkey", 0, 0, 0, 0, 0, ^uintptr(2), 0, 0, 0)
	if hwnd == 0 {
		return
	}
	b.hwnd = hwnd

	if !RegisterHotKey(hwnd, HotkeyToggleMode, 0x0002|0x0004, 0x59) {
		MessageBox(0, "切换模式快捷键注册失败，可能被其他程序占用。", "Pinswitch 提示", 0x00000010)
	}
	
	if !RegisterHotKey(hwnd, HotkeyToggleHide, 0x0004|0x0002|0x0008, 0x59) {
		MessageBox(0, "显示/隐藏快捷键注册失败，可能被其他程序占用。", "Pinswitch 提示", 0x00000010)
	}

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

	regEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		k.Close()
		return
	}

	quitEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		windows.CloseHandle(regEvent)
		k.Close()
		return
	}

	watchCtx, cancel := context.WithCancel(ctx)
	exited := make(chan struct{})

	go func() {
		<-watchCtx.Done()
		windows.SetEvent(quitEvent)
		close(exited)
	}()

	defer func() {
		cancel()
		<-exited
		windows.CloseHandle(quitEvent)
		windows.CloseHandle(regEvent)
		k.Close()
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
	ret, err := CreateMutex("Local\\PinswitchUniqueMutexSecure")
	if err == syscall.Errno(183) || ret == 0 {
		oldHwnd := FindWindow("PinswitchHotkeyWindow_Unique_Class")
		if oldHwnd != 0 {
			if GetAsyncKeyState(0x10) { 
				PostMessage(oldHwnd, WM_USER+778, 0, 0)
			} else { 
				PostMessage(oldHwnd, WM_USER+777, 0, 0)
			}
		}
		return
	}

	defer func() {
		if ret != 0 {
			CloseHandle(syscall.Handle(ret))
		}
	}()

	brain := NewSwitchBrain()
	exitChan := make(chan struct{})

	go func() {
		brain.StartHotkeyListener()
		close(exitChan)
	}()

	brain.OnTrayToggle = func(hide bool) {
		if hide {
			systray.Quit()
		} else {
			go func() {
				tray := NewTrayUI(brain)
				tray.Start()
			}()
		}
	}

	if !brain.IsTrayHidden() {
		go func() {
			tray := NewTrayUI(brain)
			tray.Start()
		}()
	}

	<-exitChan
}
