package openrouter

import (
	"os"
	"testing"
)

func TestFetchUsageLive(t *testing.T) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		t.Skip("OPENROUTER_API_KEY is not set")
	}

	usage, err := FetchUsage(&Auth{APIKey: apiKey})
	if err != nil {
		t.Fatalf("FetchUsage: %v", err)
	}

	t.Logf("label=%s management=%t limit=%0.2f remaining=%0.2f total=%0.2f daily=%0.2f weekly=%0.2f monthly=%0.2f",
		usage.Key.Label,
		usage.Key.IsManagementKey,
		usage.Key.Limit,
		usage.Key.LimitRemaining,
		usage.Key.Usage,
		usage.Key.UsageDaily,
		usage.Key.UsageWeekly,
		usage.Key.UsageMonthly,
	)
}
