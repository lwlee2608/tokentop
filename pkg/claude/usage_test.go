package claude

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLimits(t *testing.T) {
	body := `{
		"five_hour": {"utilization": 15.0, "resets_at": "2026-07-02T05:40:00+00:00"},
		"limits": [
			{"kind": "session", "group": "session", "percent": 15, "severity": "normal", "resets_at": "2026-07-02T05:40:00+00:00", "scope": null, "is_active": true},
			{"kind": "weekly_all", "group": "weekly", "percent": 2, "severity": "normal", "resets_at": "2026-07-02T08:00:00+00:00", "scope": null, "is_active": false},
			{"kind": "weekly_scoped", "group": "weekly", "percent": 3, "severity": "normal", "resets_at": "2026-07-02T07:59:59+00:00", "scope": {"model": {"id": null, "display_name": "Fable"}, "surface": null}, "is_active": false}
		]
	}`

	limits, err := parseLimits([]byte(body))
	require.NoError(t, err)
	for i := range limits {
		limits[i].ResetAt = limits[i].ResetAt.UTC()
	}

	assert.Equal(t, []Limit{
		{Kind: "session", Group: "session", Percent: 15, ResetAt: time.Date(2026, 7, 2, 5, 40, 0, 0, time.UTC)},
		{Kind: "weekly_all", Group: "weekly", Percent: 2, ResetAt: time.Date(2026, 7, 2, 8, 0, 0, 0, time.UTC)},
		{Kind: "weekly_scoped", Group: "weekly", Model: "Fable", Percent: 3, ResetAt: time.Date(2026, 7, 2, 7, 59, 59, 0, time.UTC)},
	}, limits)
}

func TestParseLimitsInvalidJSON(t *testing.T) {
	_, err := parseLimits([]byte("not json"))
	assert.Error(t, err)
}
