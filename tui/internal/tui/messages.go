package tui

import "github.com/tapass/tapass-tui/internal/model"

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

type syncRightMsg struct {
	EntryPath     string
	Attrs         []AttrInfo
	SelectedAttr  string
	SetDetailMode bool
	ClearOnly     bool
}

type startNewMsg struct{ Prefix string }

type refreshTOTPMsg struct{}

type helpToggleMsg struct{}

type helpCloseMsg struct{}

type setHelpSizeMsg struct{ Width, Height int }

type initialPathMsg struct{ Path string }

type setWelcomeSizeMsg struct{ Width, Height int }
