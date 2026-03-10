package openrouter

import (
	"fmt"
	"os"
	"strings"
)

type Auth struct {
	APIKey string
}

func LoadAuth() (*Auth, error) {
	apiKey := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY is not set")
	}

	return &Auth{APIKey: apiKey}, nil
}
