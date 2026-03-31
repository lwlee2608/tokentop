package claude

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

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
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		logger.Warn("request returned non-ok status", "status", resp.StatusCode, "duration_ms", time.Since(started).Milliseconds())
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

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
	w := &RateWindow{Status: status}
	if v, err := strconv.ParseFloat(utilization, 64); err == nil {
		w.Utilization = v
	}
	if v, err := strconv.ParseInt(resetEpoch, 10, 64); err == nil && v > 0 {
		w.ResetAt = time.Unix(v, 0)
	}
	return w
}
