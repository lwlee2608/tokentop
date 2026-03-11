package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lwlee2608/tokentop/internal/config"
	"github.com/lwlee2608/tokentop/internal/tui"
	"github.com/lwlee2608/tokentop/pkg/codex"
	"github.com/lwlee2608/tokentop/pkg/openrouter"
)

var AppVersion = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var codexAuth *codex.Auth
	if cfg.Providers.Codex.Enabled {
		auth, err := codex.LoadAuth()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: codex: %v\n", err)
		} else {
			codexAuth = auth
		}
	}

	var orAuth *openrouter.Auth
	if cfg.Providers.OpenRouter.Enabled {
		auth, err := openrouter.LoadAuth()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: openrouter: %v\n", err)
		} else {
			orAuth = auth
		}
	}

	// TODO: anthropic provider (not yet implemented)

	if codexAuth == nil && orAuth == nil {
		fmt.Fprintf(os.Stderr, "Error: no providers available\n")
		os.Exit(1)
	}

	p := tea.NewProgram(tui.New(codexAuth, orAuth, AppVersion), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
