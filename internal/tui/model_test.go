package tui

import "testing"

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
