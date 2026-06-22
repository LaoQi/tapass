package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/LaoQi/tapass/tools/version"
	"github.com/LaoQi/tapass/tui/internal/tui"
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

	app := tui.NewApp(dbPath)

	p := tea.NewProgram(app)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
