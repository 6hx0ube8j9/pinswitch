//go:build windows

package main

import (
	"pinswitch/core"
	"pinswitch/ui"
	"pinswitch/winapi"
	"syscall"
)

func main() {
	ret, err := winapi.CreateMutex("Local\\PinswitchUniqueMutexSecure")
	
	if err == syscall.Errno(183) {
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
	tray := ui.NewTrayUI(engine)
	tray.Start()
}
