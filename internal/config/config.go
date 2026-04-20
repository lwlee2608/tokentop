package config

import (
	"bytes"
	_ "embed"
	"os"
	"path/filepath"
	"strings"

	"github.com/lwlee2608/adder"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Log          LogConfig          `mapstructure:"log"`
	Providers    ProvidersConfig    `mapstructure:"providers"`
	CodexUI      CodexUIConfig      `mapstructure:"codex_ui"`
	ClaudeUI     ClaudeUIConfig     `mapstructure:"claude_ui"`
	OpenRouterUI OpenRouterUIConfig `mapstructure:"openrouter_ui"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
	Path  string `mapstructure:"path"`
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
	Compact    bool `mapstructure:"compact"`
	CodeReview bool `mapstructure:"code_review"`
	PaceTick   bool `mapstructure:"pace_tick"`
}

type ClaudeUIConfig struct {
	Compact  bool `mapstructure:"compact"`
	PaceTick bool `mapstructure:"pace_tick"`
}

type OpenRouterUIConfig struct {
	Summary    bool `mapstructure:"summary"`
	DailySpend bool `mapstructure:"daily_spend"`
	TopModels  bool `mapstructure:"top_models"`
	APIKeys    bool `mapstructure:"api_keys"`
}

func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	a := adder.New()
	a.SetConfigName("config")
	a.SetConfigType("yaml")
	a.AddConfigPath(filepath.Join(home, ".config", "tokentop"))
	a.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	a.AutomaticEnv()

	cfg := &Config{
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
		},
	}

	if err := a.ReadInConfig(); err != nil {
		if strings.HasPrefix(err.Error(), "config file not found") {
			writeDefaultConfig(filepath.Join(home, ".config", "tokentop", "config.yaml"))
			// Retry after writing the default config
			if err := a.ReadInConfig(); err != nil {
				return cfg, nil
			}
		} else {
			return nil, err
		}
	}

	if err := a.Unmarshal(cfg); err != nil {
		return nil, err
	}

	configPath := filepath.Join(home, ".config", "tokentop", "config.yaml")
	if backfillDefaults(configPath) {
		if err := a.ReadInConfig(); err != nil {
			return nil, err
		}
		if err := a.Unmarshal(cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

//go:embed default_config.yaml
var defaultConfigYAML []byte

func writeDefaultConfig(path string) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	_ = os.WriteFile(path, defaultConfigYAML, 0644)
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
