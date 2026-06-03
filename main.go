//go:build windows

package main

import (
	_ "embed"
	"log"
	"unsafe"

	"github.com/energye/systray"
	"golang.org/x/sys/windows"
)

//go:embed icons/quan.ico
var iconQuan []byte

//go:embed icons/shuang.ico
var iconShuang []byte

var (
	mFullPinyin   *systray.MenuItem
	mDoublePinyin *systray.MenuItem
	user32        = windows.NewLazySystemDLL("user32.dll")
	procGetMessage = user32.NewProc("GetMessageW")
)

const (
	registryPath  = `SOFTWARE\Microsoft\InputMethod\Settings\CHS`
	registryValue = "Enable Double Pinyin"
	
	HOTKEY_ID = 1
	// 默认快捷键：Ctrl + Shift + X
	MY_HOTKEY_MODIFIERS = 0x0002 | 0x0004 
	MY_HOTKEY_VK        = 0x58             
)

func main() {
	go startHotkeyListener()
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTitle("输入法切换")
	// 移除了托盘图标的悬停提示
	systray.SetTooltip("") 

	// 移除了“全拼模式”和“双拼模式”的鼠标悬停提示信息（第二个参数传空字符串 ""）
	mFullPinyin = systray.AddMenuItem("全拼模式", "")
	mDoublePinyin = systray.AddMenuItem("双拼模式", "")

	mFullPinyin.Click(func() {
		toggleMode(0)
	})
	mDoublePinyin.Click(func() {
		toggleMode(1)
	})

	systray.AddSeparator()

	// 移除了“退出程序”的鼠标悬停提示信息
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
	windows.UnregisterHotKey(0, HOTKEY_ID)
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
	err := windows.RegisterHotKey(0, HOTKEY_ID, MY_HOTKEY_MODIFIERS, MY_HOTKEY_VK)
	if err != nil {
		log.Println("全局快捷键注册失败:", err)
		return
	}

	var msg windows.Msg
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
	var hKey windows.Handle

	pathPtr, err := windows.UTF16PtrFromString(registryPath)
	if err != nil {
		return 0
	}
	valuePtr, err := windows.UTF16PtrFromString(registryValue)
	if err != nil {
		return 0
	}

	err = windows.RegOpenKeyEx(windows.HKEY_CURRENT_USER, pathPtr, 0, windows.KEY_QUERY_VALUE, &hKey)
	if err != nil {
		log.Println("打开注册表失败:", err)
		return 0
	}
	defer windows.RegCloseKey(hKey)

	var value uint32
	var size uint32 = uint32(unsafe.Sizeof(value))
	var valType uint32

	err = windows.RegQueryValueEx(hKey, valuePtr, nil, &valType, (*byte)(unsafe.Pointer(&value)), &size)
	if err != nil {
		return 0
	}

	return value
}

func setDoublePinyinRegistry(value uint32) {
	pathPtr, err := windows.UTF16PtrFromString(registryPath)
	if err != nil {
		return
	}
	valuePtr, err := windows.UTF16PtrFromString(registryValue)
	if err != nil {
		return
	}

	err = windows.RegSetKeyValue(
		windows.HKEY_CURRENT_USER,
		pathPtr,
		valuePtr,
		windows.REG_DWORD,
		(*byte)(unsafe.Pointer(&value)),
		uint32(unsafe.Sizeof(value)),
	)
	if err != nil {
		log.Println("修改注册表失败:", err)
	}
}
