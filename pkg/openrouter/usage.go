package openrouter

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"time"
)

var baseURL = "https://openrouter.ai/api/v1"

type Usage struct {
	Key           KeyUsage
	Credits       *Credits
	Activity      *Activity
	DailyActivity *DailyActivity
	APIKeys       []APIKey
}

type KeyUsage struct {
	Label              string
	Limit              float64
	LimitRemaining     float64
	LimitReset         string
	Usage              float64
	UsageDaily         float64
	UsageWeekly        float64
	UsageMonthly       float64
	BYOKUsage          float64
	BYOKUsageDaily     float64
	BYOKUsageWeekly    float64
	BYOKUsageMonthly   float64
	IsManagementKey    bool
	IncludeBYOKInLimit bool
	IsFreeTier         bool
}

type Credits struct {
	Total     float64
	Used      float64
	Remaining float64
}

type Activity struct {
	Totals ActivityTotals
	Models []ModelUsage
}

type APIKey struct {
	Name         string
	Label        string
	Usage        float64
	UsageDaily   float64
	UsageWeekly  float64
	UsageMonthly float64
}

type ActivityTotals struct {
	Spend            float64
	BYOKSpend        float64
	Requests         float64
	PromptTokens     float64
	CompletionTokens float64
	ReasoningTokens  float64
}

type ModelUsage struct {
	Model            string
	Spend            float64
	Requests         float64
	PromptTokens     float64
	CompletionTokens float64
	ReasoningTokens  float64
}

type keyResponse struct {
	Data struct {
		Label              string  `json:"label"`
		Limit              float64 `json:"limit"`
		LimitRemaining     float64 `json:"limit_remaining"`
		LimitReset         string  `json:"limit_reset"`
		Usage              float64 `json:"usage"`
		UsageDaily         float64 `json:"usage_daily"`
		UsageWeekly        float64 `json:"usage_weekly"`
		UsageMonthly       float64 `json:"usage_monthly"`
		BYOKUsage          float64 `json:"byok_usage"`
		BYOKUsageDaily     float64 `json:"byok_usage_daily"`
		BYOKUsageWeekly    float64 `json:"byok_usage_weekly"`
		BYOKUsageMonthly   float64 `json:"byok_usage_monthly"`
		IsManagementKey    bool    `json:"is_management_key"`
		IncludeBYOKInLimit bool    `json:"include_byok_in_limit"`
		IsFreeTier         bool    `json:"is_free_tier"`
	} `json:"data"`
}

type creditsResponse struct {
	Data struct {
		TotalCredits float64 `json:"total_credits"`
		TotalUsage   float64 `json:"total_usage"`
	} `json:"data"`
}

type keysResponse struct {
	Data []struct {
		Name         string  `json:"name"`
		Label        string  `json:"label"`
		Usage        float64 `json:"usage"`
		UsageDaily   float64 `json:"usage_daily"`
		UsageWeekly  float64 `json:"usage_weekly"`
		UsageMonthly float64 `json:"usage_monthly"`
	} `json:"data"`
}

type activityResponse struct {
	Data []activityItem `json:"data"`
}

type activityItem struct {
	Date               string  `json:"date"`
	Model              string  `json:"model"`
	Usage              float64 `json:"usage"`
	BYOKUsageInference float64 `json:"byok_usage_inference"`
	Requests           float64 `json:"requests"`
	PromptTokens       float64 `json:"prompt_tokens"`
	CompletionTokens   float64 `json:"completion_tokens"`
	ReasoningTokens    float64 `json:"reasoning_tokens"`
}

type DailyUsage struct {
	Date   string
	Models []ModelUsage
	Total  float64
}

type DailyActivity struct {
	Days []DailyUsage
}

func FetchUsage(auth *Auth) (*Usage, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	key, err := fetchKey(client, auth)
	if err != nil {
		return nil, err
	}

	usage := &Usage{Key: mapKeyUsage(key)}
	if !usage.Key.IsManagementKey {
		return usage, nil
	}

	credits, err := fetchCredits(client, auth)
	if err != nil {
		return nil, err
	}

	activity, err := fetchActivity(client, auth)
	if err != nil {
		return nil, err
	}

	keys, err := fetchKeys(client, auth)
	if err != nil {
		return nil, err
	}

	usage.Credits = &Credits{
		Total:     credits.Data.TotalCredits,
		Used:      credits.Data.TotalUsage,
		Remaining: credits.Data.TotalCredits - credits.Data.TotalUsage,
	}
	usage.Activity = buildActivity(activity.Data)
	usage.DailyActivity = buildDailyActivity(activity.Data)
	for _, k := range keys.Data {
		usage.APIKeys = append(usage.APIKeys, APIKey{
			Name:         k.Name,
			Label:        k.Label,
			Usage:        k.Usage,
			UsageDaily:   k.UsageDaily,
			UsageWeekly:  k.UsageWeekly,
			UsageMonthly: k.UsageMonthly,
		})
	}
	sort.Slice(usage.APIKeys, func(i, j int) bool {
		return usage.APIKeys[i].UsageMonthly > usage.APIKeys[j].UsageMonthly
	})

	return usage, nil
}

func mapKeyUsage(response *keyResponse) KeyUsage {
	return KeyUsage{
		Label:              response.Data.Label,
		Limit:              response.Data.Limit,
		LimitRemaining:     response.Data.LimitRemaining,
		LimitReset:         response.Data.LimitReset,
		Usage:              response.Data.Usage,
		UsageDaily:         response.Data.UsageDaily,
		UsageWeekly:        response.Data.UsageWeekly,
		UsageMonthly:       response.Data.UsageMonthly,
		BYOKUsage:          response.Data.BYOKUsage,
		BYOKUsageDaily:     response.Data.BYOKUsageDaily,
		BYOKUsageWeekly:    response.Data.BYOKUsageWeekly,
		BYOKUsageMonthly:   response.Data.BYOKUsageMonthly,
		IsManagementKey:    response.Data.IsManagementKey,
		IncludeBYOKInLimit: response.Data.IncludeBYOKInLimit,
		IsFreeTier:         response.Data.IsFreeTier,
	}
}

// ParseActivityJSON decodes a raw `/activity` response and returns the
// aggregated Activity and DailyActivity views. Intended for tests and
// tooling that want to replay recorded responses.
func ParseActivityJSON(data []byte) (*Activity, *DailyActivity, error) {
	var resp activityResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, nil, fmt.Errorf("parse activity: %w", err)
	}
	return buildActivity(resp.Data), buildDailyActivity(resp.Data), nil
}

func buildActivity(items []activityItem) *Activity {
	byModel := make(map[string]*ModelUsage, len(items))
	activity := &Activity{}

	for _, item := range items {
		activity.Totals.Spend += item.Usage
		activity.Totals.BYOKSpend += item.BYOKUsageInference
		activity.Totals.Requests += item.Requests
		activity.Totals.PromptTokens += item.PromptTokens
		activity.Totals.CompletionTokens += item.CompletionTokens
		activity.Totals.ReasoningTokens += item.ReasoningTokens

		model := byModel[item.Model]
		if model == nil {
			model = &ModelUsage{Model: item.Model}
			byModel[item.Model] = model
		}

		model.Spend += item.Usage
		model.Requests += item.Requests
		model.PromptTokens += item.PromptTokens
		model.CompletionTokens += item.CompletionTokens
		model.ReasoningTokens += item.ReasoningTokens
	}

	activity.Models = make([]ModelUsage, 0, len(byModel))
	for _, model := range byModel {
		activity.Models = append(activity.Models, *model)
	}

	sort.Slice(activity.Models, func(i, j int) bool {
		if activity.Models[i].Spend == activity.Models[j].Spend {
			return activity.Models[i].Requests > activity.Models[j].Requests
		}
		return activity.Models[i].Spend > activity.Models[j].Spend
	})

	return activity
}

func buildDailyActivity(items []activityItem) *DailyActivity {
	byDate := make(map[string]map[string]*ModelUsage)

	for _, item := range items {
		models := byDate[item.Date]
		if models == nil {
			models = make(map[string]*ModelUsage)
			byDate[item.Date] = models
		}
		m := models[item.Model]
		if m == nil {
			m = &ModelUsage{Model: item.Model}
			models[item.Model] = m
		}
		m.Spend += item.Usage
		m.Requests += item.Requests
		m.PromptTokens += item.PromptTokens
		m.CompletionTokens += item.CompletionTokens
		m.ReasoningTokens += item.ReasoningTokens
	}

	sortedDates := make([]string, 0, len(byDate))
	for d := range byDate {
		sortedDates = append(sortedDates, d)
	}
	sort.Strings(sortedDates)

	daily := &DailyActivity{Days: make([]DailyUsage, 0, len(sortedDates))}
	for _, date := range sortedDates {
		models := byDate[date]
		day := DailyUsage{Date: date, Models: make([]ModelUsage, 0, len(models))}
		for _, m := range models {
			day.Models = append(day.Models, *m)
			day.Total += m.Spend
		}
		sort.Slice(day.Models, func(i, j int) bool {
			return day.Models[i].Spend > day.Models[j].Spend
		})
		daily.Days = append(daily.Days, day)
	}

	return daily
}

func fetchKeys(client *http.Client, auth *Auth) (*keysResponse, error) {
	var result keysResponse
	if err := doJSON(client, auth, http.MethodGet, baseURL+"/keys", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func fetchKey(client *http.Client, auth *Auth) (*keyResponse, error) {
	var result keyResponse
	if err := doJSON(client, auth, http.MethodGet, baseURL+"/key", &result); err != nil {
		if err := doJSON(client, auth, http.MethodGet, baseURL+"/auth/key", &result); err != nil {
			return nil, err
		}
	}
	return &result, nil
}

func fetchCredits(client *http.Client, auth *Auth) (*creditsResponse, error) {
	var result creditsResponse
	if err := doJSON(client, auth, http.MethodGet, baseURL+"/credits", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func fetchActivity(client *http.Client, auth *Auth) (*activityResponse, error) {
	var result activityResponse
	if err := doJSON(client, auth, http.MethodGet, baseURL+"/activity", &result); err != nil {
		return nil, err
	}
	return &result, nil
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
	logger := slog.With("provider", "openrouter", "method", method, "url", url)
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
		return fmt.Errorf("OpenRouter API %s %s", url, formatErrorBody(resp.StatusCode, body))
	}

	if err := json.Unmarshal(body, target); err != nil {
		logger.Warn("parse response failed", "error", err, "status", resp.StatusCode, "duration_ms", time.Since(started).Milliseconds())
		return fmt.Errorf("parse response: %w", err)
	}

	logger.Debug("request completed", "status", resp.StatusCode, "duration_ms", time.Since(started).Milliseconds())

	return nil
}
