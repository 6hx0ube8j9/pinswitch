package main

import (
	_ "embed"

	"github.com/energye/systray"
	"github.com/pkg/browser"
)

//go:embed icons/app.ico
var appIcon []byte

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// 喂给托盘标准的 16x16 字节流
	systray.SetIcon(appIcon) 
	
	systray.SetTitle("Bing 工具")
	systray.SetTooltip("左键点击进入 Bing，右键弹出菜单")

	systray.SetOnClick(func(menu systray.IMenu) {
		browser.OpenURL("https://www.bing.com/")
	})

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
