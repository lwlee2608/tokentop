package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/lwlee2608/adder"
)

type Config struct {
	Providers ProvidersConfig `mapstructure:"providers"`
}

type ProvidersConfig struct {
	Codex      ProviderConfig `mapstructure:"codex"`
	OpenRouter ProviderConfig `mapstructure:"openrouter"`
	Anthropic  ProviderConfig `mapstructure:"anthropic"`
}

type ProviderConfig struct {
	Enabled bool `mapstructure:"enabled"`
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

	cfg := defaultConfig()

	if err := a.ReadInConfig(); err != nil {
		if strings.HasPrefix(err.Error(), "config file not found") {
			return cfg, nil
		}
		return nil, err
	}

	if err := a.Unmarshal(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Providers: ProvidersConfig{
			Codex: ProviderConfig{Enabled: true},
		},
	}
}
