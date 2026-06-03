//go:build windows

package main

import (
	_ "embed"
	"log"
	"unsafe"

	"github.com/energye/systray"
	"golang.org/x/sys/windows"
)

//go:embed icons/app.ico
var appIcon []byte

// 定义全局变量以便在不同函数中控制菜单状态
var (
	mFullPinyin   *systray.MenuItem
	mDoublePinyin *systray.MenuItem
)

const (
	registryPath  = `SOFTWARE\Microsoft\InputMethod\Settings\CHS`
	registryValue = "Enable Double Pinyin"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(appIcon)
	systray.SetTitle("输入法切换")
	systray.SetTooltip("左键点击切换模式，右键弹出菜单")

	// 1. 创建右键菜单项
	mFullPinyin = systray.AddMenuItem("全拼模式", "切换到全拼")
	mDoublePinyin = systray.AddMenuItem("双拼模式", "切换到双拼")

	// 绑定右键菜单点击事件
	mFullPinyin.Click(func() {
		setDoublePinyinRegistry(0)
		updateMenuState(0)
	})
	mDoublePinyin.Click(func() {
		setDoublePinyinRegistry(1)
		updateMenuState(1)
	})

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("退出程序", "关闭工具")
	mQuit.Click(func() {
		systray.Quit()
	})

	// 2. 初始化：启动时读取当前注册表状态并更新 UI
	currentMode := getDoublePinyinRegistry()
	updateMenuState(currentMode)

	// 3. 绑定左键点击事件：直接切换模式
	systray.SetOnClick(func(menu systray.IMenu) {
		// 读取当前状态，取反进行切换
		nowMode := getDoublePinyinRegistry()
		var targetMode uint32 = 1
		if nowMode == 1 {
			targetMode = 0
		}

		setDoublePinyinRegistry(targetMode)
		updateMenuState(targetMode)
	})
}

func onExit() {}

// getDoublePinyinRegistry 使用官方标准 windows 库读取注册表
func getDoublePinyinRegistry() uint32 {
	var hKey windows.Handle

	// 将 Go 字符串安全转换为 Windows 的 UTF16 指针
	pathPtr, err := windows.UTF16PtrFromString(registryPath)
	if err != nil {
		return 0
	}
	valuePtr, err := windows.UTF16PtrFromString(registryValue)
	if err != nil {
		return 0
	}

	// 调用原生 Win32 API: RegOpenKeyEx
	err = windows.RegOpenKeyEx(windows.HKEY_CURRENT_USER, pathPtr, 0, windows.KEY_QUERY_VALUE, &hKey)
	if err != nil {
		log.Println("打开注册表失败:", err)
		return 0 // 默认返回全拼
	}
	defer windows.RegCloseKey(hKey)

	var value uint32
	var size uint32 = uint32(unsafe.Sizeof(value))
	var valType uint32

	// 调用原生 Win32 API: RegQueryValueEx
	err = windows.RegQueryValueEx(hKey, valuePtr, nil, &valType, (*byte)(unsafe.Pointer(&value)), &size)
	if err != nil {
		// 键值不存在，Windows 默认也是全拼
		return 0
	}

	return value
}

// setDoublePinyinRegistry 使用官方标准 windows 库修改注册表
func setDoublePinyinRegistry(value uint32) {
	var hKey windows.Handle

	pathPtr, err := windows.UTF16PtrFromString(registryPath)
	if err != nil {
		return
	}
	valuePtr, err := windows.UTF16PtrFromString(registryValue)
	if err != nil {
		return
	}

	// 调用原生 Win32 API: RegOpenKeyEx
	err = windows.RegOpenKeyEx(windows.HKEY_CURRENT_USER, pathPtr, 0, windows.KEY_SET_VALUE, &hKey)
	if err != nil {
		log.Println("打开注册表失败:", err)
		return
	}
	defer windows.RegCloseKey(hKey)

	var size uint32 = uint32(unsafe.Sizeof(value))

	// 调用原生 Win32 API: RegSetValueEx
	err = windows.RegSetValueEx(hKey, valuePtr, 0, windows.REG_DWORD, (*byte)(unsafe.Pointer(&value)), size)
	if err != nil {
		log.Println("修改注册表失败:", err)
	}
}

// updateMenuState 根据模式更新菜单的勾选和置灰状态
func updateMenuState(mode uint32) {
	if mode == 1 {
		mDoublePinyin.Check()
		mDoublePinyin.Disable()

		mFullPinyin.Uncheck()
		mFullPinyin.Enable()
	} else {
		mFullPinyin.Check()
		mFullPinyin.Disable()

		mDoublePinyin.Uncheck()
		mDoublePinyin.Enable()
	}
}
