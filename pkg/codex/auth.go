package codex

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Auth struct {
	AuthMode string `json:"auth_mode"`
	Tokens   struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		AccountID    string `json:"account_id"`
	} `json:"tokens"`
}

func LoadAuth() (*Auth, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(home, ".codex", "auth.json"))
	if err != nil {
		return nil, fmt.Errorf("reading ~/.codex/auth.json: %w", err)
	}
	var auth Auth
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, fmt.Errorf("parsing auth.json: %w", err)
	}
	return &auth, nil
}
