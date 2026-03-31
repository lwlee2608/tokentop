package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Auth struct {
	AccessToken      string `json:"accessToken"`
	RefreshToken     string `json:"refreshToken"`
	ExpiresAt        int64  `json:"expiresAt"`
	SubscriptionType string `json:"subscriptionType"`
	RateLimitTier    string `json:"rateLimitTier"`
}

type credentialsFile struct {
	ClaudeAiOauth *Auth `json:"claudeAiOauth"`
}

func LoadAuth() (*Auth, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(home, ".claude", ".credentials.json"))
	if err != nil {
		return nil, fmt.Errorf("reading ~/.claude/.credentials.json: %w", err)
	}
	var creds credentialsFile
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}
	if creds.ClaudeAiOauth == nil || creds.ClaudeAiOauth.AccessToken == "" {
		return nil, fmt.Errorf("no OAuth credentials found in ~/.claude/.credentials.json")
	}
	return creds.ClaudeAiOauth, nil
}
