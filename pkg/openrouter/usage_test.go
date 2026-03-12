package openrouter

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchUsageLive(t *testing.T) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		t.Skip("OPENROUTER_API_KEY is not set")
	}

	usage, err := FetchUsage(&Auth{APIKey: apiKey})
	require.NoError(t, err)
	assert.NotEmpty(t, usage.Key.Label)

	t.Logf("label=%s management=%t limit=$%.2f remaining=$%.2f usage=$%.2f daily=$%.2f weekly=$%.2f monthly=$%.2f",
		usage.Key.Label,
		usage.Key.IsManagementKey,
		usage.Key.Limit,
		usage.Key.LimitRemaining,
		usage.Key.Usage,
		usage.Key.UsageDaily,
		usage.Key.UsageWeekly,
		usage.Key.UsageMonthly,
	)

	if usage.Credits != nil {
		t.Logf("credits: total=$%.2f used=$%.2f remaining=$%.2f",
			usage.Credits.Total, usage.Credits.Used, usage.Credits.Remaining)
	}

	if usage.Activity != nil {
		logActivity(t, "all keys", usage.Activity)
	}

	if len(usage.APIKeys) > 0 {
		t.Logf("API keys:")
		for _, k := range usage.APIKeys {
			t.Logf("  %-15s %s  total=$%.2f daily=$%.2f weekly=$%.2f monthly=$%.2f",
				k.Name, k.Label, k.Usage, k.UsageDaily, k.UsageWeekly, k.UsageMonthly)
		}
	}
}

func TestActivityResponseUnmarshal(t *testing.T) {
	raw := `{"data":[
		{"date":"2026-03-10","model":"anthropic/claude-opus-4.6","usage":5.12,"byok_usage_inference":0,"requests":10,"prompt_tokens":1000,"completion_tokens":500,"reasoning_tokens":0},
		{"date":"2026-03-11","model":"openai/gpt-5.1","usage":3.45,"byok_usage_inference":0.5,"requests":5,"prompt_tokens":800,"completion_tokens":400,"reasoning_tokens":100}
	]}`

	var resp activityResponse
	require.NoError(t, json.Unmarshal([]byte(raw), &resp))
	require.Len(t, resp.Data, 2)

	assert.Equal(t, "2026-03-10", resp.Data[0].Date)
	assert.Equal(t, "anthropic/claude-opus-4.6", resp.Data[0].Model)
	assert.InDelta(t, 5.12, resp.Data[0].Usage, 0.001)

	assert.Equal(t, "2026-03-11", resp.Data[1].Date)
	assert.Equal(t, "openai/gpt-5.1", resp.Data[1].Model)
	assert.InDelta(t, 0.5, resp.Data[1].BYOKUsageInference, 0.001)
	assert.InDelta(t, 100.0, resp.Data[1].ReasoningTokens, 0.001)
}

func TestBuildDailyActivity(t *testing.T) {
	items := []activityItem{
		{Date: "2026-03-10", Model: "anthropic/claude-opus-4.6", Usage: 5.0, Requests: 10, PromptTokens: 1000, CompletionTokens: 500},
		{Date: "2026-03-10", Model: "openai/gpt-5.1", Usage: 3.0, Requests: 5, PromptTokens: 800, CompletionTokens: 400},
		{Date: "2026-03-11", Model: "anthropic/claude-opus-4.6", Usage: 2.0, Requests: 4, PromptTokens: 600, CompletionTokens: 300},
		{Date: "2026-03-11", Model: "openai/gpt-5.1", Usage: 7.0, Requests: 15, PromptTokens: 2000, CompletionTokens: 1000},
		{Date: "2026-03-12", Model: "anthropic/claude-sonnet-4.6", Usage: 1.5, Requests: 3, PromptTokens: 400, CompletionTokens: 200},
	}

	daily := buildDailyActivity(items)

	assert.Len(t, daily.Days, 3)

	// Days should be sorted chronologically
	assert.Equal(t, "2026-03-10", daily.Days[0].Date)
	assert.Equal(t, "2026-03-11", daily.Days[1].Date)
	assert.Equal(t, "2026-03-12", daily.Days[2].Date)

	// Day 0: 2 models, sorted by spend desc
	assert.Len(t, daily.Days[0].Models, 2)
	assert.Equal(t, "anthropic/claude-opus-4.6", daily.Days[0].Models[0].Model)
	assert.InDelta(t, 5.0, daily.Days[0].Models[0].Spend, 0.001)
	assert.Equal(t, "openai/gpt-5.1", daily.Days[0].Models[1].Model)
	assert.InDelta(t, 3.0, daily.Days[0].Models[1].Spend, 0.001)
	assert.InDelta(t, 8.0, daily.Days[0].Total, 0.001)

	// Day 1: gpt-5.1 is top spender
	assert.Len(t, daily.Days[1].Models, 2)
	assert.Equal(t, "openai/gpt-5.1", daily.Days[1].Models[0].Model)
	assert.InDelta(t, 7.0, daily.Days[1].Models[0].Spend, 0.001)
	assert.InDelta(t, 9.0, daily.Days[1].Total, 0.001)

	// Day 2: single model
	assert.Len(t, daily.Days[2].Models, 1)
	assert.Equal(t, "anthropic/claude-sonnet-4.6", daily.Days[2].Models[0].Model)
	assert.InDelta(t, 1.5, daily.Days[2].Total, 0.001)
}

func TestBuildDailyActivityAggregatesSameModelEndpoints(t *testing.T) {
	// Same model on same date but from different endpoints should be aggregated
	items := []activityItem{
		{Date: "2026-03-10", Model: "anthropic/claude-opus-4.6", Usage: 2.0, Requests: 5, PromptTokens: 500, CompletionTokens: 200},
		{Date: "2026-03-10", Model: "anthropic/claude-opus-4.6", Usage: 3.0, Requests: 8, PromptTokens: 700, CompletionTokens: 300},
	}

	daily := buildDailyActivity(items)

	assert.Len(t, daily.Days, 1)
	assert.Len(t, daily.Days[0].Models, 1)
	assert.InDelta(t, 5.0, daily.Days[0].Models[0].Spend, 0.001)
	assert.InDelta(t, 13.0, daily.Days[0].Models[0].Requests, 0.001)
	assert.InDelta(t, 1200.0, daily.Days[0].Models[0].PromptTokens, 0.001)
	assert.InDelta(t, 500.0, daily.Days[0].Models[0].CompletionTokens, 0.001)
	assert.InDelta(t, 5.0, daily.Days[0].Total, 0.001)
}

func TestBuildDailyActivityEmpty(t *testing.T) {
	daily := buildDailyActivity(nil)
	assert.Empty(t, daily.Days)
}

func logActivity(t *testing.T, label string, a *Activity) {
	t.Helper()
	t.Logf("[%s] spend=$%.2f requests=%.0f prompt=%.0f completion=%.0f reasoning=%.0f",
		label, a.Totals.Spend, a.Totals.Requests,
		a.Totals.PromptTokens, a.Totals.CompletionTokens, a.Totals.ReasoningTokens)
	for _, m := range a.Models {
		t.Logf("  model=%-40s spend=$%8.2f requests=%6.0f prompt=%10.0f completion=%10.0f reasoning=%10.0f",
			m.Model, m.Spend, m.Requests, m.PromptTokens, m.CompletionTokens, m.ReasoningTokens)
	}
}
