package tui

import "github.com/LaoQi/tapass/tui/internal/model"

type OpenVaultMsg struct {
	DB   *model.DB
	Path string
}

type CreateVaultMsg struct {
	DB   *model.DB
	Path string
}

type BackToMainMsg struct{}

type OpenDBConfigMsg struct{}

type AttrChangedMsg struct {
	Key string
}

type PasswordChangedMsg struct{}

type SaveVaultMsg struct{}

type SaveAndQuitMsg struct{}

type VaultSavedMsg struct {
	QuitAfter bool
}

type ErrorMsg struct {
	Err error
}

type dirtyMsg struct{ Dirty bool }

type cancelQuitMsg struct{}

type mainViewModelEventMsg struct{ Event model.Event }

type searchEnterMsg struct{}

type searchExitMsg struct{}

type searchFocusMsg struct{}

type searchBlurMsg struct{}

type moveUpMsg struct{}

type moveDownMsg struct{}

type setPrefixMsg struct{ Prefix string }

type refreshMsg struct{}

type dbEventMsg struct{ Event model.Event }

type tickMsg struct{}

type copyClearMsg struct{}

// 右栏展示意图（显式互斥；替代原先字段互相覆盖的 syncRightMsg）
// showAttrListMsg：右栏显示某分组/条目下的属性列表
type showAttrListMsg struct {
	Prefix string
	Attrs  []AttrInfo
}

// showAttrDetailMsg：右栏显示单个属性详情
type showAttrDetailMsg struct {
	EntryPath string
	Attr      string
}

// clearDetailMsg：右栏清空（无选中项）
type clearDetailMsg struct{}

type startNewMsg struct{ Prefix string }

type refreshTOTPMsg struct{}

type resizeMsg struct{ Width, Height int }

type initialPathMsg struct{ Path string }

type setFocusMsg struct{ Focused bool }
