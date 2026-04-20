package tui

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/lwlee2608/tokentop/internal/config"
	"github.com/lwlee2608/tokentop/pkg/codex"
	"github.com/muesli/termenv"
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
			CodeReview: true,
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
	goldenPath := filepath.Join("testdata", "codex_compact.golden")

	if *updateGolden {
		if err := os.WriteFile(goldenPath, []byte(got), 0644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if got != string(want) {
		t.Errorf("codex section mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestBuildBarCellsStartOfWindowMarksUsageOverPace(t *testing.T) {
	cells := buildBarCells(50, 0, 10)

	for i := 0; i < 5; i++ {
		if cells[i] != barCellOverPace {
			t.Fatalf("cell %d = %v, want %v", i, cells[i], barCellOverPace)
		}
	}
	for i := 5; i < len(cells); i++ {
		if cells[i] != barCellEmpty {
			t.Fatalf("cell %d = %v, want %v", i, cells[i], barCellEmpty)
		}
	}
}

func TestBuildBarCellsWithoutPaceDataUsesNormalFill(t *testing.T) {
	cells := buildBarCells(50, -1, 10)

	for i := 0; i < 5; i++ {
		if cells[i] != barCellFilled {
			t.Fatalf("cell %d = %v, want %v", i, cells[i], barCellFilled)
		}
	}
	for i := 5; i < len(cells); i++ {
		if cells[i] != barCellEmpty {
			t.Fatalf("cell %d = %v, want %v", i, cells[i], barCellEmpty)
		}
	}
}

func TestOverPaceColorKeepsRedBarsDistinct(t *testing.T) {
	if got := overPaceColor(red); got != brightRed {
		t.Fatalf("overPaceColor(red) = %q, want %q", got, brightRed)
	}
}
