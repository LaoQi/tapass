package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/tapass/tapass-tui/internal/tui"
)

func main() {
	dbPath := ""
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	app := tui.NewApp()
	app.SetInitialDBPath(dbPath)

	p := tea.NewProgram(app)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
