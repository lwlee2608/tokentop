package openrouter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

var baseURL = "https://openrouter.ai/api/v1"

type Usage struct {
	Key      KeyUsage
	Credits  *Credits
	Activity *Activity
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

type activityResponse struct {
	Data []activityItem `json:"data"`
}

type activityItem struct {
	Model              string  `json:"model"`
	Usage              float64 `json:"usage"`
	BYOKUsageInference float64 `json:"byok_usage_inference"`
	Requests           float64 `json:"requests"`
	PromptTokens       float64 `json:"prompt_tokens"`
	CompletionTokens   float64 `json:"completion_tokens"`
	ReasoningTokens    float64 `json:"reasoning_tokens"`
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

	usage.Credits = &Credits{
		Total:     credits.Data.TotalCredits,
		Used:      credits.Data.TotalUsage,
		Remaining: credits.Data.TotalCredits - credits.Data.TotalUsage,
	}
	usage.Activity = buildActivity(activity.Data)

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

func doJSON(client *http.Client, auth *Auth, method, url string, target any) error {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+auth.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OpenRouter API %s returned status %d: %s", url, resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	return nil
}
