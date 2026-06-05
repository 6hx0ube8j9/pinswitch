//go:build windows

package main

import (
	"os"
	"syscall"
	"time"

	"pinswitch/core"
	"pinswitch/ui"
	"pinswitch/winapi"
)

func main() {
	isRestart := len(os.Args) > 1 && os.Args[1] == "-restart"

	var ret uintptr
	var err error

	if isRestart {
		for i := 0; i < 30; i++ {
			ret, err = winapi.CreateMutex("Local\\PinswitchUniqueMutexSecure")
			if err != syscall.Errno(183) && ret != 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	} else {
		ret, err = winapi.CreateMutex("Local\\PinswitchUniqueMutexSecure")
	}

    if err == syscall.Errno(183) {
		if isRestart {
			return
		}

		oldHwnd := winapi.FindWindow("PinswitchHotkeyWindow_Unique_Class")
		if oldHwnd != 0 {
			if winapi.GetAsyncKeyState(0x10) {
				winapi.PostMessage(oldHwnd, winapi.WM_USER+778, 0, 0)
			} else {
				winapi.PostMessage(oldHwnd, winapi.WM_USER+777, 0, 0)
			}
		}
		
		winapi.PostQuitMessage(0)
		var msg winapi.Msg
		for winapi.GetMessage(&msg, 0, 0, 0) > 0 {
			winapi.TranslateMessage(&msg)
			winapi.DispatchMessage(&msg)
		}
		
		return
	} else if ret == 0 {
		return
	}

	defer func() {
		if ret != 0 {
			winapi.CloseHandle(syscall.Handle(ret))
		}
	}()

	engine := core.NewSwitchEngine()

	if engine.IsTrayHidden() {
		ui.RunHeadless(engine)
	} else {
		tray := ui.NewTrayUI(engine)
		tray.Start()
	}
}
