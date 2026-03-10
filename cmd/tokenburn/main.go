package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/lwlee2608/tokenburn/pkg/codex"
)

var AppVersion = "dev"

const (
	barWidth = 25

	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	fgWhite = "\033[97m"
	fgGray  = "\033[90m"

	fgGreen  = "\033[32m"
	fgYellow = "\033[33m"
	fgRed    = "\033[31m"

	bgGreen  = "\033[42m"
	bgYellow = "\033[43m"
	bgRed    = "\033[41m"
	bgGray   = "\033[48;5;237m"
)

func main() {
	auth, err := codex.LoadAuth()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(dim + "Fetching Codex usage limits..." + reset)
	usage, err := codex.FetchUsage(auth)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Print("\033[1A\033[2K") // clear "Fetching..." line

	// Header
	fmt.Printf("%s%s tokenburn %s %s%s%s\n", bold+fgWhite, "┌──────────", AppVersion, strings.Repeat("─", barWidth+2), "┐", reset)
	fmt.Printf("%s Plan: %-20s%s\n", fgGray, usage.PlanType, reset)
	fmt.Println()

	// 5-hour window
	printUsageBar(
		"5h Limit",
		usage.PrimaryUsedPercent,
		fmt.Sprintf("resets %s (%s)", usage.PrimaryResetAt.Local().Format("3:04 PM"), timeUntil(usage.PrimaryResetAt)),
	)
	fmt.Println()

	// Weekly window
	printUsageBar(
		"Weekly",
		usage.SecondaryUsedPercent,
		fmt.Sprintf("resets %s (%s)", usage.SecondaryResetAt.Local().Format("Mon Jan 2 3:04 PM"), timeUntil(usage.SecondaryResetAt)),
	)

	if usage.CreditsHasCredits {
		fmt.Println()
		fmt.Printf("%s Credits: %s (unlimited: %v)%s\n", fgGray, usage.CreditsBalance, usage.CreditsUnlimited, reset)
	}

	fmt.Printf("%s%s%s\n", dim, strings.Repeat("─", barWidth+38), reset)
}

func printUsageBar(label string, usedPercent float64, resetInfo string) {
	used := math.Min(usedPercent, 100)
	remaining := 100 - used

	filledCount := int(math.Round(used / 100 * barWidth))
	emptyCount := barWidth - filledCount

	// Color based on usage level
	var barFg, barBg, pctColor string
	switch {
	case used >= 90:
		barFg, barBg, pctColor = fgRed, bgRed, fgRed
	case used >= 70:
		barFg, barBg, pctColor = fgYellow, bgYellow, fgYellow
	default:
		barFg, barBg, pctColor = fgGreen, bgGreen, fgGreen
	}

	// Build bar: filled portion + empty portion
	filled := barBg + fgWhite + strings.Repeat(" ", filledCount) + reset
	empty := bgGray + strings.Repeat(" ", emptyCount) + reset

	// Label line
	fmt.Printf(" %s%-10s%s\n", bold+fgWhite, label, reset)

	// Bar line with used/free percentages
	fmt.Printf("  Used:%s%4.0f%%%s  %s%s  %s%4.0f%% free%s\n",
		pctColor, used, reset,
		filled, empty,
		barFg, remaining, reset,
	)

	// Reset time
	fmt.Printf("  %s%s%s\n", dim, resetInfo, reset)
}

func timeUntil(t time.Time) string {
	d := time.Until(t)
	if d < 0 {
		return "expired"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("in %dh %dm", h, m)
	}
	return fmt.Sprintf("in %dm", m)
}
