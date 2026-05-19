package main


import (
	"github.com/energye/systray"
	"github.com/pkg/browser"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// 标准做法：传 nil 自动抓取 exe 本身携带的主图标
	systray.SetIcon(nil) 
	
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
