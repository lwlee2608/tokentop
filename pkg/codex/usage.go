package codex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Usage struct {
	PlanType            string         `json:"plan_type"`
	RateLimit           RateLimit      `json:"rate_limit"`
	CodeReviewRateLimit RateLimit      `json:"code_review_rate_limit"`
	Credits             UsageCredits   `json:"credits"`
}

type RateLimit struct {
	Allowed         bool          `json:"allowed"`
	LimitReached    bool          `json:"limit_reached"`
	PrimaryWindow   *UsageWindow  `json:"primary_window"`
	SecondaryWindow *UsageWindow  `json:"secondary_window"`
}

type UsageWindow struct {
	UsedPercent        float64 `json:"used_percent"`
	LimitWindowSeconds int     `json:"limit_window_seconds"`
	ResetAfterSeconds  int     `json:"reset_after_seconds"`
	ResetAt            int64   `json:"reset_at"`
}

func (w *UsageWindow) ResetTime() time.Time {
	if w == nil || w.ResetAt == 0 {
		return time.Time{}
	}
	return time.Unix(w.ResetAt, 0)
}

func (w *UsageWindow) WindowMinutes() int {
	if w == nil {
		return 0
	}
	return w.LimitWindowSeconds / 60
}

type UsageCredits struct {
	HasCredits bool    `json:"has_credits"`
	Unlimited  bool    `json:"unlimited"`
	Balance    *string `json:"balance"`
}

func FetchUsage(auth *Auth) (*Usage, error) {
	req, err := http.NewRequest("GET", "https://chatgpt.com/backend-api/wham/usage", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+auth.Tokens.AccessToken)
	req.Header.Set("ChatGPT-Account-ID", auth.Tokens.AccountID)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var usage Usage
	if err := json.Unmarshal(body, &usage); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &usage, nil
}
