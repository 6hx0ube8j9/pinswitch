package ui

import (
	_ "embed"
	"context"
	"runtime"
	"pinswitch/core"
	"pinswitch/winapi"
	"github.com/energye/systray"
)

//go:embed icons/quan.ico
var iconQuan []byte

//go:embed icons/shuang.ico
var iconShuang []byte

type TrayUI struct {
	engine        *core.SwitchEngine
	mFullPinyin   *systray.MenuItem
	mDoublePinyin *systray.MenuItem
	mAutoStart    *systray.MenuItem
	hwnd          uintptr
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewTrayUI(engine *core.SwitchEngine) *TrayUI {
	ctx, cancel := context.WithCancel(context.Background())
	return &TrayUI{engine: engine, ctx: ctx, cancel: cancel}
}

func (t *TrayUI) Start() {
	systray.Run(t.onReady, t.onExit)
}

func (t *TrayUI) onReady() {
	systray.SetTitle("pinswitch")
	systray.SetTooltip("Ctrl+Shift+Y 切换输入法")

	t.mFullPinyin = systray.AddMenuItem("全拼模式", "")
	t.mDoublePinyin = systray.AddMenuItem("双拼模式", "")
	systray.AddSeparator()
	t.mAutoStart = systray.AddMenuItem("开机自启动", "")
	mQuit := systray.AddMenuItem("退出程序", "")

	t.mFullPinyin.Click(func() { t.engine.SetIMEMode(0); t.SyncUI() })
	t.mDoublePinyin.Click(func() { t.engine.SetIMEMode(1); t.SyncUI() })
	t.mAutoStart.Click(func() { t.engine.ToggleAutoStart(); t.SyncUI() })
	mQuit.Click(func() { systray.Quit() })

	systray.SetOnClick(func(menu systray.IMenu) {
		current := t.engine.GetIMEMode()
		if t.engine.SetIMEMode(1 - current) {
			t.SyncUI()
		}
	})

	t.SyncUI()
	go t.engine.WatchRegistry(t.ctx, t.SyncUI)
	go t.StartHotkeyListener()
}

func (t *TrayUI) onExit() {
	t.cancel()
	if t.hwnd != 0 {
		winapi.DestroyWindow(t.hwnd)
	}
}

func (t *TrayUI) SyncUI() {
	mode := t.engine.GetIMEMode()
	if mode == 1 {
		systray.SetIcon(iconShuang)
		t.mDoublePinyin.Check()
		t.mDoublePinyin.Disable()
		t.mFullPinyin.Uncheck()
		t.mFullPinyin.Enable()
	} else {
		systray.SetIcon(iconQuan)
		t.mFullPinyin.Check()
		t.mFullPinyin.Disable()
		t.mDoublePinyin.Uncheck()
		t.mDoublePinyin.Enable()
	}

	if t.engine.IsAutoStart() {
		t.mAutoStart.Check()
	} else {
		t.mAutoStart.Uncheck()
	}
}

func (t *TrayUI) StartHotkeyListener() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	className := "PinswitchHotkeyWindow"
	winapi.RegisterClass(className, func(hwnd uintptr, msg uint32, wparam uintptr, lparam uintptr) uintptr {
		switch msg {
		case 0x0312: // WM_HOTKEY
			if wparam == 1 {
				current := t.engine.GetIMEMode()
				if t.engine.SetIMEMode(1 - current) {
					t.SyncUI()
				}
			}
			return 0
		case 0x0002: // WM_DESTROY
			winapi.UnregisterHotKey(hwnd, 1)
			winapi.PostQuitMessage(0)
			return 0
		}
		return winapi.DefWindowProc(hwnd, msg, wparam, lparam)
	})

	t.hwnd = winapi.CreateWindowEx(className)
	if t.hwnd == 0 {
		return
	}

	if !winapi.RegisterHotKey(t.hwnd, 1, 0x0002|0x0004, 0x59) {
		return
	}

	var msg winapi.TagMSG
	for {
		res := winapi.GetMessage(&msg)
		if res <= 0 {
			break
		}
		winapi.TranslateMessage(&msg)
		winapi.DispatchMessage(&msg)
	}
}
