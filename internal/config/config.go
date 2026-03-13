package config

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"

	"github.com/lwlee2608/adder"
)

type Config struct {
	Log          LogConfig        `mapstructure:"log"`
	Providers    ProvidersConfig  `mapstructure:"providers"`
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
