package codex

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Usage struct {
	PlanType               string
	PrimaryUsedPercent     float64
	PrimaryWindowMinutes   int
	PrimaryResetAt         time.Time
	SecondaryUsedPercent   float64
	SecondaryWindowMinutes int
	SecondaryResetAt       time.Time
	CreditsHasCredits      bool
	CreditsBalance         string
	CreditsUnlimited       bool
}

func FetchUsage(auth *Auth) (*Usage, error) {
	body := `{
		"model": "gpt-5.3-codex",
		"instructions": "",
		"input": [{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]}],
		"tools": [],
		"store": false,
		"stream": true
	}`

	req, err := http.NewRequest("POST", "https://chatgpt.com/backend-api/codex/responses", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+auth.Tokens.AccessToken)
	req.Header.Set("ChatGPT-Account-ID", auth.Tokens.AccountID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("originator", "codex_cli_rs")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	usage := &Usage{
		PlanType: resp.Header.Get("x-codex-plan-type"),
	}

	usage.PrimaryUsedPercent = parseFloat(resp.Header.Get("x-codex-primary-used-percent"))
	usage.PrimaryWindowMinutes = parseInt(resp.Header.Get("x-codex-primary-window-minutes"))
	usage.PrimaryResetAt = parseUnix(resp.Header.Get("x-codex-primary-reset-at"))

	usage.SecondaryUsedPercent = parseFloat(resp.Header.Get("x-codex-secondary-used-percent"))
	usage.SecondaryWindowMinutes = parseInt(resp.Header.Get("x-codex-secondary-window-minutes"))
	usage.SecondaryResetAt = parseUnix(resp.Header.Get("x-codex-secondary-reset-at"))

	usage.CreditsHasCredits = resp.Header.Get("x-codex-credits-has-credits") == "True"
	usage.CreditsBalance = resp.Header.Get("x-codex-credits-balance")
	usage.CreditsUnlimited = resp.Header.Get("x-codex-credits-unlimited") == "True"

	return usage, nil
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func parseInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

func parseUnix(s string) time.Time {
	v, _ := strconv.ParseInt(s, 10, 64)
	if v == 0 {
		return time.Time{}
	}
	return time.Unix(v, 0)
}
