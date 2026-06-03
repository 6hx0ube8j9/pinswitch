//go:build windows

package main

import (
	_ "embed"
	"log"
	"syscall"
	"unsafe"

	"github.com/energye/systray"
)

//go:embed icons/quan.ico
var iconQuan []byte

//go:embed icons/shuang.ico
var iconShuang []byte

var (
	mFullPinyin   *systray.MenuItem
	mDoublePinyin *systray.MenuItem

	// 动态加载 Windows 核心 DLL，彻底绕过 CGO_ENABLED=0 的编译阉割
	user32           = syscall.NewLazyDLL("user32.dll")
	procRegisterHotKey = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
	procGetMessage   = user32.NewProc("GetMessageW")

	advapi32          = syscall.NewLazyDLL("advapi32.dll")
	procRegOpenKeyEx  = advapi32.NewProc("RegOpenKeyExW")
	procRegCloseKey   = advapi32.NewProc("RegCloseKey")
	procRegQueryValueEx = advapi32.NewProc("RegQueryValueExW")
	procRegSetKeyValue = advapi32.NewProc("RegSetKeyValueW")
)

// 定义标准 Windows MSG 结构体
type tagMSG struct {
	Hwnd    syscall.Handle
	Message uint32
	Wparam  uintptr
	Lparam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

const (
	registryPath  = `SOFTWARE\Microsoft\InputMethod\Settings\CHS`
	registryValue = "Enable Double Pinyin"

	HKEY_CURRENT_USER = 0x80000001
	KEY_QUERY_VALUE   = 0x0001
	REG_DWORD         = 4

	HOTKEY_ID           = 1
	MY_HOTKEY_MODIFIERS = 0x0002 | 0x0004 // Ctrl + Shift
	MY_HOTKEY_VK        = 0x58             // 'X'
)

func main() {
	go startHotkeyListener()
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTitle("输入法切换")
	systray.SetTooltip("")

	mFullPinyin = systray.AddMenuItem("全拼模式", "")
	mDoublePinyin = systray.AddMenuItem("双拼模式", "")

	mFullPinyin.Click(func() {
		toggleMode(0)
	})
	mDoublePinyin.Click(func() {
		toggleMode(1)
	})

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("退出程序", "")
	mQuit.Click(func() {
		systray.Quit()
	})

	currentMode := getDoublePinyinRegistry()
	updateUI(currentMode)

	systray.SetOnClick(func(menu systray.IMenu) {
		nowMode := getDoublePinyinRegistry()
		toggleMode(1 - nowMode)
	})
}

func onExit() {
	// 调用原生动态链接库注销快捷键
	procUnregisterHotKey.Call(0, HOTKEY_ID)
}

func toggleMode(targetMode uint32) {
	setDoublePinyinRegistry(targetMode)
	updateUI(targetMode)
}

func updateUI(mode uint32) {
	if mode == 1 {
		systray.SetIcon(iconShuang)
		mDoublePinyin.Check()
		mDoublePinyin.Disable()
		mFullPinyin.Uncheck()
		mFullPinyin.Enable()
	} else {
		systray.SetIcon(iconQuan)
		mFullPinyin.Check()
		mFullPinyin.Disable()
		mDoublePinyin.Uncheck()
		mDoublePinyin.Enable()
	}
}

func startHotkeyListener() {
	// 调用动态链接库注册全局快捷键
	r1, _, _ := procRegisterHotKey.Call(0, HOTKEY_ID, MY_HOTKEY_MODIFIERS, MY_HOTKEY_VK)
	if r1 == 0 {
		log.Println("全局快捷键注册失败")
		return
	}

	var msg tagMSG
	for {
		r1, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r1) <= 0 {
			break
		}

		if msg.Message == 0x0312 && msg.Wparam == HOTKEY_ID {
			nowMode := getDoublePinyinRegistry()
			toggleMode(1 - nowMode)
		}
	}
}

func getDoublePinyinRegistry() uint32 {
	var hKey syscall.Handle

	pathPtr, _ := syscall.UTF16PtrFromString(registryPath)
	valuePtr, _ := syscall.UTF16PtrFromString(registryValue)

	// 调用原生的 RegOpenKeyExW
	r1, _, _ := procRegOpenKeyEx.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(pathPtr)), 0, KEY_QUERY_VALUE, uintptr(unsafe.Pointer(&hKey)))
	if r1 != 0 {
		return 0
	}
	defer procRegCloseKey.Call(uintptr(hKey))

	var value uint32
	var size uint32 = 4
	var valType uint32

	// 调用原生的 RegQueryValueExW
	r1, _, _ = procRegQueryValueEx.Call(uintptr(hKey), uintptr(unsafe.Pointer(valuePtr)), 0, uintptr(unsafe.Pointer(&valType)), uintptr(unsafe.Pointer(&value)), uintptr(unsafe.Pointer(&size)))
	if r1 != 0 {
		return 0
	}

	return value
}

func setDoublePinyinRegistry(value uint32) {
	pathPtr, _ := syscall.UTF16PtrFromString(registryPath)
	valuePtr, _ := syscall.UTF16PtrFromString(registryValue)

	// 调用原生的 RegSetKeyValueW 
	procRegSetKeyValue.Call(
		HKEY_CURRENT_USER,
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(valuePtr)),
		REG_DWORD,
		uintptr(unsafe.Pointer(&value)),
		4,
	)
}
