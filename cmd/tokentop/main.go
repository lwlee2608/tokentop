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
	"github.com/lwlee2608/tokentop/pkg/claude"
	"github.com/lwlee2608/tokentop/pkg/codex"
	"github.com/lwlee2608/tokentop/pkg/openrouter"
	"github.com/lwlee2608/tokentop/pkg/velocirouter"
)

var AppVersion = "dev"

func main() {
	onlyCodex := flag.Bool("codex", false, "start with only Codex provider")
	onlyClaude := flag.Bool("claude", false, "start with only Claude provider")
	onlyOpenRouter := flag.Bool("openrouter", false, "start with only OpenRouter provider")
	onlyVelociRouter := flag.Bool("velocirouter", false, "start with only VelociRouter provider")
	allProviders := flag.Bool("all", false, "start with all providers enabled")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(AppVersion)
		return
	}

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

	on := config.ProviderConfig{Enabled: true}
	switch {
	case *onlyCodex:
		cfg.Providers = config.ProvidersConfig{Codex: on}
	case *onlyClaude:
		cfg.Providers = config.ProvidersConfig{Anthropic: on}
	case *onlyOpenRouter:
		cfg.Providers = config.ProvidersConfig{OpenRouter: on}
	case *onlyVelociRouter:
		cfg.Providers = config.ProvidersConfig{VelociRouter: on}
	case *allProviders:
		cfg.Providers = config.ProvidersConfig{Codex: on, OpenRouter: on, Anthropic: on, VelociRouter: on}
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
			fmt.Fprintln(os.Stderr, "Hint:", codex.LoginHint)
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
			fmt.Fprintln(os.Stderr, "Hint:", openrouter.LoginHint)
		} else {
			orAuth = auth
			slog.Info("openrouter provider enabled")
			if slog.Default().Enabled(nil, slog.LevelDebug) {
				if masked, err := adder.PrettyJSON(orAuth); err == nil {
					slog.Debug("openrouter auth", "auth", masked)
				}
			}
		}
	}

	var claudeAuth *claude.Auth
	if cfg.Providers.Anthropic.Enabled {
		auth, err := claude.LoadAuth()
		if err != nil {
			slog.Warn("claude auth unavailable", "error", err)
			fmt.Fprintf(os.Stderr, "Warning: claude: %v\n", err)
			fmt.Fprintln(os.Stderr, "Hint:", claude.LoginHint)
		} else {
			claudeAuth = auth
			slog.Info("claude provider enabled")
		}
	}

	var vrAuth *velocirouter.Auth
	if cfg.Providers.VelociRouter.Enabled {
		auth, err := velocirouter.LoadAuth()
		if err != nil {
			slog.Warn("velocirouter auth unavailable", "error", err)
			fmt.Fprintf(os.Stderr, "Warning: velocirouter: %v\n", err)
			fmt.Fprintln(os.Stderr, "Hint:", velocirouter.LoginHint)
		} else {
			vrAuth = auth
			slog.Info("velocirouter provider enabled")
			if slog.Default().Enabled(nil, slog.LevelDebug) {
				if masked, err := adder.PrettyJSON(vrAuth); err == nil {
					slog.Debug("velocirouter auth", "auth", masked)
				}
			}
		}
	}

	if codexAuth == nil && orAuth == nil && claudeAuth == nil && vrAuth == nil &&
		!cfg.Providers.Codex.Enabled && !cfg.Providers.Anthropic.Enabled && !cfg.Providers.OpenRouter.Enabled && !cfg.Providers.VelociRouter.Enabled {
		slog.Error("no providers available")
		fmt.Fprintf(os.Stderr, "Error: no providers available\n")
		os.Exit(1)
	}

	p := tea.NewProgram(tui.New(
		tui.CodexProvider{Auth: codexAuth, Enabled: cfg.Providers.Codex.Enabled, UI: cfg.CodexUI},
		tui.OpenRouterProvider{Auth: orAuth, Enabled: cfg.Providers.OpenRouter.Enabled, UI: cfg.OpenRouterUI},
		tui.ClaudeProvider{Auth: claudeAuth, Enabled: cfg.Providers.Anthropic.Enabled, UI: cfg.ClaudeUI},
		tui.VelociRouterProvider{Auth: vrAuth, Enabled: cfg.Providers.VelociRouter.Enabled, UI: cfg.VelociRouterUI},
		cfg.UI,
		cfg.Refresh,
		AppVersion,
	), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		slog.Error("tui exited with error", "error", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	slog.Info("tokentop exited")
}
