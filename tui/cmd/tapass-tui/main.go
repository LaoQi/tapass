package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tapass/tapass-tui/internal/store/local"
	"github.com/tapass/tapass-tui/internal/tui"
)

func main() {
	dbPath := ""
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	store := local.New()
	app := tui.NewApp(store)
	app.SetStore(store)
	app.SetInitialDBPath(dbPath)

	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}