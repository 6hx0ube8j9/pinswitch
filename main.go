package main

import (
	"github.com/energye/systray"
	"github.com/pkg/browser"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// 红色像素图标数据
	dummyIcon := []byte{
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x10, 0x10, 0x00, 0x00, 0x01, 0x00, 0x20, 0x00, 0x68, 0x04,
		0x00, 0x00, 0x16, 0x00, 0x00, 0x00, 0x28, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x20, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00,
	}

	systray.SetIcon(dummyIcon)
	systray.SetTitle("Bing 工具")
	systray.SetTooltip("左键点击图标或右键菜单均可打开 Bing")

	// 1. 修复：添加 menu 参数以匹配接口要求
	systray.SetOnClick(func(menu systray.IMenu) {
		openBing()
	})

	// 2. 创建右键菜单项
	mOpen := systray.AddMenuItem("打开浏览器", "访问 Bing")
	mOpen.Click(func() {
		openBing()
	})

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("退出", "退出程序")
	mQuit.Click(func() {
		systray.Quit()
	})
}

func openBing() {
	_ = browser.OpenURL("https://www.bing.com/")
}

func onExit() {}
