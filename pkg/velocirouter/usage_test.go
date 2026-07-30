package velocirouter

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleUsage = `{
  "key": {
    "id": "3f1c2d4e-5a6b-4c8d-9e0f-1a2b3c4d5e6f",
    "name": "laptop",
    "key_prefix": "rt-EXAMPLE00",
    "created_at": "2026-01-14T09:20:00Z"
  },
  "lifetime": {
    "requests": 1284,
    "prompt_tokens": 4820000,
    "completion_tokens": 180000,
    "total_tokens": 5000000,
    "cached_tokens": 2400000,
    "cache_write_tokens": 120000,
    "cost_usd": 42.5,
    "last_used_at": "2026-02-09T11:42:03Z"
  },
  "windows": {
    "weekly": {
      "limit_usd": null,
      "spent_usd": 18.2,
      "reset_at": "2026-02-16T00:00:00Z"
    },
    "5h": {
      "limit_usd": 20,
      "spent_usd": 3.75,
      "reset_at": "2026-02-09T13:00:00Z"
    }
  }
}`

func TestParseUsageJSON(t *testing.T) {
	usage, err := ParseUsageJSON([]byte(sampleUsage))
	require.NoError(t, err)

	assert.Equal(t, "laptop", usage.Key.Name)
	assert.Equal(t, "rt-EXAMPLE00", usage.Key.Prefix)

	assert.Equal(t, float64(1284), usage.Lifetime.Requests)
	assert.Equal(t, float64(5000000), usage.Lifetime.TotalTokens)
	assert.Equal(t, 42.5, usage.Lifetime.CostUSD)
	require.NotNil(t, usage.Lifetime.LastUsedAt)

	require.Len(t, usage.Windows, 2)

	// shortest window first, regardless of JSON map order
	short := usage.Windows[0]
	assert.Equal(t, "5h", short.Name)
	assert.Equal(t, 5*time.Hour, short.Duration)
	assert.True(t, short.HasLimit())
	assert.InDelta(t, 18.75, short.UsedPercent(), 0.001)

	week := usage.Windows[1]
	assert.Equal(t, "weekly", week.Name)
	assert.Equal(t, 7*24*time.Hour, week.Duration)
	assert.False(t, week.HasLimit())
	assert.Equal(t, float64(0), week.UsedPercent())
	assert.Equal(t, 18.2, week.SpentUSD)
}

func TestParseUsageJSONUnusedKey(t *testing.T) {
	usage, err := ParseUsageJSON([]byte(`{"key":{"name":"fresh"},"lifetime":{"last_used_at":null},"windows":{}}`))
	require.NoError(t, err)

	assert.Nil(t, usage.Lifetime.LastUsedAt)
	assert.Empty(t, usage.Windows)
}

func TestParseUsageJSONUnknownWindowSortsLast(t *testing.T) {
	usage, err := ParseUsageJSON([]byte(`{"windows":{
		"lunar":{"limit_usd":5,"spent_usd":1},
		"weekly":{"limit_usd":50,"spent_usd":1},
		"5h":{"limit_usd":20,"spent_usd":1}
	}}`))
	require.NoError(t, err)

	names := make([]string, len(usage.Windows))
	for i, w := range usage.Windows {
		names[i] = w.Name
	}
	assert.Equal(t, []string{"5h", "weekly", "lunar"}, names)
}

func TestWindowDuration(t *testing.T) {
	tests := []struct {
		name string
		want time.Duration
	}{
		{name: "5h", want: 5 * time.Hour},
		{name: "30m", want: 30 * time.Minute},
		{name: "daily", want: 24 * time.Hour},
		{name: "weekly", want: 7 * 24 * time.Hour},
		{name: "monthly", want: 30 * 24 * time.Hour},
		{name: "3d", want: 3 * 24 * time.Hour},
		{name: "lunar", want: 0},
		{name: "", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, WindowDuration(tt.name))
		})
	}
}

func TestFetchUsageLive(t *testing.T) {
	if os.Getenv("VELOCIROUTER_LIVE_TEST") == "" {
		t.Skip("VELOCIROUTER_LIVE_TEST is not set")
	}

	auth, err := LoadAuth()
	require.NoError(t, err)
	usage, err := FetchUsage(auth)
	require.NoError(t, err)

	body, err := json.MarshalIndent(usage, "", "  ")
	require.NoError(t, err)
	t.Logf("decoded usage JSON:\n%s", body)
}
