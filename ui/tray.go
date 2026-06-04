package ui

import (
	_ "embed"
	"runtime"
	"time"
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
}

func NewTrayUI(engine *core.SwitchEngine) *TrayUI {
	return &TrayUI{engine: engine}
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
		t.engine.SetIMEMode(1 - t.engine.GetIMEMode())
		t.SyncUI() 
	})

	t.SyncUI()
	go t.engine.WatchRegistry(t.SyncUI)
}

func (t *TrayUI) onExit() {
	winapi.UnregisterHotKey(1)
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

	if !winapi.RegisterHotKey(1, 0x0002|0x0004, 0x59) {
		println("❌ [Error] Ctrl+Shift+Y 热键注册失败，可能被占用！")
		return
	}

	var msg winapi.TagMSG
	var lastTrigger time.Time
	const cooldown = 220 * time.Millisecond

	for {
		res := winapi.GetMessage(&msg)
		if res <= 0 {
			break
		}

		if msg.Message == 0x0312 && msg.Wparam == 1 {
			now := time.Now()
			if now.Sub(lastTrigger) < cooldown {
				continue
			}
			lastTrigger = now

			currentMode := t.engine.GetIMEMode()
			t.engine.SetIMEMode(1 - currentMode)
			t.SyncUI()
		}
		time.Sleep(1 * time.Millisecond)
	}
}
