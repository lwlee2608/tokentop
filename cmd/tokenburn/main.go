package main

import (
	"fmt"
	"os"
	"time"

	"github.com/lwlee2608/tokenburn/pkg/codex"
)

func main() {
	auth, err := codex.LoadAuth()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Fetching Codex usage limits...")
	usage, err := codex.FetchUsage(auth)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("Plan: %s\n", usage.PlanType)
	fmt.Println()

	// 5-hour window
	primaryRemaining := 100 - usage.PrimaryUsedPercent
	fmt.Printf("5 Hour Usage Limit (%d min window)\n", usage.PrimaryWindowMinutes)
	fmt.Printf("  %.0f%% remaining\n", primaryRemaining)
	fmt.Printf("  Resets: %s (%s)\n", usage.PrimaryResetAt.Local().Format("3:04 PM"), timeUntil(usage.PrimaryResetAt))
	fmt.Println()

	// Weekly window
	secondaryRemaining := 100 - usage.SecondaryUsedPercent
	fmt.Printf("Weekly Usage Limit (%d min window)\n", usage.SecondaryWindowMinutes)
	fmt.Printf("  %.0f%% remaining\n", secondaryRemaining)
	fmt.Printf("  Resets: %s (%s)\n", usage.SecondaryResetAt.Local().Format("Mon Jan 2 3:04 PM"), timeUntil(usage.SecondaryResetAt))
	fmt.Println()

	if usage.CreditsHasCredits {
		fmt.Printf("Credits: %s (unlimited: %v)\n", usage.CreditsBalance, usage.CreditsUnlimited)
	}
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
