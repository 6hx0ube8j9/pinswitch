package main

import (
	_ "embed" // 🌟 引入官方 embed 库

	"github.com/energye/systray"
	"github.com/pkg/browser"
)

var appIcon []byte

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// 🌟 将 nil 改为 appIcon，让托盘直接从内存读取图标数据，百分之百显示
	systray.SetIcon(appIcon) 
	
	systray.SetTitle("Bing 工具")
	systray.SetTooltip("左键点击进入 Bing，右键弹出菜单")

	// 1. 处理左键点击图标
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

func onExit() {}
