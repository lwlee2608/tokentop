package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	if runtime.GOOS == "darwin" {
		auth, kcErr := loadAuthFromKeychain()
		if kcErr == nil {
			return auth, nil
		}
		auth, fileErr := loadAuthFromFile()
		if fileErr == nil {
			return auth, nil
		}
		return nil, fmt.Errorf("no Claude credentials found: keychain: %v; file: %v", kcErr, fileErr)
	}
	return loadAuthFromFile()
}

func loadAuthFromFile() (*Auth, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(home, ".claude", ".credentials.json"))
	if err != nil {
		return nil, fmt.Errorf("reading ~/.claude/.credentials.json: %w", err)
	}
	return parseCredentials(data, "~/.claude/.credentials.json")
}

func loadAuthFromKeychain() (*Auth, error) {
	out, err := exec.Command("security", "find-generic-password", "-s", "Claude Code-credentials", "-w").Output()
	if err != nil {
		return nil, fmt.Errorf("reading claude keychain: %w", err)
	}
	return parseCredentials(bytes.TrimSpace(out), "claude keychain")
}

func parseCredentials(data []byte, source string) (*Auth, error) {
	var creds credentialsFile
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}
	if creds.ClaudeAiOauth == nil || creds.ClaudeAiOauth.AccessToken == "" {
		return nil, fmt.Errorf("no OAuth credentials found in %s", source)
	}
	return creds.ClaudeAiOauth, nil
}
