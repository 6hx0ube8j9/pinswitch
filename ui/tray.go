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
	hHook         syscall.Handle // 保存钩子句柄以便退出时释放
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
	// 程序退出时，务必卸载全局键盘钩子，还操作系统一片纯净
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

// 【终极原生方案】基于 Windows 核心低级键盘钩子，彻底无视疯狂连击
func (t *TrayUI) StartHotkeyListener() {
	// 锁定该协程到固定的操作系统线程（Windows 钩子要求必须有稳固的线程上下文）
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var lastTrigger time.Time
	const cooldown = 220 * time.Millisecond // 硬件级防抖时间

	// 编写 Windows 键盘事件回调函数
	keyboardProc := func(nCode int, wParam uintptr, lParam uintptr) uintptr {
		if nCode >= 0 && wParam == winapi.WM_KEYDOWN {
			kbd := (*winapi.KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
			
			// 0x59 是 'Y' 键
			if kbd.VkCode == 0x59 {
				// 精准检查此时此刻 Ctrl 和 Shift 是否同时处于按下状态
				ctrlDown := winapi.GetAsyncKeyState(winapi.VK_CONTROL) & 0x8000 != 0
				shiftDown := winapi.GetAsyncKeyState(winapi.VK_SHIFT) & 0x8000 != 0
				
				if ctrlDown && shiftDown {
					now := time.Now()
					if now.Sub(lastTrigger) >= cooldown {
						lastTrigger = now
						
						// 🌟 核心安全设计：使用 go 关键字开启新协程去处理复杂的 I/O 读写
						// 让 Windows 的键盘回调在 1 微秒内瞬间返回，彻底杜绝任何卡死崩溃！
						go func() {
							currentMode := t.engine.GetIMEMode()
							t.engine.SetIMEMode(1 - currentMode)
							t.SyncUI()
						}()
					}
				}
			}
		}
		// 将消息传递给系统中的下一个钩子
		return winapi.CallNextHookEx(t.hHook, nCode, wParam, lParam)
	}

	// 注册全局底层键盘钩子
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

	// 钩子需要一个标准的底层消息循环来维持其生命周期
	var msg winapi.TagMSG
	for winapi.GetMessage(&msg) > 0 {
		// 这里的消息循环极度干净，不处理任何业务，只负责维持钩子存活
	}
}
