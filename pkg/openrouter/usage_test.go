package openrouter

import (
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
		t.Logf("activity totals: spend=$%.2f requests=%.0f prompt_tokens=%.0f completion_tokens=%.0f reasoning_tokens=%.0f",
			usage.Activity.Totals.Spend, usage.Activity.Totals.Requests,
			usage.Activity.Totals.PromptTokens, usage.Activity.Totals.CompletionTokens, usage.Activity.Totals.ReasoningTokens)
		for _, m := range usage.Activity.Models {
			t.Logf("  model=%-40s spend=$%8.2f requests=%6.0f prompt=%10.0f completion=%10.0f reasoning=%10.0f",
				m.Model, m.Spend, m.Requests, m.PromptTokens, m.CompletionTokens, m.ReasoningTokens)
		}
	}
}
