package claude

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var ErrUnauthorized = errors.New("unauthorized")

type Usage struct {
	SubscriptionType string
	RateLimitTier    string
	SessionLimit     *RateWindow
	WeeklyLimit      *RateWindow
}

type RateWindow struct {
	Utilization float64
	Status      string
	ResetAt     time.Time
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

	// Make a minimal API call to get rate limit headers
	body := `{"model":"claude-haiku-4-5-20251001","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`
	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", strings.NewReader(body))
	if err != nil {
		logger.Error("build request failed", "error", err)
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+auth.AccessToken)
	req.Header.Set("Content-Type", "application/json")
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Warn("request returned non-ok status", "status", resp.StatusCode, "duration_ms", time.Since(started).Milliseconds(), "body", string(body))
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("Anthropic API %s: %w", formatErrorBody(resp.StatusCode, body), ErrUnauthorized)
		}
		return nil, fmt.Errorf("Anthropic API %s", formatErrorBody(resp.StatusCode, body))
	}
	io.Copy(io.Discard, resp.Body)

	usage := &Usage{
		SubscriptionType: auth.SubscriptionType,
		RateLimitTier:    auth.RateLimitTier,
	}

	if v := resp.Header.Get("anthropic-ratelimit-unified-5h-utilization"); v != "" {
		usage.SessionLimit = parseRateWindow(
			v,
			resp.Header.Get("anthropic-ratelimit-unified-5h-status"),
			resp.Header.Get("anthropic-ratelimit-unified-5h-reset"),
		)
	}

	if v := resp.Header.Get("anthropic-ratelimit-unified-7d-utilization"); v != "" {
		usage.WeeklyLimit = parseRateWindow(
			v,
			resp.Header.Get("anthropic-ratelimit-unified-7d-status"),
			resp.Header.Get("anthropic-ratelimit-unified-7d-reset"),
		)
	}

	logger.Debug("request completed",
		"status", resp.StatusCode,
		"duration_ms", time.Since(started).Milliseconds(),
		"session_util", resp.Header.Get("anthropic-ratelimit-unified-5h-utilization"),
		"weekly_util", resp.Header.Get("anthropic-ratelimit-unified-7d-utilization"),
	)

	return usage, nil
}

func parseRateWindow(utilization, status, resetEpoch string) *RateWindow {
	v, err := strconv.ParseFloat(utilization, 64)
	if err != nil {
		return nil
	}
	w := &RateWindow{Status: status, Utilization: v}
	if v, err := strconv.ParseInt(resetEpoch, 10, 64); err == nil && v > 0 {
		w.ResetAt = time.Unix(v, 0)
	}
	return w
}
