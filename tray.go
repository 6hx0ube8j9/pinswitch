//go:build windows

package main

import (
	_ "embed"
	"syscall"
	"unsafe"
)

//go:embed icons/quan.ico
var iconQuan []byte

//go:embed icons/shuang.ico
var iconShuang []byte

const (
	MenuIDFullPinyin   = 1001
	MenuIDDoublePinyin = 1002
	MenuIDAutoStart    = 1003
	MenuIDHelp         = 1004
	MenuIDExit         = 1005
)

type TrayUI struct {
	brain       *SwitchBrain
	hIconQuan   uintptr
	hIconShuang uintptr
	isVisible   bool
}

func NewTrayUI(brain *SwitchBrain) *TrayUI {
	return &TrayUI{
		brain:       brain,
		hIconQuan:   HICONFromICOBytes(iconQuan),
		hIconShuang: HICONFromICOBytes(iconShuang),
		isVisible:   false,
	}
}

func (t *TrayUI) Close() {
	if t.isVisible {
		nid := t.getNotifyData()
		ShellNotifyIcon(NIM_DELETE, &nid)
		t.isVisible = false
	}
	if t.hIconQuan != 0 {
		DestroyIcon(t.hIconQuan)
		t.hIconQuan = 0
	}
	if t.hIconShuang != 0 {
		DestroyIcon(t.hIconShuang)
		t.hIconShuang = 0
	}
}

func (t *TrayUI) Show() {
	t.isVisible = true
	t.SyncUI(NIM_ADD)
}

func (t *TrayUI) Hide() {
	if !t.isVisible {
		return
	}
	t.isVisible = false
	nid := t.getNotifyData()
	ShellNotifyIcon(NIM_DELETE, &nid)
}

func (t *TrayUI) SyncUI(action uint32) {
	if !t.isVisible {
		return
	}
	nid := t.getNotifyData()
	if !ShellNotifyIcon(action, &nid) && action == NIM_MODIFY {
		ShellNotifyIcon(NIM_ADD, &nid)
	}
}

func (t *TrayUI) getNotifyData() NotifyIconData {
	nid := NotifyIconData{
		CbSize:           uint32(unsafe.Sizeof(NotifyIconData{})),
		HWnd:             t.brain.hwnd,
		UID:              1,
		UFlags:           NIF_MESSAGE | NIF_ICON | NIF_TIP,
		UCallbackMessage: WM_TRAYICON,
	}

	mode := t.brain.GetIMEMode()
	tip := "Pinswitch: 全拼模式"
	nid.HIcon = t.hIconQuan

	if mode == 1 {
		tip = "Pinswitch: 双拼模式"
		nid.HIcon = t.hIconShuang
	}

	tip16, _ := syscall.UTF16FromString(tip)
	if len(tip16) > 128 {
		tip16 = tip16[:128]
		tip16[127] = 0
	}
	copy(nid.SzTip[:], tip16)

	return nid
}

func (t *TrayUI) ShowMenu() {
	SetForegroundWindow(t.brain.hwnd)
	hMenu := CreatePopupMenu()
	defer DestroyMenu(hMenu)

	mode := t.brain.GetIMEMode()
	quanFlag := uint32(MF_STRING)
	shuangFlag := uint32(MF_STRING)
	if mode == 0 {
		quanFlag |= MF_CHECKED
	} else {
		shuangFlag |= MF_CHECKED
	}

	autoFlag := uint32(MF_STRING)
	if t.brain.IsAutoStart() {
		autoFlag |= MF_CHECKED
	}

	AppendMenu(hMenu, quanFlag, MenuIDFullPinyin, "全拼输入")
	AppendMenu(hMenu, shuangFlag, MenuIDDoublePinyin, "双拼输入")
	AppendMenu(hMenu, MF_SEPARATOR, 0, "")
	AppendMenu(hMenu, autoFlag, MenuIDAutoStart, "开机启动")
	AppendMenu(hMenu, MF_SEPARATOR, 0, "")
	AppendMenu(hMenu, MF_STRING, MenuIDHelp, "快捷键说明")
	AppendMenu(hMenu, MF_STRING, MenuIDExit, "退出程序")

	pt := GetCursorPos()
	TrackPopupMenu(hMenu, TPM_BOTTOMALIGN|TPM_LEFTALIGN|TPM_RIGHTBUTTON, pt.X, pt.Y, 0, t.brain.hwnd, 0)
	PostMessage(t.brain.hwnd, WM_NULL, 0, 0)
}

func (t *TrayUI) HandleMenuClick(cmdID int) {
	switch cmdID {
	case MenuIDFullPinyin:
		t.brain.SetIMEMode(0)
		t.SyncUI(NIM_MODIFY)
	case MenuIDDoublePinyin:
		t.brain.SetIMEMode(1)
		t.SyncUI(NIM_MODIFY)
	case MenuIDAutoStart:
		t.brain.ToggleAutoStart()
	case MenuIDHelp:
		helpText := "【快捷键说明】\n\n" +
			"Shift+Ctrl+Y：切换全拼/双拼\n" +
			"Shift+Ctrl+Win+Y：显示/隐藏托盘图标\n" +
			"Shift+双击程序：显示/隐藏托盘图标"
		MessageBox(t.brain.hwnd, helpText, "Pinswitch", 0x00000040)
	case MenuIDExit:
		PostMessage(t.brain.hwnd, WM_CLOSE, 0, 0)
	}
}
