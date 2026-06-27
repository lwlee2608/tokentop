package tui

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/lwlee2608/tokentop/internal/config"
	"github.com/lwlee2608/tokentop/pkg/claude"
	"github.com/lwlee2608/tokentop/pkg/codex"
	"github.com/lwlee2608/tokentop/pkg/openrouter"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func forceTrueColorProfile(t *testing.T) {
	t.Helper()

	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(prev)
	})
}

func TestCodexSectionRenderSnapshot(t *testing.T) {
	forceTrueColorProfile(t)

	now := time.Now()
	balance := "12.34"
	m := Model{
		width: 85,
		codexUIConfig: config.CodexUIConfig{
			CodeReview: false,
			PaceTick:   true,
		},
		codexUsage: &codex.Usage{
			PlanType: "Pro",
			RateLimit: codex.RateLimit{
				PrimaryWindow: &codex.UsageWindow{
					UsedPercent:        42,
					LimitWindowSeconds: 5 * 60 * 60,
					ResetAt:            now.Add(2 * time.Hour).Unix(),
				},
				SecondaryWindow: &codex.UsageWindow{
					UsedPercent:        75,
					LimitWindowSeconds: 7 * 24 * 60 * 60,
					ResetAt:            now.Add(4 * 24 * time.Hour).Unix(),
				},
			},
			CodeReviewRateLimit: codex.RateLimit{
				PrimaryWindow: &codex.UsageWindow{
					UsedPercent:        90,
					LimitWindowSeconds: 7 * 24 * 60 * 60,
					ResetAt:            now.Add(3 * 24 * time.Hour).Unix(),
				},
			},
			Credits: codex.UsageCredits{
				HasCredits: true,
				Balance:    &balance,
			},
		},
	}

	got := m.codexSection()
	if testing.Verbose() {
		fmt.Print(got)
	}
	goldenPath := filepath.Join("testdata", "codex_compact.golden")

	if *updateGolden {
		require.NoError(t, os.WriteFile(goldenPath, []byte(got), 0644), "write golden")
	}

	want, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "read golden")
	assert.Equal(t, string(want), got, "codex section mismatch")
}

func TestClaudeSectionRenderSnapshot(t *testing.T) {
	forceTrueColorProfile(t)

	now := time.Now()
	m := Model{
		width: 85,
		claudeUIConfig: config.ClaudeUIConfig{
			PaceTick: true,
		},
		claudeUsage: &claude.Usage{
			SubscriptionType: "Max",
			RateLimitTier:    "default_claude_max_5x",
			SessionLimit: &claude.RateWindow{
				Utilization: 0.42,
				ResetAt:     now.Add(2 * time.Hour),
			},
			WeeklyLimit: &claude.RateWindow{
				Utilization: 0.75,
				ResetAt:     now.Add(4 * 24 * time.Hour),
			},
		},
	}

	got := m.claudeSection()
	if testing.Verbose() {
		fmt.Print(got)
	}
	goldenPath := filepath.Join("testdata", "claude_compact.golden")

	if *updateGolden {
		require.NoError(t, os.WriteFile(goldenPath, []byte(got), 0644), "write golden")
	}

	want, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "read golden")
	assert.Equal(t, string(want), got, "claude section mismatch")
}

func TestCodexSectionShowsLoginHintWhenCredentialsMissing(t *testing.T) {
	forceTrueColorProfile(t)
	m := Model{width: 85, codexEnabled: true}
	got := m.codexSection()
	assert.Contains(t, got, "credentials not found")
	assert.Contains(t, got, codex.LoginHint)
}

func TestCodexSectionShowsLoginHintOnAuthFailure(t *testing.T) {
	forceTrueColorProfile(t)
	m := Model{
		width:           85,
		codexEnabled:    true,
		codexAuth:       &codex.Auth{},
		codexErr:        "Codex API 401: invalid token: unauthorized",
		codexAuthFailed: true,
	}
	got := m.codexSection()
	assert.Contains(t, got, "401")
	assert.Contains(t, got, codex.LoginHint)
}

func TestClaudeSectionShowsLoginHintWhenCredentialsMissing(t *testing.T) {
	forceTrueColorProfile(t)
	m := Model{width: 85, claudeEnabled: true}
	got := m.claudeSection()
	assert.Contains(t, got, "credentials not found")
	assert.Contains(t, got, claude.LoginHint)
}

func TestClaudeSectionShowsLoginHintOnAuthFailure(t *testing.T) {
	forceTrueColorProfile(t)
	m := Model{
		width:            85,
		claudeEnabled:    true,
		claudeAuth:       &claude.Auth{},
		claudeErr:        "Anthropic API 401: invalid token: unauthorized",
		claudeAuthFailed: true,
	}
	got := m.claudeSection()
	assert.Contains(t, got, "401")
	assert.Contains(t, got, claude.LoginHint)
}

func TestOpenRouterSectionShowsHintWhenCredentialsMissing(t *testing.T) {
	forceTrueColorProfile(t)
	m := Model{width: 85, orEnabled: true}
	got := m.orSection()
	assert.Contains(t, got, "credentials not found")
	assert.Contains(t, got, openrouter.LoginHint)
}

func TestOpenRouterSectionShowsHintOnAuthFailure(t *testing.T) {
	forceTrueColorProfile(t)
	m := Model{
		width:        85,
		orEnabled:    true,
		orAuth:       &openrouter.Auth{APIKey: "sk-or-bad"},
		orErr:        "OpenRouter API /key 401: invalid token: unauthorized",
		orAuthFailed: true,
	}
	got := m.orSection()
	assert.Contains(t, got, "401")
	assert.Contains(t, got, openrouter.LoginHint)
}

func TestOpenRouterSectionRenderSnapshot(t *testing.T) {
	forceTrueColorProfile(t)

	raw, err := os.ReadFile(filepath.Join("testdata", "openrouter_activity.json"))
	require.NoError(t, err, "read activity json")

	activity, daily, err := openrouter.ParseActivityJSON(raw)
	require.NoError(t, err, "parse activity json")

	m := Model{
		width: 85,
		orUIConfig: config.OpenRouterUIConfig{
			Summary:    true,
			DailySpend: true,
			TopModels:  true,
			Metric:     "tokens",
		},
		orMetric: metricTokens,
		orUsage: &openrouter.Usage{
			Key: openrouter.KeyUsage{
				Label:           "tokentop",
				IsManagementKey: true,
			},
			Credits: &openrouter.Credits{
				Total:     2000.00,
				Used:      1850.72,
				Remaining: 149.28,
			},
			Activity:      activity,
			DailyActivity: daily,
		},
	}

	got := m.orSection()
	if testing.Verbose() {
		fmt.Print(got)
	}
	goldenPath := filepath.Join("testdata", "openrouter_tokens.golden")

	if *updateGolden {
		require.NoError(t, os.WriteFile(goldenPath, []byte(got), 0644), "write golden")
	}

	want, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "read golden")
	assert.Equal(t, string(want), got, "openrouter section mismatch")
}

func TestOpenRouterCompactDailyKeySnapshot(t *testing.T) {
	forceTrueColorProfile(t)

	m := Model{
		width:      85,
		orUIConfig: config.OpenRouterUIConfig{},
		orAuth: &openrouter.Auth{
			APIKey: "sk-or-v1-771xxxxxxxa29",
		},
		orUsage: &openrouter.Usage{
			Key: openrouter.KeyUsage{
				Limit:           3.00,
				LimitRemaining:  1.05,
				LimitReset:      "daily",
				IsManagementKey: false,
				UsageDaily:      1.95,
				UsageWeekly:     3.1272,
				UsageMonthly:    0.0,
			},
		},
	}

	got := m.orSection()
	if testing.Verbose() {
		fmt.Print(got)
	}
	goldenPath := filepath.Join("testdata", "openrouter_compact_daily.golden")

	if *updateGolden {
		require.NoError(t, os.WriteFile(goldenPath, []byte(got), 0644), "write golden")
	}

	want, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "read golden")
	assert.Equal(t, string(want), got, "openrouter compact section mismatch")
}

func TestClipLineKeepsWideRunesWithinMaxWidth(t *testing.T) {
	got := clipLine("Status: 警告 warning", 10)

	assert.LessOrEqual(t, lipgloss.Width(got), 10)
}

func TestBuildBarCellsStartOfWindowMarksUsageOverPace(t *testing.T) {
	cells := buildBarCells(50, 0, 10)

	for i := range 5 {
		assert.Equalf(t, barCellOverPace, cells[i], "cell %d", i)
	}
	for i := 5; i < len(cells); i++ {
		assert.Equalf(t, barCellEmpty, cells[i], "cell %d", i)
	}
}

func TestBuildBarCellsWithoutPaceDataUsesNormalFill(t *testing.T) {
	cells := buildBarCells(50, -1, 10)

	for i := range 5 {
		assert.Equalf(t, barCellFilled, cells[i], "cell %d", i)
	}
	for i := 5; i < len(cells); i++ {
		assert.Equalf(t, barCellEmpty, cells[i], "cell %d", i)
	}
}

func TestOverPaceColorKeepsRedBarsDistinct(t *testing.T) {
	assert.Equal(t, brightRed, overPaceColor(red))
}
