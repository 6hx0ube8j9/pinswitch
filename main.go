package main

import (
	"github.com/energye/systray"
	"github.com/pkg/browser"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// 依然是这个简单的红色图标数据，保证 Windows 能加载图标
	dummyIcon := []byte{
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x10, 0x10, 0x00, 0x00, 0x01, 0x00, 0x20, 0x00, 0x68, 0x04,
		0x00, 0x00, 0x16, 0x00, 0x00, 0x00, 0x28, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x20, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00,
	}

	systray.SetIcon(dummyIcon)
	systray.SetTitle("Bing 工具")
	systray.SetTooltip("左键点击图标或右键菜单均可打开 Bing")

	// 1. 处理左键点击图标
	// energye 库使用 SetOnCLick (注意 C 是大写) 注册回调
	systray.SetOnClick(func() {
		openBing()
	})

	// 2. 创建菜单项并设置点击回调
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

func onExit() {
	// 退出时的清理
}
