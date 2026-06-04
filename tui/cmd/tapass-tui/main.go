package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tapass/tapass-tui/internal/store/local"
	"github.com/tapass/tapass-tui/internal/tui"
)

func main() {
	store := local.New()
	app := tui.NewApp(store)
	app.SetStore(store)

	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}