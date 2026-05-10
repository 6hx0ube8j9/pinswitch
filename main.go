package main

import (
	"github.com/getlantern/systray"
	"github.com/pkg/browser"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// 设置一个空的图标（或者放你的图标字节）
	systray.SetIcon([]byte{}) 
	systray.SetTitle("Bing工具")
	systray.SetTooltip("左键或菜单均可访问Bing")

	// 1. 添加菜单项
	mOpen := systray.AddMenuItem("打开浏览器", "访问 Bing")
	mQuit := systray.AddMenuItem("退出", "退出程序")

	// 2. 核心逻辑
	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				openBing()
			case <-mQuit.ClickedCh:
				systray.Quit()
			// 注意：systray 在某些版本中通过 ClickedCh 统一处理
			// 如果库版本支持，左键点击图标也会触发默认菜单项
			}
		}
	}()

	// 技巧：有些 Windows 驱动下，第一个菜单项就是左键点击的默认行为
}

func openBing() {
	_ = browser.OpenURL("https://www.bing.com/")
}

func onExit() {}
