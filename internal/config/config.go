package config

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/lwlee2608/adder"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Log          LogConfig          `mapstructure:"log"`
	Refresh      RefreshConfig      `mapstructure:"refresh"`
	Providers    ProvidersConfig    `mapstructure:"providers"`
	CodexUI      CodexUIConfig      `mapstructure:"codex_ui"`
	ClaudeUI     ClaudeUIConfig     `mapstructure:"claude_ui"`
	OpenRouterUI OpenRouterUIConfig `mapstructure:"openrouter_ui"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
	Path  string `mapstructure:"path"`
}

type RefreshConfig struct {
	IntervalSeconds int `mapstructure:"interval_seconds"`
}

type ProvidersConfig struct {
	Codex      ProviderConfig `mapstructure:"codex"`
	OpenRouter ProviderConfig `mapstructure:"openrouter"`
	Anthropic  ProviderConfig `mapstructure:"anthropic"`
}

type ProviderConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type CodexUIConfig struct {
	CodeReview bool `mapstructure:"code_review"`
	PaceTick   bool `mapstructure:"pace_tick"`
}

type ClaudeUIConfig struct {
	PaceTick bool `mapstructure:"pace_tick"`
}

type OpenRouterUIConfig struct {
	Summary    bool   `mapstructure:"summary"`
	DailySpend bool   `mapstructure:"daily_spend"`
	TopModels  bool   `mapstructure:"top_models"`
	APIKeys    bool   `mapstructure:"api_keys"`
	Metric     string `mapstructure:"metric"` // spend | requests | tokens
}

func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(home, ".config", "tokentop")
	configPath := filepath.Join(configDir, "config.yaml")

	a := adder.New()
	a.SetConfigName("config")
	a.SetConfigType("yaml")
	a.AddConfigPath(configDir)
	a.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	a.AutomaticEnv()

	cfg := &Config{
		Refresh: RefreshConfig{
			IntervalSeconds: 300,
		},
		CodexUI: CodexUIConfig{
			PaceTick: true,
		},
		ClaudeUI: ClaudeUIConfig{
			PaceTick: true,
		},
		OpenRouterUI: OpenRouterUIConfig{
			Summary:    true,
			DailySpend: true,
			TopModels:  true,
			APIKeys:    true,
			Metric:     "spend",
		},
	}

	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		if werr := writeDefaultConfig(configPath); werr != nil {
			slog.Warn("could not write default config, continuing with built-in defaults", "path", configPath, "error", werr)
			return cfg, nil
		}
	}

	if err := a.ReadInConfig(); err != nil {
		return nil, err
	}
	if err := a.Unmarshal(cfg); err != nil {
		return nil, err
	}

	if backfillDefaults(configPath) {
		if err := a.ReadInConfig(); err != nil {
			return nil, err
		}
		if err := a.Unmarshal(cfg); err != nil {
			return nil, err
		}
	}
	if cfg.Refresh.IntervalSeconds <= 0 {
		return nil, fmt.Errorf("refresh.interval_seconds must be greater than 0")
	}

	return cfg, nil
}

//go:embed default_config.yaml
var defaultConfigYAML []byte

func writeDefaultConfig(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, defaultConfigYAML, 0644)
}

// backfillDefaults reads the existing config file and the embedded default,
// merges any missing keys from the default into the existing config, and
// writes back if anything changed.
func backfillDefaults(path string) bool {
	existing, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	var existingMap map[string]any
	if err := yaml.Unmarshal(existing, &existingMap); err != nil {
		return false
	}
	if existingMap == nil {
		existingMap = map[string]any{}
	}

	var defaultMap map[string]any
	if err := yaml.Unmarshal(defaultConfigYAML, &defaultMap); err != nil {
		return false
	}
	if defaultMap == nil {
		return false
	}

	if !mergeDefaults(existingMap, defaultMap) {
		return false
	}

	out, err := yaml.Marshal(existingMap)
	if err != nil {
		return false
	}

	// Only write if content actually changed
	if !bytes.Equal(bytes.TrimSpace(existing), bytes.TrimSpace(out)) {
		if err := os.WriteFile(path, out, 0644); err != nil {
			return false
		}
		return true
	}

	return false
}

// mergeDefaults recursively fills missing keys in dst from src.
// Returns true if any key was added.
func mergeDefaults(dst, src map[string]any) bool {
	changed := false
	for k, srcVal := range src {
		dstVal, exists := dst[k]
		if !exists {
			dst[k] = srcVal
			changed = true
			continue
		}
		// If both are maps, recurse
		dstMap, dstOk := dstVal.(map[string]any)
		srcMap, srcOk := srcVal.(map[string]any)
		if dstOk && srcOk {
			if mergeDefaults(dstMap, srcMap) {
				changed = true
			}
		}
	}
	return changed
}
