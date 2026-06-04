package ui

import (
	_ "embed"
	"runtime"
	"syscall"
	"time"
	"unsafe"
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
	hHook         syscall.Handle
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
	if t.hHook != 0 {
		winapi.UnhookWindowsHookEx(t.hHook)
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

	var lastTrigger time.Time
	const cooldown = 220 * time.Millisecond

	keyboardProc := func(nCode int, wParam uintptr, lParam uintptr) uintptr {
		if nCode >= 0 && wParam == winapi.WM_KEYDOWN {
			kbd := (*winapi.KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
			
			if kbd.VkCode == 0x59 {
				ctrlDown := winapi.GetAsyncKeyState(winapi.VK_CONTROL) < 0
				shiftDown := winapi.GetAsyncKeyState(winapi.VK_SHIFT) < 0
				
				if ctrlDown && shiftDown {
					now := time.Now()
					if now.Sub(lastTrigger) >= cooldown {
						lastTrigger = now
						
						go func() {
							currentMode := t.engine.GetIMEMode()
							t.engine.SetIMEMode(1 - currentMode)
							t.SyncUI()
						}()
					}
				}
			}
		}
		return winapi.CallNextHookEx(t.hHook, nCode, wParam, lParam)
	}

	t.hHook = winapi.SetWindowsHookEx(
		winapi.WH_KEYBOARD_LL,
		syscall.NewCallback(keyboardProc),
		0,
		0,
	)

	if t.hHook == 0 {
		println("❌ [Error] 全局键盘钩子挂载失败！")
		return
	}

	var msg winapi.TagMSG
	for winapi.GetMessage(&msg) > 0 {
	}
}
