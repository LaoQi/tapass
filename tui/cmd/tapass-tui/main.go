package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/tapass/tapass-tools/version"
	"github.com/tapass/tapass-tui/internal/tui"
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("tapass-tui %s\n", version.String())
		return
	}

	dbPath := ""
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	app := tui.NewApp()
	app = app.SetInitialDBPath(dbPath)

	p := tea.NewProgram(app)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
