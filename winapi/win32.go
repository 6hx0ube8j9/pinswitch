var (
	// 在 user32 声明中增加以下核心 API
	procSetWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
)

const (
	WH_KEYBOARD_LL = 13
	WM_KEYDOWN     = 0x0100
	VK_SHIFT       = 0x10
	VK_CONTROL     = 0x11
)

// KBDLLHOOKSTRUCT 键盘钩子结构体
type KBDLLHOOKSTRUCT struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

// 暴露出底层的原生调用
func SetWindowsHookEx(idHook int, lpfn uintptr, hmod uintptr, dwThreadId uint32) syscall.Handle {
	r1, _, _ := procSetWindowsHookExW.Call(uintptr(idHook), lpfn, hmod, uintptr(dwThreadId))
	return syscall.Handle(r1)
}

func UnhookWindowsHookEx(hhk syscall.Handle) bool {
	r1, _, _ := procUnhookWindowsHookEx.Call(uintptr(hhk))
	return r1 != 0
}

func CallNextHookEx(hhk syscall.Handle, nCode int, wParam uintptr, lParam uintptr) uintptr {
	r1, _, _ := procCallNextHookEx.Call(uintptr(hhk), uintptr(nCode), wParam, lParam)
	return r1
}

// 补充一个获取异步按键状态的 API，用来精准判断 Ctrl 和 Shift 是否同时按下
var procGetAsyncKeystate = user32.NewProc("GetAsyncKeyState")
func GetAsyncKeyState(vKey int) int16 {
	r1, _, _ := procGetAsyncKeystate.Call(uintptr(vKey))
	return int16(r1)
}
