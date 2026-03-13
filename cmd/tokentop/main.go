package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lwlee2608/adder"
	"github.com/lwlee2608/tokentop/internal/config"
	"github.com/lwlee2608/tokentop/internal/tui"
	"github.com/lwlee2608/tokentop/pkg/codex"
	"github.com/lwlee2608/tokentop/pkg/openrouter"
)

var AppVersion = "dev"

func main() {
	onlyCodex := flag.Bool("codex", false, "start with only Codex provider")
	onlyOpenRouter := flag.Bool("openrouter", false, "start with only OpenRouter provider")
	allProviders := flag.Bool("all", false, "start with all providers enabled")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	closeLogger, err := initLogger(cfg.Log.Level, cfg.Log.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: init logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = closeLogger() }()

	slog.Info("starting tokentop", "version", AppVersion)

	switch {
	case *onlyCodex:
		cfg.Providers.Codex.Enabled = true
		cfg.Providers.OpenRouter.Enabled = false
	case *onlyOpenRouter:
		cfg.Providers.Codex.Enabled = false
		cfg.Providers.OpenRouter.Enabled = true
	case *allProviders:
		cfg.Providers.Codex.Enabled = true
		cfg.Providers.OpenRouter.Enabled = true
	}

	if parseLogLevel(cfg.Log.Level) == slog.LevelDebug {
		configJSON, err := adder.PrettyJSON(cfg)
		if err == nil {
			slog.Debug("config loaded", "config", configJSON)
		}
	}

	var codexAuth *codex.Auth
	if cfg.Providers.Codex.Enabled {
		auth, err := codex.LoadAuth()
		if err != nil {
			slog.Warn("codex auth unavailable", "error", err)
			fmt.Fprintf(os.Stderr, "Warning: codex: %v\n", err)
		} else {
			codexAuth = auth
			slog.Info("codex provider enabled")
		}
	}

	var orAuth *openrouter.Auth
	if cfg.Providers.OpenRouter.Enabled {
		auth, err := openrouter.LoadAuth()
		if err != nil {
			slog.Warn("openrouter auth unavailable", "error", err)
			fmt.Fprintf(os.Stderr, "Warning: openrouter: %v\n", err)
		} else {
			orAuth = auth
			slog.Info("openrouter provider enabled")
		}
	}

	// TODO: anthropic provider (not yet implemented)

	if codexAuth == nil && orAuth == nil {
		slog.Error("no providers available")
		fmt.Fprintf(os.Stderr, "Error: no providers available\n")
		os.Exit(1)
	}

	p := tea.NewProgram(tui.New(codexAuth, orAuth, AppVersion), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		slog.Error("tui exited with error", "error", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	slog.Info("tokentop exited")
}
