//go:build windows

package main

import (
	"pinswitch/core"
	"pinswitch/ui"
	"pinswitch/winapi"
	"syscall"
)

func main() {
	// 1. 尝试创建全局唯一互斥锁
	ret, err := winapi.CreateMutex("Local\\PinswitchUniqueMutexSecure")
	
	// ERROR_ALREADY_EXISTS 错误码是 183
	if err == syscall.Errno(183) {
		// 2. 发现老实例，寻找其窗口
		oldHwnd := winapi.FindWindow("PinswitchHotkeyWindow_Unique_Class")
		if oldHwnd != 0 {
			var pid uint32
			winapi.GetWindowThreadProcessId(oldHwnd, &pid)
			
			// 3. 使用非阻塞的 PostMessage 发送 WM_CLOSE (0x0010)，避免两个进程相互卡死
			winapi.PostMessage(oldHwnd, 0x0010, 0, 0)
			
			// 4. 【原生同步核心】打开老进程句柄，进行内核级同步死等
			if pid != 0 {
				// SYNCHRONIZE (0x00100000) 权限允许我们等待该进程结束
				hProcess, errOpen := syscall.OpenProcess(0x00100000, false, pid)
				if errOpen == nil && hProcess != 0 {
					// 最多挂起等待 2000 毫秒。一旦老进程退出，这里瞬间秒活激活！
					syscall.WaitForSingleObject(hProcess, 2000)
					syscall.CloseHandle(hProcess)
				}
			}
		}
		
		// 5. 老实例已经让位，新实例二次尝试接管 Mutex
		ret, err = winapi.CreateMutex("Local\\PinswitchUniqueMutexSecure")
		if err == syscall.Errno(183) || ret == 0 {
			// 如果 2 秒内老实例由于顽固原因没死透，新实例选择直接闪退，严防多开！
			return
		}
	} else if ret == 0 {
		return
	}

	defer func() {
		if ret != 0 {
			winapi.CloseHandle(syscall.Handle(ret))
		}
	}()

	engine := core.NewSwitchEngine()
	tray := ui.NewTrayUI(engine)
	tray.Start()
}
