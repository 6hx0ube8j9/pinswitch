package main

import (
	"github.com/energye/systray"
	"github.com/pkg/browser"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// 如果删除图标设置，Windows 极大概率会报错或导致图标不可见
	// systray.SetIcon(nil) // 这样写通常会导致程序崩溃
	
	systray.SetTitle("Bing 工具")
	systray.SetTooltip("测试无图标状态")

	systray.SetOnClick(func(menu systray.IMenu) {
		openBing()
	})

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
