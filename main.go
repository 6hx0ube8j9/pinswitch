//go:build windows

package main

import (
	"os"
	"pinswitch/core"
	"pinswitch/winapi"
	"pinswitch/ui"
)

func main() {
	winapi.KillOldInstances("pinswitch.exe", uint32(os.Getpid()))

	engine := core.NewSwitchEngine()
	tray := ui.NewTrayUI(engine)

	go tray.StartHotkeyListener()

	tray.Start()
}
