package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lwlee2608/tokenburn/internal/tui"
	"github.com/lwlee2608/tokenburn/pkg/codex"
)

var AppVersion = "dev"

func main() {
	auth, err := codex.LoadAuth()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(tui.New(auth, AppVersion), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
