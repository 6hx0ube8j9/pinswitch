//go:build windows

package main

import (
	"os/exec"
	"syscall"
	"time"
	"pinswitch/core"
	"pinswitch/ui"
	"pinswitch/winapi"
)

func KillOldInstances() {
	cmd := exec.Command("taskkill", "/F", "/IM", "pinswitch.exe")
	cmd.SysProcAttr = &exec.Cmd{
		SysProcAttr: &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		},
	}.SysProcAttr
	_ = cmd.Run()
}

func main() {
	ret, err := winapi.CreateMutex("Local\\PinswitchUniqueMutexSecure")
	if ret == 0 || err == syscall.Errno(183) {
		KillOldInstances()
		time.Sleep(200 * time.Millisecond)
		return
	}

	engine := core.NewSwitchEngine()
	tray := ui.NewTrayUI(engine)
	tray.Start()
}
