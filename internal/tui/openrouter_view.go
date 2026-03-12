package tui

import (
	"fmt"
	"math"
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
		b.WriteString(m.renderORDailyChart(u))
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

const (
	chartMaxHeight = 10
	chartMaxDays   = 30
	chartTopModels = 6
)

func (m Model) renderORDailyChart(u *openrouter.Usage) string {
	if u.DailyActivity == nil || len(u.DailyActivity.Days) == 0 {
		return ""
	}

	days := u.DailyActivity.Days
	if len(days) > chartMaxDays {
		days = days[len(days)-chartMaxDays:]
	}

	// Find top models across all days by total spend
	modelSpend := make(map[string]float64)
	for _, day := range days {
		for _, model := range day.Models {
			modelSpend[model.Model] += model.Spend
		}
	}
	topModels := topNModels(modelSpend, chartTopModels)
	topSet := make(map[string]bool, len(topModels))
	for _, m := range topModels {
		topSet[m] = true
	}

	// Find max daily total for scaling
	var maxTotal float64
	for _, day := range days {
		if day.Total > maxTotal {
			maxTotal = day.Total
		}
	}
	if maxTotal == 0 {
		return ""
	}

	// Build stacked columns: each day is a column of colored blocks
	height := chartMaxHeight
	var b strings.Builder
	b.WriteByte('\n')
	b.WriteString("  " + labelStyle.Render("Daily Spend") + "\n")

	// Y-axis labels + chart rows (top to bottom)
	for row := height; row >= 1; row-- {
		threshold := maxTotal * float64(row) / float64(height)
		if row == height {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  $%5.0f │", maxTotal)))
		} else if row == height/2 {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  $%5.0f │", maxTotal/2)))
		} else if row == 1 {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  $%5.0f │", maxTotal/float64(height))))
		} else {
			b.WriteString(dimStyle.Render("        │"))
		}

		for _, day := range days {
			if day.Total >= threshold {
				// Determine which model "owns" this segment
				colorIdx := segmentColor(day, threshold, maxTotal, float64(height), topModels, topSet)
				b.WriteString(modelBarFilledStyle(colorIdx).Render("▐█"))
			} else {
				b.WriteString("  ")
			}
		}
		b.WriteByte('\n')
	}

	// X-axis
	b.WriteString(dimStyle.Render("        └" + strings.Repeat("──", len(days))))
	b.WriteByte('\n')

	// Date labels (show first, middle, last)
	if len(days) > 0 {
		dateLineWidth := 2 * len(days)
		dateLine := make([]byte, dateLineWidth)
		for i := range dateLine {
			dateLine[i] = ' '
		}
		labels := []int{0}
		if len(days) > 2 {
			labels = append(labels, len(days)/2)
		}
		if len(days) > 1 {
			labels = append(labels, len(days)-1)
		}
		b.WriteString("         ")
		for i, day := range days {
			isLabel := false
			for _, li := range labels {
				if i == li {
					isLabel = true
					break
				}
			}
			if isLabel {
				lbl := day.Date[5:] // "MM-DD"
				b.WriteString(dimStyle.Render(lbl))
				// Pad to maintain alignment
				remaining := 2 - len(lbl)
				if remaining > 0 {
					b.WriteString(strings.Repeat(" ", remaining))
				}
			} else {
				b.WriteString("  ")
			}
		}
		b.WriteByte('\n')
	}

	// Legend
	b.WriteString("  ")
	for i, model := range topModels {
		shortName := truncate(modelShortName(model), 14)
		b.WriteString(modelBarFilledStyle(i).Render("█") + " " + dimStyle.Render(shortName) + "  ")
	}
	b.WriteByte('\n')

	return b.String()
}

func segmentColor(day openrouter.DailyUsage, threshold, maxTotal, height float64, topModels []string, topSet map[string]bool) int {
	// Walk up the stacked bar to find which model occupies this row
	cumulative := 0.0
	segmentSize := maxTotal / height

	// Group spend by category (top models + others)
	spendByModel := make(map[string]float64)
	var othersSpend float64
	for _, model := range day.Models {
		if topSet[model.Model] {
			spendByModel[model.Model] += model.Spend
		} else {
			othersSpend += model.Spend
		}
	}

	// Stack in order: top models first, then others
	for i, name := range topModels {
		spend := spendByModel[name]
		cumulative += spend
		if cumulative >= threshold-segmentSize*0.5 {
			return i
		}
	}
	cumulative += othersSpend
	if cumulative >= threshold-segmentSize*0.5 {
		return len(topModels) % len(modelBarColors)
	}

	return 0
}

func topNModels(modelSpend map[string]float64, n int) []string {
	type ms struct {
		model string
		spend float64
	}
	all := make([]ms, 0, len(modelSpend))
	for m, s := range modelSpend {
		all = append(all, ms{m, s})
	}
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].spend > all[i].spend {
				all[i], all[j] = all[j], all[i]
			}
		}
	}
	result := make([]string, 0, int(math.Min(float64(n), float64(len(all)))))
	for i := 0; i < len(all) && i < n; i++ {
		result = append(result, all[i].model)
	}
	return result
}

func modelShortName(model string) string {
	// "anthropic/claude-opus-4.6" -> "claude-opus-4.6"
	if idx := strings.LastIndex(model, "/"); idx >= 0 {
		return model[idx+1:]
	}
	return model
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
