//go:build windows

package main

import (
	"context"
	_ "embed"
	"os"
	"os/signal"
	"syscall"

	"github.com/energye/systray"
)

//go:embed icons/quan.ico
var iconQuan []byte

//go:embed icons/shuang.ico
var iconShuang []byte

type TrayUI struct {
	brain         *SwitchBrain
	mFullPinyin   *systray.MenuItem
	mDoublePinyin *systray.MenuItem
	mAutoStart    *systray.MenuItem
	ctx           context.Context
	cancel        context.CancelFunc
	currentMode   uint32
}

func NewTrayUI(brain *SwitchBrain) *TrayUI {
	ctx, cancel := context.WithCancel(context.Background())
	return &TrayUI{
		brain:       brain,
		ctx:         ctx,
		cancel:      cancel,
		currentMode: 999,
	}
}

func (t *TrayUI) Start() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigs:
			systray.Quit()
		case <-t.ctx.Done():
		}
	}()

	systray.Run(t.onReady, t.onExit)
}

func (t *TrayUI) onReady() {
	systray.SetTitle("pinswitch")

	t.mFullPinyin = systray.AddMenuItem("全拼输入", "")
	t.mDoublePinyin = systray.AddMenuItem("双拼输入", "")
	systray.AddSeparator()
	t.mAutoStart = systray.AddMenuItem("开机启动", "")
	systray.AddSeparator()
	mHelp := systray.AddMenuItem("快捷键说明", "")
	mQuit := systray.AddMenuItem("退出程序", "")

	t.SyncUI()

	go t.brain.WatchRegistry(t.ctx, func() {
		t.SyncUI()
	})

	t.mFullPinyin.Click(func() {
		t.brain.SetIMEMode(0)
	})

	t.mDoublePinyin.Click(func() {
		t.brain.SetIMEMode(1)
	})

	t.mAutoStart.Click(func() {
		t.brain.ToggleAutoStart()
		t.SyncUI() 
	})

	mHelp.Click(func() {
		helpText := "【快捷键说明】\n\n" +
			"Shift+Ctrl+Y：切换全拼/双拼\n" +
			"Shift+Ctrl+Win+Y：显示/隐藏托盘图标\n" +
			"Shift+双击程序：显示/隐藏托盘图标"
		MessageBox(0, helpText, "Pinswitch", 0x00000040)
	})

	mQuit.Click(func() {
		systray.Quit()
	})

	systray.SetOnClick(func(menu systray.IMenu) {
		t.brain.ToggleMode()
	})
}

func (t *TrayUI) onExit() {
	t.cancel()
	t.brain.Close()
}

func (t *TrayUI) SyncUI() {
	mode := t.brain.GetIMEMode()

	if mode == t.currentMode {
		return
	}
	t.currentMode = mode

	if mode == 1 {
		systray.SetIcon(iconShuang)
		systray.SetTooltip("当前: 双拼模式")
		t.mDoublePinyin.Check()
		t.mDoublePinyin.Disable()
		t.mFullPinyin.Uncheck()
		t.mFullPinyin.Enable()
	} else {
		systray.SetIcon(iconQuan)
		systray.SetTooltip("当前: 全拼模式")
		t.mFullPinyin.Check()
		t.mFullPinyin.Disable()
		t.mDoublePinyin.Uncheck()
		t.mDoublePinyin.Enable()
	}

	if t.brain.IsAutoStart() {
		t.mAutoStart.Check()
	} else {
		t.mAutoStart.Uncheck()
	}
}
