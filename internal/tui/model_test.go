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
	"github.com/lwlee2608/tokentop/pkg/codex"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func TestCodexSectionRenderSnapshot(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	now := time.Now()
	balance := "12.34"
	m := Model{
		width: 80,
		codexUIConfig: config.CodexUIConfig{
			Compact:    true,
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
	fmt.Print(got)
	goldenPath := filepath.Join("testdata", "codex_compact.golden")

	if *updateGolden {
		require.NoError(t, os.WriteFile(goldenPath, []byte(got), 0644), "write golden")
	}

	want, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "read golden")
	assert.Equal(t, string(want), got, "codex section mismatch")
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
