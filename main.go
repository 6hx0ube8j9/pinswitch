package main

import (
	"github.com/energye/systray"
	"github.com/pkg/browser"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// 一个简单的 16x16 红色像素图标数据
	dummyIcon := []byte{
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x10, 0x10, 0x00, 0x00, 0x01, 0x00, 0x20, 0x00, 0x68, 0x04,
		0x00, 0x00, 0x16, 0x00, 0x00, 0x00, 0x28, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x20, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00,
	}

	systray.SetIcon(dummyIcon)
	systray.SetTitle("Bing 工具")
	systray.SetTooltip("左键单击或右键菜单访问 Bing")

	// 创建菜单项
	mOpen := systray.AddMenuItem("打开浏览器", "访问 Bing")
	mQuit := systray.AddMenuItem("退出", "退出程序")

	go func() {
		for {
			select {
			// 1. 监听左键点击图标 (最新 API)
			case <-systray.OnClick(): 
				openBing()

			// 2. 监听菜单项点击 (最新 API)
			case <-mOpen.Click(): 
				openBing()

			// 3. 监听退出菜单点击
			case <-mQuit.Click():
				systray.Quit()
			}
		}
	}()
}

func openBing() {
	_ = browser.OpenURL("https://www.bing.com/")
}

func onExit() {}
