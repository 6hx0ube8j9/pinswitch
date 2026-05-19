package main

import (
	_ "embed"
	"github.com/energye/systray"
	"github.com/pkg/browser"
)

var iconData []byte

func main() {
	// 启动托盘运行循环
	systray.Run(onReady, onExit)
}

func onReady() {
	// 设置托盘图标
	systray.SetIcon(iconData)
	systray.SetTitle("Bing 工具")
	systray.SetTooltip("左键点击进入 Bing，右键弹出菜单")

	// 1. 处理左键点击图标 (Windows 专用事件)
	systray.SetOnClick(func(menu systray.IMenu) {
		browser.OpenURL("https://www.bing.com/")
	})

	// 2. 处理右键菜单
	mOpen := systray.AddMenuItem("打开浏览器", "访问 Bing")
	mOpen.Click(func() {
		browser.OpenURL("https://www.bing.com/")
	})

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("退出程序", "关闭工具")
	mQuit.Click(func() {
		systray.Quit()
	})
}

func onExit() {
	// 程序退出时的清理逻辑（可选）
}
