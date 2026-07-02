package claude

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

var ErrUnauthorized = errors.New("unauthorized")

const LoginHint = "run `claude /login` to sign in"

type Usage struct {
	SubscriptionType string
	RateLimitTier    string
	Limits           []Limit
}

type Limit struct {
	Kind    string
	Group   string
	Model   string
	Percent float64
	ResetAt time.Time
}

func formatErrorBody(status int, body []byte) string {
	var parsed struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Error.Message != "" {
		return fmt.Sprintf("%d: %s", status, parsed.Error.Message)
	}
	snippet := string(body)
	if len(snippet) > 100 {
		snippet = snippet[:100] + "..."
	}
	return fmt.Sprintf("%d: %s", status, snippet)
}

func FetchUsage(auth *Auth) (*Usage, error) {
	logger := slog.With("provider", "claude")
	started := time.Now()

	req, err := http.NewRequest("GET", "https://api.anthropic.com/api/oauth/usage", nil)
	if err != nil {
		logger.Error("build request failed", "error", err)
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+auth.AccessToken)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("anthropic-dangerous-direct-browser-access", "true")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Warn("request failed", "error", err, "duration_ms", time.Since(started).Milliseconds())
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Warn("read body failed", "error", err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		logger.Warn("request returned non-ok status", "status", resp.StatusCode, "duration_ms", time.Since(started).Milliseconds(), "body", string(body))
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("Anthropic API %s: %w", formatErrorBody(resp.StatusCode, body), ErrUnauthorized)
		}
		return nil, fmt.Errorf("Anthropic API %s", formatErrorBody(resp.StatusCode, body))
	}

	limits, err := parseLimits(body)
	if err != nil {
		logger.Warn("parse usage failed", "error", err)
		return nil, err
	}

	logger.Debug("request completed",
		"status", resp.StatusCode,
		"duration_ms", time.Since(started).Milliseconds(),
		"limits", len(limits),
	)

	return &Usage{
		SubscriptionType: auth.SubscriptionType,
		RateLimitTier:    auth.RateLimitTier,
		Limits:           limits,
	}, nil
}

func parseLimits(body []byte) ([]Limit, error) {
	var parsed struct {
		Limits []struct {
			Kind     string    `json:"kind"`
			Group    string    `json:"group"`
			Percent  float64   `json:"percent"`
			ResetsAt time.Time `json:"resets_at"`
			Scope    *struct {
				Model *struct {
					DisplayName string `json:"display_name"`
				} `json:"model"`
			} `json:"scope"`
		} `json:"limits"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parsing usage response: %w", err)
	}

	limits := make([]Limit, 0, len(parsed.Limits))
	for _, l := range parsed.Limits {
		limit := Limit{
			Kind:    l.Kind,
			Group:   l.Group,
			Percent: l.Percent,
			ResetAt: l.ResetsAt,
		}
		if l.Scope != nil && l.Scope.Model != nil {
			limit.Model = l.Scope.Model.DisplayName
		}
		limits = append(limits, limit)
	}
	return limits, nil
}
