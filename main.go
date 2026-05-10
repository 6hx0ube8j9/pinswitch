package main

import (
	"github.com/energye/systray"
	"github.com/pkg/browser"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// 创建一个简单的 16x16 红色像素图标字节流，防止 Windows 报错
	// 这样你就不用手动上传 icon.ico 了
	dummyIcon := []byte{
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x10, 0x10, 0x00, 0x00, 0x01, 0x00, 0x20, 0x00, 0x68, 0x04,
		0x00, 0x00, 0x16, 0x00, 0x00, 0x00, 0x28, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x20, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00,
	}

	systray.SetIcon(dummyIcon)
	systray.SetTitle("Bing工具")
	systray.SetTooltip("左键直接打开，右键弹出菜单")

	mOpen := systray.AddMenuItem("打开浏览器", "访问 Bing")
	mQuit := systray.AddMenuItem("退出", "退出程序")

	go func() {
		for {
			select {
			case <-systray.OnClickCh(): // 监听左键点击
				openBing()
			case <-mOpen.ClickedCh: // 监听菜单点击
				openBing()
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()
}

func openBing() {
	_ = browser.OpenURL("https://www.bing.com/")
}

func onExit() {}
