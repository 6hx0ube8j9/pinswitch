package ui

import (
	_ "embed"
	"context"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
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
	
	hwnd          uintptr
	ctx           context.Context
	cancel        context.CancelFunc

	currentMode   uint32
	lastToggle    time.Time
	toggleMu      sync.Mutex
}

func NewTrayUI(engine *core.SwitchEngine) *TrayUI {
	ctx, cancel := context.WithCancel(context.Background())
	return &TrayUI{
		engine:      engine,
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
	systray.SetTooltip("Ctrl+Shift+Y 切换输入法")

	t.mFullPinyin = systray.AddMenuItem("全拼模式", "")
	t.mDoublePinyin = systray.AddMenuItem("双拼模式", "")
	systray.AddSeparator()
	t.mAutoStart = systray.AddMenuItem("开机启动", "")
	mQuit := systray.AddMenuItem("退出程序", "")

	t.SyncUI() 

	go t.StartHotkeyListener()
	go t.engine.WatchRegistry(t.ctx, func() {
		t.SyncUI()
	})

	t.mFullPinyin.Click(func() {
		t.engine.SetIMEMode(0)
		t.SyncUI()
	})
	
	t.mDoublePinyin.Click(func() {
		t.engine.SetIMEMode(1)
		t.SyncUI()
	})
	
	t.mAutoStart.Click(func() {
		t.engine.ToggleAutoStart()
		t.SyncUI()
	})
	
	mQuit.Click(func() {
		systray.Quit()
	})

	systray.SetOnClick(func(menu systray.IMenu) {
		t.toggleMode()
	})
}

func (t *TrayUI) onExit() {
	t.cancel() 

	if t.hwnd != 0 {
		winapi.PostMessage(t.hwnd, 0x0010, 0, 0)
	}
}

func (t *TrayUI) toggleMode() {
	t.toggleMu.Lock()
	defer t.toggleMu.Unlock()

	if time.Since(t.lastToggle) < 200*time.Millisecond {
		return
	}
	t.lastToggle = time.Now()

	current := t.engine.GetIMEMode()
	if t.engine.SetIMEMode(1 - current) {
		t.SyncUI() 
	}
}

func (t *TrayUI) SyncUI() {
	mode := t.engine.GetIMEMode()
	
	if mode == t.currentMode {
		return
	}
	t.currentMode = mode

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

	className := "PinswitchHotkeyWindow_Unique_Class"
	winapi.RegisterClass(className, func(hwnd uintptr, msg uint32, wparam uintptr, lparam uintptr) uintptr {
		switch msg {
		case 0x0312: 
			if wparam == 1 {
				t.toggleMode()
			}
			return 0
		case 0x0400 + 777: 
			t.toggleMode()
			return 0
		case 0x0010: 
			winapi.UnregisterHotKey(hwnd, 1)
			winapi.DestroyWindow(hwnd)
			winapi.PostQuitMessage(0)
			return 0
		}
		return winapi.DefWindowProc(hwnd, msg, wparam, lparam)
	})

	hwnd := winapi.CreateWindowEx(0, className, "PinswitchHotkey", 0, 0, 0, 0, 0, 0, 0, 0, 0)
	if hwnd == 0 {
		return
	}
	t.hwnd = hwnd

	winapi.RegisterHotKey(hwnd, 1, 0x0002|0x0004, 0x59) 

	var msg winapi.Msg
	for {
		ret := winapi.GetMessage(&msg, 0, 0, 0)
		if ret == 0 || ret == -1 {
			break
		}
		winapi.TranslateMessage(&msg)
		winapi.DispatchMessage(&msg)
	}
}
