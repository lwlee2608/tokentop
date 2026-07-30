package velocirouter

import (
	"fmt"
	"os"
	"strings"
)

const LoginHint = "set VELOCIROUTER_API_KEY env var (see velocirouter.site)"

type Auth struct {
	APIKey string `mask:"first=3,last=4"`
}

func LoadAuth() (*Auth, error) {
	apiKey := strings.TrimSpace(os.Getenv("VELOCIROUTER_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("VELOCIROUTER_API_KEY is not set")
	}

	return &Auth{APIKey: apiKey}, nil
}
