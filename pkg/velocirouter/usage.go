package velocirouter

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

var ErrUnauthorized = errors.New("unauthorized")

var baseURL = "https://api.velocirouter.site/api/v1"

type Usage struct {
	Key      Key
	Lifetime Lifetime
	Windows  []Window
}

type Key struct {
	ID        string
	Name      string
	Prefix    string
	CreatedAt time.Time
}

type Lifetime struct {
	Requests         float64
	PromptTokens     float64
	CompletionTokens float64
	TotalTokens      float64
	CachedTokens     float64
	CacheWriteTokens float64
	CostUSD          float64
	LastUsedAt       *time.Time
}

type Window struct {
	Name     string
	LimitUSD *float64
	SpentUSD float64
	ResetAt  time.Time
	Duration time.Duration
}

// UsedPercent is meaningful only when the window has a limit.
func (w Window) UsedPercent() float64 {
	if w.LimitUSD == nil || *w.LimitUSD <= 0 {
		return 0
	}
	return w.SpentUSD / *w.LimitUSD * 100
}

func (w Window) HasLimit() bool {
	return w.LimitUSD != nil && *w.LimitUSD > 0
}

type usageResponse struct {
	Key struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		KeyPrefix string    `json:"key_prefix"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"key"`
	Lifetime struct {
		Requests         float64    `json:"requests"`
		PromptTokens     float64    `json:"prompt_tokens"`
		CompletionTokens float64    `json:"completion_tokens"`
		TotalTokens      float64    `json:"total_tokens"`
		CachedTokens     float64    `json:"cached_tokens"`
		CacheWriteTokens float64    `json:"cache_write_tokens"`
		CostUSD          float64    `json:"cost_usd"`
		LastUsedAt       *time.Time `json:"last_used_at"`
	} `json:"lifetime"`
	Windows map[string]struct {
		LimitUSD *float64  `json:"limit_usd"`
		SpentUSD float64   `json:"spent_usd"`
		ResetAt  time.Time `json:"reset_at"`
	} `json:"windows"`
}

func FetchUsage(auth *Auth) (*Usage, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	var resp usageResponse
	if err := doJSON(client, auth, http.MethodGet, baseURL+"/usage/key", &resp); err != nil {
		return nil, err
	}

	return mapUsage(&resp), nil
}

// ParseUsageJSON decodes a raw `/usage/key` response. Intended for tests and
// tooling that want to replay recorded responses.
func ParseUsageJSON(data []byte) (*Usage, error) {
	var resp usageResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse usage: %w", err)
	}
	return mapUsage(&resp), nil
}

func mapUsage(resp *usageResponse) *Usage {
	usage := &Usage{
		Key: Key{
			ID:        resp.Key.ID,
			Name:      resp.Key.Name,
			Prefix:    resp.Key.KeyPrefix,
			CreatedAt: resp.Key.CreatedAt,
		},
		Lifetime: Lifetime{
			Requests:         resp.Lifetime.Requests,
			PromptTokens:     resp.Lifetime.PromptTokens,
			CompletionTokens: resp.Lifetime.CompletionTokens,
			TotalTokens:      resp.Lifetime.TotalTokens,
			CachedTokens:     resp.Lifetime.CachedTokens,
			CacheWriteTokens: resp.Lifetime.CacheWriteTokens,
			CostUSD:          resp.Lifetime.CostUSD,
			LastUsedAt:       resp.Lifetime.LastUsedAt,
		},
		Windows: make([]Window, 0, len(resp.Windows)),
	}

	for name, w := range resp.Windows {
		usage.Windows = append(usage.Windows, Window{
			Name:     name,
			LimitUSD: w.LimitUSD,
			SpentUSD: w.SpentUSD,
			ResetAt:  w.ResetAt,
			Duration: WindowDuration(name),
		})
	}

	sort.Slice(usage.Windows, func(i, j int) bool {
		a, b := usage.Windows[i], usage.Windows[j]
		if (a.Duration == 0) != (b.Duration == 0) {
			return b.Duration == 0
		}
		if a.Duration != b.Duration {
			return a.Duration < b.Duration
		}
		return a.Name < b.Name
	})

	return usage
}

// WindowDuration resolves a window key ("5h", "weekly") to its length.
// Returns 0 when the key is not recognised.
func WindowDuration(name string) time.Duration {
	name = strings.ToLower(strings.TrimSpace(name))
	switch name {
	case "daily":
		return 24 * time.Hour
	case "weekly":
		return 7 * 24 * time.Hour
	case "monthly":
		return 30 * 24 * time.Hour
	}
	if rest, ok := strings.CutSuffix(name, "d"); ok {
		if days, err := strconv.Atoi(rest); err == nil && days > 0 {
			return time.Duration(days) * 24 * time.Hour
		}
	}
	if d, err := time.ParseDuration(name); err == nil && d > 0 {
		return d
	}
	return 0
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

func doJSON(client *http.Client, auth *Auth, method, url string, target any) error {
	logger := slog.With("provider", "velocirouter", "method", method, "url", url)
	started := time.Now()

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		logger.Error("build request failed", "error", err)
		return err
	}
	req.Header.Set("Authorization", "Bearer "+auth.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		logger.Warn("request failed", "error", err, "duration_ms", time.Since(started).Milliseconds())
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Warn("read response failed", "error", err, "status", resp.StatusCode, "duration_ms", time.Since(started).Milliseconds())
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Warn("request returned non-ok status", "status", resp.StatusCode, "duration_ms", time.Since(started).Milliseconds(), "body", string(body))
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("VelociRouter API %s %s: %w", url, formatErrorBody(resp.StatusCode, body), ErrUnauthorized)
		}
		return fmt.Errorf("VelociRouter API %s %s", url, formatErrorBody(resp.StatusCode, body))
	}

	if err := json.Unmarshal(body, target); err != nil {
		logger.Warn("parse response failed", "error", err, "status", resp.StatusCode, "duration_ms", time.Since(started).Milliseconds())
		return fmt.Errorf("parse response: %w", err)
	}

	logger.Debug("request completed", "status", resp.StatusCode, "duration_ms", time.Since(started).Milliseconds())

	return nil
}
