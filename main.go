//go:build windows

package main

import (
	"syscall"
	"time"
	"pinswitch/core"
	"pinswitch/ui"
	"pinswitch/winapi"
)

func main() {
	ret, err := winapi.CreateMutex("Local\\PinswitchUniqueMutexSecure")
	if ret == 0 || err == syscall.Errno(183) {
		oldHwnd := winapi.FindWindow("PinswitchHotkeyWindow_Unique_Class")
		if oldHwnd != 0 {
			winapi.SendMessage(oldHwnd, 0x0010, 0, 0)
			time.Sleep(250 * time.Millisecond)
		}
		ret, err = winapi.CreateMutex("Local\\PinswitchUniqueMutexSecure")
		if ret == 0 || err == syscall.Errno(183) {
			return
		}
	}

	engine := core.NewSwitchEngine()
	tray := ui.NewTrayUI(engine)
	tray.Start()
}
