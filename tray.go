//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	ResIconQuan   = 101
	ResIconShuang = 102

	MenuIDFullPinyin   = 1001
	MenuIDDoublePinyin = 1002
	MenuIDAutoStart    = 1003
	MenuIDHelp         = 1004
	MenuIDExit         = 1005
)

var pinswitchTrayGUID = windows.GUID{
	Data1: 0x5b3c4d2e,
	Data2: 0x1a2b,
	Data3: 0x3c4d,
	Data4: [8]byte{0x5e, 0x6f, 0x7a, 0x8b, 0x9c, 0x0d, 0x1e, 0x2f},
}

type TrayUI struct {
	brain       *SwitchBrain
	currentSize int
	hIconQuan   uintptr
	hIconShuang uintptr
	isVisible   bool
	useGUID     bool
}

func NewTrayUI(brain *SwitchBrain) *TrayUI {
	t := &TrayUI{
		brain:   brain,
		useGUID: true,
	}
	t.reloadIcons(GetTaskbarSmallIconSize())
	return t
}

func (t *TrayUI) reloadIcons(targetSize int) {
	if targetSize <= 0 {
		targetSize = 16
	}
	if t.currentSize == targetSize && t.hIconQuan != 0 && t.hIconShuang != 0 {
		return
	}

	newQuan := LoadIconResource(ResIconQuan, targetSize)
	newShuang := LoadIconResource(ResIconShuang, targetSize)

	if newQuan == 0 || newShuang == 0 {
		if newQuan != 0 {
			DestroyIcon(newQuan)
		}
		if newShuang != 0 {
			DestroyIcon(newShuang)
		}
		return
	}

	oldQuan := t.hIconQuan
	oldShuang := t.hIconShuang

	t.hIconQuan = newQuan
	t.hIconShuang = newShuang
	t.currentSize = targetSize

	if oldQuan != 0 {
		DestroyIcon(oldQuan)
	}
	if oldShuang != 0 {
		DestroyIcon(oldShuang)
	}
}

func (t *TrayUI) CheckDpiAndRefresh() {
	if !t.isVisible {
		return
	}
	targetSize := GetTaskbarSmallIconSize()
	if targetSize != t.currentSize {
		t.reloadIcons(targetSize)
		t.shellNotify(NIM_MODIFY)
	}
}

func (t *TrayUI) Close() {
	if t.isVisible {
		t.shellNotify(NIM_DELETE)
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
	t.shellNotify(NIM_ADD)
}

func (t *TrayUI) Hide() {
	if !t.isVisible {
		return
	}
	t.isVisible = false
	t.shellNotify(NIM_DELETE)
}

func (t *TrayUI) SyncUI() {
	if !t.isVisible {
		return
	}
	t.shellNotify(NIM_MODIFY)
}

func (t *TrayUI) shellNotify(action uint32) bool {
	nid := t.getNotifyData()

	if ShellNotifyIcon(action, &nid) {
		return true
	}

	if action == NIM_ADD && t.useGUID {
		t.useGUID = false
		nid = t.getNotifyData()
		if ShellNotifyIcon(NIM_ADD, &nid) {
			return true
		}
	}

	if action == NIM_MODIFY {
		return ShellNotifyIcon(NIM_ADD, &nid)
	}

	return false
}

func (t *TrayUI) getNotifyData() NotifyIconData {
	nid := NotifyIconData{
		CbSize:           uint32(unsafe.Sizeof(NotifyIconData{})),
		HWnd:             t.brain.hwnd,
		UID:              1,
		UFlags:           NIF_MESSAGE | NIF_ICON | NIF_TIP,
		UCallbackMessage: WM_TRAYICON,
	}

	if t.useGUID {
		nid.UFlags |= NIF_GUID
		nid.GuidItem = pinswitchTrayGUID
	}

	mode := t.brain.GetIMEMode()
	tip := "Pinswitch: 全拼模式"
	nid.HIcon = t.hIconQuan

	if mode == IMEModeDouble {
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

	if mode == IMEModeDouble {
		shuangFlag |= MF_CHECKED
	} else {
		quanFlag |= MF_CHECKED
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
		t.brain.SetIMEMode(IMEModeFull)
	case MenuIDDoublePinyin:
		t.brain.SetIMEMode(IMEModeDouble)
	case MenuIDAutoStart:
		t.brain.ToggleAutoStart()
	case MenuIDHelp:
		helpText := "【快捷键说明】\n\n" +
			"Shift+Ctrl+Y：切换全拼/双拼\n" +
			"Shift+Ctrl+Win+Y：显示/隐藏托盘图标\n" +
			"Shift+双击程序：显示/隐藏托盘图标"
		MessageBox(t.brain.hwnd, helpText, "Pinswitch", MB_OK|MB_ICONINFORMATION)
	case MenuIDExit:
		PostMessage(t.brain.hwnd, WM_CLOSE, 0, 0)
	}
}
