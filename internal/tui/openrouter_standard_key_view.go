package tui

import (
	"fmt"
	"strings"

	"github.com/lwlee2608/tokentop/pkg/openrouter"
)

func (m Model) renderORStandardBody(u *openrouter.Usage) string {
	var b strings.Builder
	bw := m.barWidth()

	periodLabel, hasPeriod := parseLimitResetPeriod(u.Key.LimitReset)
	barCoversPeriod := false
	if u.Key.Limit > 0 {
		usedPct := (u.Key.Limit - u.Key.LimitRemaining) / u.Key.Limit * 100
		label := "Cr Limit"
		if hasPeriod {
			label = fmt.Sprintf("%-8s", periodLabel)
			barCoversPeriod = true
		}
		info := truncate(u.Key.LimitReset, compactResetWidth)
		if hasPeriod {
			info = fmt.Sprintf("$%.2f", u.Key.LimitRemaining)
		}
		b.WriteString(renderCompactBar(label, usedPct, -1, bw, info))
	}

	parts := make([]string, 0, 3)
	if !barCoversPeriod || periodLabel != "Daily" {
		parts = append(parts, fmt.Sprintf("Daily: $%.4f", u.Key.UsageDaily))
	}
	if !barCoversPeriod || periodLabel != "Weekly" {
		parts = append(parts, fmt.Sprintf("Weekly: $%.4f", u.Key.UsageWeekly))
	}
	if !barCoversPeriod || periodLabel != "Monthly" {
		parts = append(parts, fmt.Sprintf("Monthly: $%.4f", u.Key.UsageMonthly))
	}
	if len(parts) > 0 {
		b.WriteString(dimStyle.Render("  Usage          " + strings.Join(parts, " | ")))
		b.WriteByte('\n')
	}

	return b.String()
}
