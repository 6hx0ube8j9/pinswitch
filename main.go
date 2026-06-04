//go:build windows

package main

import (
	"syscall"
	"pinswitch/core"
	"pinswitch/winapi"
	"pinswitch/ui"
)

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex = kernel32.NewProc("CreateMutexW")
)

func main() {
	mutexName, _ := syscall.UTF16PtrFromString("Local\\PinswitchUniqueMutexSecure")
	ret, _, _ := procCreateMutex.Call(0, 1, uintptr(unsafe.Pointer(mutexName)))
	
	if ret == 0 || syscall.GetLastError() == syscall.Errno(183) {
		return
	}

	engine := core.NewSwitchEngine()
	tray := ui.NewTrayUI(engine)

	go tray.StartHotkeyListener()

	tray.Start()
}
