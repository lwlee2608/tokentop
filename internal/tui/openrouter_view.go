package tui

import (
	"fmt"
	"strings"

	"github.com/lwlee2608/tokentop/pkg/openrouter"
)

const (
	maxModels = 8
	maxKeys   = 10
)

func (m Model) orSection() string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render(" OpenRouter"))
	b.WriteByte('\n')

	if m.orUsage == nil && m.orErr == "" {
		b.WriteString(dimStyle.Render("  Loading..."))
		b.WriteByte('\n')
		return b.String()
	}

	if m.orErr != "" {
		c := yellow
		if m.orUsage == nil {
			c = red
		}
		b.WriteString(pctStyle(c).Render(fmt.Sprintf("  ⚠️  %s (retry %d/%d)", m.orErr, m.orRetries, maxRetries)))
		b.WriteByte('\n')
		if m.orUsage == nil {
			return b.String()
		}
	}

	u := m.orUsage
	bw := m.barWidth()

	keyLabel := u.Key.Label
	switch {
	case u.Key.IsFreeTier:
		keyLabel += " (free tier)"
	case u.Key.IsManagementKey:
		keyLabel += " (management)"
	}
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Key: %s", keyLabel)))
	b.WriteByte('\n')

	if u.Key.Limit > 0 {
		b.WriteByte('\n')
		usedPct := (u.Key.Limit - u.Key.LimitRemaining) / u.Key.Limit * 100
		b.WriteString(renderBar("Credit Limit", usedPct, bw,
			fmt.Sprintf("$%.4f remaining (resets %s)", u.Key.LimitRemaining, u.Key.LimitReset),
		))
		b.WriteByte('\n')
	}

	b.WriteString(dimStyle.Render(fmt.Sprintf("  Usage — Daily: $%.4f | Weekly: $%.4f | Monthly: $%.4f",
		u.Key.UsageDaily, u.Key.UsageWeekly, u.Key.UsageMonthly)))
	b.WriteByte('\n')

	if u.Key.IsManagementKey {
		b.WriteString(renderORSummary(u))
		b.WriteString(m.renderORModels(u))
		b.WriteString(renderORKeys(u))
	}

	b.WriteByte('\n')
	return b.String()
}

func renderORSummary(u *openrouter.Usage) string {
	if u.Credits == nil && u.Activity == nil {
		return ""
	}
	var b strings.Builder
	b.WriteByte('\n')
	parts := make([]string, 0, 3)
	if u.Credits != nil {
		parts = append(parts, fmt.Sprintf("Credits $%.2f/$%.2f left", u.Credits.Remaining, u.Credits.Total))
	}
	if u.Activity != nil {
		t := u.Activity.Totals
		parts = append(parts, fmt.Sprintf("Spend $%.2f | %.0f req", t.Spend, t.Requests))
		tokens := fmt.Sprintf("%s in + %s out", formatTokens(t.PromptTokens), formatTokens(t.CompletionTokens))
		if t.ReasoningTokens > 0 {
			tokens += fmt.Sprintf(" + %s reason", formatTokens(t.ReasoningTokens))
		}
		parts = append(parts, tokens)
	}
	b.WriteString(dimStyle.Render("  " + strings.Join(parts, " | ")))
	b.WriteByte('\n')
	return b.String()
}

func (m Model) renderORModels(u *openrouter.Usage) string {
	if u.Activity == nil || len(u.Activity.Models) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteByte('\n')
	b.WriteString("  " + labelStyle.Render("Top Models") + "\n")

	models := u.Activity.Models
	if len(models) > maxModels {
		models = models[:maxModels]
	}

	maxSpend := models[0].Spend
	barWidth := m.modelBarWidth()
	for i, model := range models {
		label := truncate(model.Model, 22)
		b.WriteString(fmt.Sprintf("  %s  %s  %s  %s\n",
			dimStyle.Render(fmt.Sprintf("%-22s", label)),
			dimStyle.Render(fmt.Sprintf("$%7.2f", model.Spend)),
			renderModelBar(model.Spend, maxSpend, barWidth, i),
			dimStyle.Render(fmt.Sprintf("%4.0f req", model.Requests)),
		))
	}
	return b.String()
}

func (m Model) modelBarWidth() int {
	w := m.width - 44
	if w < 8 {
		return 8
	}
	if w > 28 {
		return 28
	}
	return w
}

func renderModelBar(spend, maxSpend float64, width int, colorIndex int) string {
	if width < 1 {
		width = 1
	}
	filled := width
	if maxSpend > 0 {
		filled = int(spend / maxSpend * float64(width))
	}
	if spend > 0 && filled == 0 {
		filled = 1
	}
	if filled > width {
		filled = width
	}

	return modelBarFilledStyle(colorIndex).Render(strings.Repeat("█", filled)) +
		modelBarEmptyStyle.Render(strings.Repeat("░", width-filled))
}

func renderORKeys(u *openrouter.Usage) string {
	if len(u.APIKeys) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteByte('\n')
	b.WriteString("  " + labelStyle.Render("API Keys") + "\n")

	keys := u.APIKeys
	if len(keys) > maxKeys {
		keys = keys[:maxKeys]
	}
	for _, k := range keys {
		name := k.Label
		if name == "" {
			name = k.Name
		}
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %-18s d:$%6.2f  w:$%6.2f  m:$%6.2f",
			truncate(name, 18), k.UsageDaily, k.UsageWeekly, k.UsageMonthly)))
		b.WriteByte('\n')
	}
	return b.String()
}

func formatTokens(n float64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", n/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", n/1_000)
	default:
		return fmt.Sprintf("%.0f", n)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
