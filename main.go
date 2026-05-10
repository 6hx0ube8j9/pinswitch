package main

import (
	_ "embed"
	"github.com/energye/systray"
	"github.com/pkg/browser"
)

//go:embed icons/app.ico
var iconData []byte

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(iconData)
	systray.SetTitle("Bing 工具")
	systray.SetTooltip("左键点击进入 Bing")

	// 左键点击直接打开
	systray.SetOnClick(func(menu systray.IMenu) {
		browser.OpenURL("https://www.bing.com/")
	})

	// 右键菜单
	mOpen := systray.AddMenuItem("打开浏览器", "访问 Bing")
	mOpen.Click(func() {
		browser.OpenURL("https://www.bing.com/")
	})

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("退出", "关闭程序")
	mQuit.Click(func() {
		systray.Quit()
	})
}

func onExit() {}
