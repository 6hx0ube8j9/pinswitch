package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/getlantern/systray"
	"github.com/pkg/browser"
)

const targetURL = "https://www.bing.com/"

func main() {
	// systray.Run 接收两个函数：启动时的初始化和退出时的清理
	systray.Run(onReady, onExit)
}

func onReady() {
	// 1. 设置图标 (这里需要字节流，你可以放一个本地的 .ico 或 .png 文件的 byte)
	// 为了演示，我们从网上抓取一个简单的图标字节，或者你可以用空的 []byte{}
	systray.SetIcon(getIcon("https://www.bing.com/favicon.ico"))
	systray.SetTitle("Bing 工具")
	systray.SetTooltip("点击访问 Bing")

	// 2. 创建菜单项
	mOpen := systray.AddMenuItem("打开浏览器", "访问 Bing 主页")
	systray.AddSeparator() // 分割线
	mQuit := systray.AddMenuItem("退出", "关闭程序")

	// 3. 监听事件
	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				// 处理菜单点击
				openBing()
			case <-mQuit.ClickedCh:
				// 处理退出
				systray.Quit()
			}
		}
	}()

	// 注意：systray 本身对“左键单击图标”的直接监听支持有限（随平台变化）
	// 大多数做法是默认菜单的第一项即为左键双击/单击的效果
	// 如果你是在 Windows 上，单击通常会弹出菜单，这是系统的标准交互。
}

func onExit() {
	// 清理逻辑
	fmt.Println("程序已退出")
}

func openBing() {
	err := browser.OpenURL(targetURL)
	if err != nil {
		fmt.Printf("打开浏览器失败: %v\n", err)
	}
}

// 辅助函数：获取图标字节流
func getIcon(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	return data
}
