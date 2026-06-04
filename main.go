//go:build windows

package main

import (
	_ "embed"
	"log"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"
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
	mAutoStart    *systray.MenuItem

	// 【核心状态锁】防止自己修改注册表时触发监听死循环
	isWritingRegistry bool = false

	// user32 核心库
	user32               = syscall.NewLazyDLL("user32.dll")
	procRegisterHotKey   = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
	procGetMessage       = user32.NewProc("GetMessageW")

	// advapi32 注册表库
	advapi32            = syscall.NewLazyDLL("advapi32.dll")
	procRegOpenKeyEx    = advapi32.NewProc("RegOpenKeyExW")
	procRegCloseKey     = advapi32.NewProc("RegCloseKey")
	procRegQueryValueEx = advapi32.NewProc("RegQueryValueExW")
	procRegSetKeyValue  = advapi32.NewProc("RegSetKeyValueW")
	procRegDeleteValue  = advapi32.NewProc("RegDeleteValueW")
	
	// 【新增高级 API】注册表主动变更通知
	procRegNotifyChangeKeyValue = advapi32.NewProc("RegNotifyChangeKeyValue")

	// kernel32 系统核心库
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = kernel32.NewProc("Process32FirstW")
	procProcess32Next            = kernel32.NewProc("Process32NextW")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procTerminateProcess         = kernel32.NewProc("TerminateProcess")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
	procCreateEvent              = kernel32.NewProc("CreateEventW")
	procWaitForSingleObject      = kernel32.NewProc("WaitForSingleObject")
	procResetEvent               = kernel32.NewProc("ResetEvent")
)

type tagMSG struct {
	Hwnd    syscall.Handle
	Message uint32
	Wparam  uintptr
	Lparam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type tagPROCESSENTRY32W struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

const (
	registryPath  = `SOFTWARE\Microsoft\InputMethod\Settings\CHS`
	registryValue = "Enable Double Pinyin"

	autoStartPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	autoStartKey  = "IMESwitcherAutoStart"

	HKEY_CURRENT_USER = 0x80000001
	KEY_QUERY_VALUE   = 0x0001
	KEY_NOTIFY        = 0x0010
	REG_SZ            = 1
	REG_DWORD         = 4

	HOTKEY_ID           = 1
	MY_HOTKEY_MODIFIERS = 0x0002 | 0x0004 // Ctrl + Shift
	MY_HOTKEY_VK        = 0x59             // 'Y'

	TH32CS_SNAPPROCESS = 0x00000002
	PROCESS_TERMINATE  = 0x0001
	
	// 注册表变更通知常量
	REG_NOTIFY_CHANGE_LAST_SET = 0x00000004
	WAIT_OBJECT_0              = 0x00000000
	INFINITE                   = 0xFFFFFFFF
)

func main() {
	// 纯原生精确防多开
	killOldInstances("ime-switcher.exe")

	// 启动全局热键监听
	go startHotkeyListener()
	
	// 启动托盘管理
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTitle("输入法切换")
	systray.SetTooltip("Ctrl+Shift+Y 切换输入法")

	mFullPinyin = systray.AddMenuItem("全拼模式", "")
	mDoublePinyin = systray.AddMenuItem("双拼模式", "")

	mFullPinyin.Click(func() {
		toggleMode(0)
	})
	mDoublePinyin.Click(func() {
		toggleMode(1)
	})

	systray.AddSeparator()

	// 开机自启动选项
	mAutoStart = systray.AddMenuItem("开机自启动", "")
	mAutoStart.Click(func() {
		if isAutoStartEnabled() {
			disableAutoStart()
			mAutoStart.Uncheck()
		} else {
			enableAutoStart()
			mAutoStart.Check()
		}
	})

	mQuit := systray.AddMenuItem("退出程序", "")
	mQuit.Click(func() {
		systray.Quit()
	})

	// 托盘左键点击事件
	systray.SetOnClick(func(menu systray.IMenu) {
		nowMode := getDoublePinyinRegistry()
		toggleMode(1 - nowMode)
	})

	// 首次主动对齐一次系统状态
	syncRegistryToUI()

	// 【优雅合并】启动由 Windows 内核事件驱动的“纯原生零开销自对齐监听”
	go startNativeRegistryWatcher()
}

func onExit() {
	procUnregisterHotKey.Call(0, HOTKEY_ID)
}

// 写入输入法模式，通过锁防止触发自己的通知
func toggleMode(targetMode uint32) {
	isWritingRegistry = true
	setDoublePinyinRegistry(targetMode)
	isWritingRegistry = false
	
	syncRegistryToUI()
}

// 高效对齐函数：将系统最真实的值投射到托盘 UI 上
func syncRegistryToUI() {
	currentIMEMode := getDoublePinyinRegistry()
	if currentIMEMode == 1 {
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

	if isAutoStartEnabled() {
		mAutoStart.Check()
	} else {
		mAutoStart.Uncheck()
	}
}

// 【终极优雅】利用纯 Windows 核心通知事件实现的对齐监听器
func startNativeRegistryWatcher() {
	var hKey syscall.Handle
	pathPtr, _ := syscall.UTF16PtrFromString(registryPath)

	// 以通知和查询权限打开注册表
	r1, _, _ := procRegOpenKeyEx.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(pathPtr)), 0, KEY_NOTIFY|KEY_QUERY_VALUE, uintptr(unsafe.Pointer(&hKey)))
	if r1 != 0 {
		return
	}
	defer procRegCloseKey.Call(uintptr(hKey))

	// 创建一个 Windows 事件同步内核对象
	hEvent, _, _ := procCreateEvent.Call(0, 1, 0, 0)
	if hEvent == 0 {
		return
	}
	defer procCloseHandle.Call(hEvent)

	for {
		// 向 Windows 注册：当该注册表项的“最后写入时间”改变时，触发 hEvent 信号
		procRegNotifyChangeKeyValue.Call(
			uintptr(hKey),
			0, // 不监听子项
			REG_NOTIFY_CHANGE_LAST_SET,
			hEvent,
			1, // 异步非阻塞等待
		)

		// 挂起协程，静默等待系统内核事件通知（不占用任何 CPU，由底层操作系统的内核唤醒）
		res, _, _ := procWaitForSingleObject.Call(hEvent, INFINITE)
		if res == WAIT_OBJECT_0 {
			procResetEvent.Call(hEvent)

			// 如果改变是由我们程序自己通过热键或托盘主动写入的，直接忽略，防止无限死循环
			if isWritingRegistry {
				continue
			}

			// 如果是第三方或者系统设置偷偷改的，立刻抓取并对齐到托盘 UI
			syncRegistryToUI()
		}
		
		// 严防死守：防止底层句柄因并发被 Go 运行时意外释放
		runtime.KeepAlive(hKey)
	}
}

func startHotkeyListener() {
	r1, _, _ := procRegisterHotKey.Call(0, HOTKEY_ID, MY_HOTKEY_MODIFIERS, MY_HOTKEY_VK)
	if r1 == 0 {
		log.Println("全局快捷键注册失败")
		return
	}

	var msg tagMSG
	var lastTriggerTime time.Time
	const cooldown = 250 * time.Millisecond

	for {
		r1, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r1) <= 0 {
			break
		}

		if msg.Message == 0x0312 && msg.Wparam == HOTKEY_ID {
			now := time.Now()
			if now.Sub(lastTriggerTime) < cooldown {
				continue
			}
			lastTriggerTime = now

			nowMode := getDoublePinyinRegistry()
			toggleMode(1 - nowMode)
		}
	}
}

func killOldInstances(targetExeName string) {
	currentPID := uint32(os.Getpid())
	snapshot, _, _ := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if syscall.Handle(snapshot) == syscall.InvalidHandle {
		return
	}
	defer procCloseHandle.Call(snapshot)

	var entry tagPROCESSENTRY32W
	entry.Size = uint32(unsafe.Sizeof(entry))

	ret, _, _ := procProcess32First.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	for ret != 0 {
		exeName := syscall.UTF16ToString(entry.ExeFile[:])
		if strings.EqualFold(exeName, targetExeName) && entry.ProcessID != currentPID {
			hProcess, _, _ := procOpenProcess.Call(PROCESS_TERMINATE, 0, uintptr(entry.ProcessID))
			if hProcess != 0 {
				procTerminateProcess.Call(hProcess, 0)
				procCloseHandle.Call(hProcess)
			}
		}
		ret, _, _ = procProcess32Next.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	}
}

// ==================== 注册表读写底层 ====================

func isAutoStartEnabled() bool {
	var hKey syscall.Handle
	pathPtr, _ := syscall.UTF16PtrFromString(autoStartPath)
	valuePtr, _ := syscall.UTF16PtrFromString(autoStartKey)

	r1, _, _ := procRegOpenKeyEx.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(pathPtr)), 0, KEY_QUERY_VALUE, uintptr(unsafe.Pointer(&hKey)))
	if r1 != 0 {
		return false
	}
	defer procRegCloseKey.Call(uintptr(hKey))

	var size uint32 = 520
	var valType uint32
	buf := make([]uint16, 260)

	r1, _, _ = procRegQueryValueEx.Call(uintptr(hKey), uintptr(unsafe.Pointer(valuePtr)), 0, uintptr(unsafe.Pointer(&valType)), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	return r1 == 0
}

func enableAutoStart() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	pathPtr, _ := syscall.UTF16PtrFromString(autoStartPath)
	valuePtr, _ := syscall.UTF16PtrFromString(autoStartKey)
	
	exePathStr := `"` + exePath + `"`
	exePathPtr, _ := syscall.UTF16PtrFromString(exePathStr)
	size := uint32((len(exePathStr) + 1) * 2)

	procRegSetKeyValue.Call(
		HKEY_CURRENT_USER,
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(valuePtr)),
		REG_SZ,
		uintptr(unsafe.Pointer(exePathPtr)),
		uintptr(size),
	)
}

func disableAutoStart() {
	var hKey syscall.Handle
	pathPtr, _ := syscall.UTF16PtrFromString(autoStartPath)
	valuePtr, _ := syscall.UTF16PtrFromString(autoStartKey)

	const KEY_SET_VALUE = 0x0002
	r1, _, _ := procRegOpenKeyEx.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(pathPtr)), 0, KEY_SET_VALUE, uintptr(unsafe.Pointer(&hKey)))
	if r1 == 0 {
		procRegDeleteValue.Call(uintptr(hKey), uintptr(unsafe.Pointer(valuePtr)))
		procRegCloseKey.Call(uintptr(hKey))
	}
}

func getDoublePinyinRegistry() uint32 {
	var hKey syscall.Handle
	pathPtr, _ := syscall.UTF16PtrFromString(registryPath)
	valuePtr, _ := syscall.UTF16PtrFromString(registryValue)

	r1, _, _ := procRegOpenKeyEx.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(pathPtr)), 0, KEY_QUERY_VALUE, uintptr(unsafe.Pointer(&hKey)))
	if r1 != 0 {
		return 0
	}
	defer procRegCloseKey.Call(uintptr(hKey))

	var value uint32
	var size uint32 = 4
	var valType uint32

	r1, _, _ = procRegQueryValueEx.Call(uintptr(hKey), uintptr(unsafe.Pointer(valuePtr)), 0, uintptr(unsafe.Pointer(&valType)), uintptr(unsafe.Pointer(&value)), uintptr(unsafe.Pointer(&size)))
	if r1 != 0 {
		return 0
	}
	return value
}

func setDoublePinyinRegistry(value uint32) {
	pathPtr, _ := syscall.UTF16PtrFromString(registryPath)
	valuePtr, _ := syscall.UTF16PtrFromString(registryValue)

	procRegSetKeyValue.Call(
		HKEY_CURRENT_USER,
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(valuePtr)),
		REG_DWORD,
		uintptr(unsafe.Pointer(&value)),
		4,
	)
}
