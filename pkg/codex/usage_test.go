package codex

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsageResponseUnmarshal(t *testing.T) {
	tests := []struct {
		name                 string
		body                 string
		primaryWindowSeconds int
		hasSecondaryWindow   bool
	}{
		{
			name: "five hour and weekly windows",
			body: `{
				"plan_type": "plus",
				"rate_limit": {
					"allowed": true,
					"limit_reached": false,
					"primary_window": {"used_percent": 42, "limit_window_seconds": 18000, "reset_after_seconds": 7200, "reset_at": 1784000000},
					"secondary_window": {"used_percent": 75, "limit_window_seconds": 604800, "reset_after_seconds": 345600, "reset_at": 1784345600}
				}
			}`,
			primaryWindowSeconds: 5 * 60 * 60,
			hasSecondaryWindow:   true,
		},
		{
			name: "weekly window only",
			body: `{
				"plan_type": "plus",
				"rate_limit": {
					"allowed": true,
					"limit_reached": false,
					"primary_window": {"used_percent": 0, "limit_window_seconds": 604800, "reset_after_seconds": 518740, "reset_at": 1784516033},
					"secondary_window": null
				},
				"code_review_rate_limit": null,
				"additional_rate_limits": null
			}`,
			primaryWindowSeconds: 7 * 24 * 60 * 60,
			hasSecondaryWindow:   false,
		},
		{
			name: "monthly window only",
			body: `{
				"plan_type": "enterprise",
				"rate_limit": {
					"allowed": true,
					"limit_reached": false,
					"primary_window": {"used_percent": 32, "limit_window_seconds": 2592000, "reset_after_seconds": 1209600, "reset_at": 1785729600},
					"secondary_window": null
				},
				"code_review_rate_limit": null
			}`,
			primaryWindowSeconds: 30 * 24 * 60 * 60,
			hasSecondaryWindow:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var usage Usage
			require.NoError(t, json.Unmarshal([]byte(tt.body), &usage))
			require.NotNil(t, usage.RateLimit.PrimaryWindow)
			assert.Equal(t, tt.primaryWindowSeconds, usage.RateLimit.PrimaryWindow.LimitWindowSeconds)
			assert.Equal(t, tt.hasSecondaryWindow, usage.RateLimit.SecondaryWindow != nil)
			assert.Nil(t, usage.CodeReviewRateLimit)
		})
	}
}

func TestFetchUsageLive(t *testing.T) {
	if os.Getenv("CODEX_LIVE_TEST") == "" {
		t.Skip("CODEX_LIVE_TEST is not set")
	}

	auth, err := LoadAuth()
	require.NoError(t, err)
	usage, err := FetchUsage(auth)
	require.NoError(t, err)

	body, err := json.MarshalIndent(usage, "", "  ")
	require.NoError(t, err)
	t.Logf("decoded usage JSON:\n%s", body)
}
