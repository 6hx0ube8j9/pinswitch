package main

import (
	_ "embed" // 必须引入 embed 包
	"github.com/energye/systray"
	"github.com/pkg/browser"
)

// 使用 go:embed 将你的图标文件编译进二进制中
// 路径必须相对于 main.go 所在的目录
//go:embed icons/app.ico
var iconData []byte

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// 1. 设置你上传的图标
	systray.SetIcon(iconData)
	systray.SetTitle("Bing 工具")
	systray.SetTooltip("左键直接进入 Bing，右键弹出菜单")

	// 2. 左键单击图标事件
	systray.SetOnClick(func(menu systray.IMenu) {
		openBing()
	})

	// 3. 右键菜单项
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
	// 退出清理逻辑
}
