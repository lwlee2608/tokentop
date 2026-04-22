package tui

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/lwlee2608/tokentop/pkg/openrouter"
)

const (
	maxModels = 7
	maxKeys   = 10
)

type orMetric string

const (
	metricSpend    orMetric = "spend"
	metricRequests orMetric = "requests"
	metricTokens   orMetric = "tokens"
)

func parseMetric(s string) orMetric {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "requests", "request", "req":
		return metricRequests
	case "tokens", "token":
		return metricTokens
	default:
		return metricSpend
	}
}

func (m orMetric) next() orMetric {
	switch m {
	case metricSpend:
		return metricRequests
	case metricRequests:
		return metricTokens
	default:
		return metricSpend
	}
}

func (m orMetric) chartTitle() string {
	switch m {
	case metricRequests:
		return "Daily Requests"
	case metricTokens:
		return "Daily Tokens"
	default:
		return "Daily Spend"
	}
}

func (m orMetric) modelsTitle() string {
	switch m {
	case metricRequests:
		return "Top Models (by requests)"
	case metricTokens:
		return "Top Models (by tokens)"
	default:
		return "Top Models"
	}
}

func modelMetricValue(mu openrouter.ModelUsage, metric orMetric) float64 {
	switch metric {
	case metricRequests:
		return mu.Requests
	case metricTokens:
		return mu.PromptTokens + mu.CompletionTokens + mu.ReasoningTokens
	default:
		return mu.Spend
	}
}

func dayMetricTotal(day openrouter.DailyUsage, metric orMetric) float64 {
	if metric == metricSpend {
		return day.Total
	}
	var total float64
	for _, m := range day.Models {
		total += modelMetricValue(m, metric)
	}
	return total
}

func metricAxisLabel(v float64, metric orMetric) string {
	switch metric {
	case metricTokens:
		return formatTokens(v)
	default:
		return fmt.Sprintf("%.0f", v)
	}
}

// formatMetricColumn renders the primary value column in Top Models (width 8).
func formatMetricColumn(v float64, metric orMetric) string {
	switch metric {
	case metricRequests:
		return fmt.Sprintf("%8.0f", v)
	case metricTokens:
		return fmt.Sprintf("%8s", formatTokens(v))
	default:
		return fmt.Sprintf("$%7.2f", v)
	}
}

// formatMetricSecondary renders the trailing context column in Top Models (width 8).
func formatMetricSecondary(mu openrouter.ModelUsage, metric orMetric) string {
	if metric == metricSpend {
		return fmt.Sprintf("%4.0f req", mu.Requests)
	}
	return fmt.Sprintf("$%7.2f", mu.Spend)
}

func (m Model) orSection() string {
	return sectionBox("OpenRouter", m.orSectionBody(), m.width)
}

func (m Model) orSectionBody() string {
	var b strings.Builder

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

	if u.Key.Limit > 0 {
		b.WriteByte('\n')
		usedPct := (u.Key.Limit - u.Key.LimitRemaining) / u.Key.Limit * 100
		b.WriteString(renderBar("Credit Limit", usedPct, -1, bw,
			fmt.Sprintf("$%.4f remaining (resets %s)", u.Key.LimitRemaining, u.Key.LimitReset),
		))
		b.WriteByte('\n')
	}

	if !u.Key.IsManagementKey {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Usage — Daily: $%.4f | Weekly: $%.4f | Monthly: $%.4f",
			u.Key.UsageDaily, u.Key.UsageWeekly, u.Key.UsageMonthly)))
		b.WriteByte('\n')
	}

	if u.Key.IsManagementKey {
		if m.orUIConfig.Summary {
			b.WriteString(renderORSummary(u))
		}
		if m.orUIConfig.DailySpend {
			b.WriteString(m.renderORDailyChart(u))
		}
		if m.orUIConfig.TopModels {
			b.WriteString(m.renderORModels(u))
		}
		if m.orUIConfig.APIKeys {
			b.WriteString(renderORKeys(u))
		}
	}

	b.WriteByte('\n')
	return b.String()
}

func renderORSummary(u *openrouter.Usage) string {
	if u.Credits == nil && u.Activity == nil {
		return ""
	}
	var b strings.Builder
	// b.WriteByte('\n')
	parts := make([]string, 0, 3)
	if u.Credits != nil {
		parts = append(parts, fmt.Sprintf("Credits $%.2f/$%.2f left", u.Credits.Remaining, u.Credits.Total))
	}
	if u.Activity != nil {
		t := u.Activity.Totals
		parts = append(parts, fmt.Sprintf("Spend $%.2f | %.0f req", t.Spend, t.Requests))
		tokens := fmt.Sprintf("%s/%s (in/out)", formatTokens(t.PromptTokens), formatTokens(t.CompletionTokens))
		if t.ReasoningTokens > 0 {
			tokens += fmt.Sprintf(" +%s reason", formatTokens(t.ReasoningTokens))
		}
		parts = append(parts, tokens)
	}
	b.WriteString(dimStyle.Render("  " + strings.Join(parts, " | ")))
	b.WriteByte('\n')
	return b.String()
}

const (
	chartMaxHeight = 12
	chartMinDays   = 7
	chartTopModels = 6
	chartGutter    = 9 // "  %5.0f │" width
)

func (m Model) renderORDailyChart(u *openrouter.Usage) string {
	if u.DailyActivity == nil || len(u.DailyActivity.Days) == 0 {
		return ""
	}

	metric := m.orMetric
	days := u.DailyActivity.Days
	avail := m.width - 2 - chartGutter
	if avail < chartMinDays*2 {
		avail = chartMinDays * 2
	}
	maxDays := avail / 2
	if len(days) > maxDays {
		days = days[len(days)-maxDays:]
	}
	colWidth := 2
	if len(days) > 0 {
		if cw := avail / len(days); cw > colWidth {
			colWidth = cw
		}
		if colWidth > 4 {
			colWidth = 4
		}
	}

	// Find top models across all days by selected metric
	modelTotals := make(map[string]float64)
	for _, day := range days {
		for _, model := range day.Models {
			modelTotals[model.Model] += modelMetricValue(model, metric)
		}
	}
	topModels := topNModels(modelTotals, chartTopModels)
	topSet := make(map[string]bool, len(topModels))
	for _, tm := range topModels {
		topSet[tm] = true
	}

	// Pre-compute per-day totals and find max for scaling
	dayTotals := make([]float64, len(days))
	var maxTotal float64
	for i, day := range days {
		t := dayMetricTotal(day, metric)
		dayTotals[i] = t
		if t > maxTotal {
			maxTotal = t
		}
	}
	if maxTotal == 0 {
		return ""
	}

	// Pre-compute each column as an array of color indices (bottom to top)
	const (
		gutterPad  = "        " // 8 spaces, must match gutter visible width
		gutter     = "  %5s │"
		gutterBlnk = gutterPad + "│"
		emptyCell  = -1
	)
	height := chartMaxHeight
	filledGlyph := "▐" + strings.Repeat("█", colWidth-1)
	emptyGlyph := strings.Repeat(" ", colWidth)

	othersColorIdx := len(topModels) % len(modelBarColors)

	// columns[dayIdx][row] = color index, where row 0 = bottom
	columns := make([][]int, len(days))
	for di, day := range days {
		col := make([]int, height)
		for r := range col {
			col[r] = emptyCell
		}
		dayTotal := dayTotals[di]
		if dayTotal <= 0 {
			columns[di] = col
			continue
		}

		// Gather value per segment: top models in order, then others
		type segment struct {
			colorIdx int
			value    float64
		}
		var segments []segment
		var othersValue float64
		valueByModel := make(map[string]float64)
		for _, model := range day.Models {
			v := modelMetricValue(model, metric)
			if topSet[model.Model] {
				valueByModel[model.Model] += v
			} else {
				othersValue += v
			}
		}
		for i, name := range topModels {
			if s := valueByModel[name]; s > 0 {
				segments = append(segments, segment{i, s})
			}
		}
		if othersValue > 0 {
			segments = append(segments, segment{othersColorIdx, othersValue})
		}

		// Allocate cells using largest-remainder method
		totalCells := int(math.Round(dayTotal / maxTotal * float64(height)))
		if totalCells == 0 {
			totalCells = 1
		}
		if totalCells > height {
			totalCells = height
		}

		cellCounts := make([]int, len(segments))
		remainders := make([]float64, len(segments))
		allocated := 0
		for i, seg := range segments {
			exact := seg.value / dayTotal * float64(totalCells)
			cellCounts[i] = int(math.Floor(exact))
			remainders[i] = exact - float64(cellCounts[i])
			allocated += cellCounts[i]
		}
		// Distribute remaining cells to segments with largest remainders
		for allocated < totalCells {
			bestIdx := 0
			for i := 1; i < len(remainders); i++ {
				if remainders[i] > remainders[bestIdx] {
					bestIdx = i
				}
			}
			cellCounts[bestIdx]++
			remainders[bestIdx] = -1 // used up
			allocated++
		}

		// Fill column bottom-up
		cellIdx := 0
		for i, seg := range segments {
			for c := 0; c < cellCounts[i] && cellIdx < height; c++ {
				col[cellIdx] = seg.colorIdx
				cellIdx++
			}
		}
		columns[di] = col
	}

	var b strings.Builder
	b.WriteByte('\n')
	b.WriteString("  " + labelStyle.Render(metric.chartTitle()) + "\n")

	// Render rows top to bottom
	for row := height; row >= 1; row-- {
		switch row {
		case height:
			b.WriteString(dimStyle.Render(fmt.Sprintf(gutter, metricAxisLabel(maxTotal, metric))))
		case height / 2:
			b.WriteString(dimStyle.Render(fmt.Sprintf(gutter, metricAxisLabel(maxTotal/2, metric))))
		case 1:
			b.WriteString(dimStyle.Render(fmt.Sprintf(gutter, metricAxisLabel(maxTotal/float64(height), metric))))
		default:
			b.WriteString(dimStyle.Render(gutterBlnk))
		}

		cellRow := row - 1 // row 1 = index 0 (bottom)
		for _, col := range columns {
			if col[cellRow] != emptyCell {
				b.WriteString(modelBarFilledStyle(col[cellRow]).Render(filledGlyph))
			} else {
				b.WriteString(emptyGlyph)
			}
		}
		b.WriteByte('\n')
	}

	// X-axis (matches gutter width)
	b.WriteString(dimStyle.Render(gutterPad + "└" + strings.Repeat(strings.Repeat("─", colWidth), len(days))))
	b.WriteByte('\n')

	// Date labels — place at first, middle, last positions
	if len(days) > 0 {
		labelPositions := map[int]bool{0: true}
		if len(days) > 2 {
			labelPositions[len(days)/2] = true
		}
		if len(days) > 1 {
			labelPositions[len(days)-1] = true
		}

		var dateLine strings.Builder
		dateLine.WriteString(gutterPad + " ") // match gutter + └
		skip := 0
		for i, day := range days {
			if skip > 0 {
				skip--
				continue
			}
			if labelPositions[i] {
				lbl := shortDate(day.Date)
				dateLine.WriteString(dimStyle.Render(lbl))
				// How many column slots this label occupies beyond the first
				extraSlots := (len(lbl) - 1) / colWidth
				skip = extraSlots
			} else {
				dateLine.WriteString(strings.Repeat(" ", colWidth))
			}
		}
		b.WriteString(dateLine.String())
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

func shortDate(date string) string {
	// Handle "2026-03-10", "2026-03-10 00:00:00", "2026-03-10T00:00:00Z", etc.
	if len(date) >= 10 {
		return date[5:10] // "MM-DD"
	}
	return date
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
	metric := m.orMetric

	models := append([]openrouter.ModelUsage(nil), u.Activity.Models...)
	if metric != metricSpend {
		sort.Slice(models, func(i, j int) bool {
			return modelMetricValue(models[i], metric) > modelMetricValue(models[j], metric)
		})
	}
	if len(models) > maxModels {
		models = models[:maxModels]
	}

	var b strings.Builder
	b.WriteByte('\n')
	b.WriteString("  " + labelStyle.Render(metric.modelsTitle()) + "\n")

	maxVal := modelMetricValue(models[0], metric)
	barWidth := m.modelBarWidth()
	for i, model := range models {
		label := truncate(modelShortName(model.Model), 22)
		val := modelMetricValue(model, metric)
		fmt.Fprintf(&b, "  %s  %s  %s  %s\n",
			dimStyle.Render(fmt.Sprintf("%-22s", label)),
			dimStyle.Render(formatMetricColumn(val, metric)),
			renderModelBar(val, maxVal, barWidth, i),
			dimStyle.Render(formatMetricSecondary(model, metric)),
		)
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
	width = max(width, 1)
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
		display := k.Name
		if k.Label != "" {
			display = fmt.Sprintf("%s (%s)", k.Label, k.Name)
		}
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %-30s d:$%6.2f  w:$%6.2f  m:$%6.2f",
			truncate(display, 30), k.UsageDaily, k.UsageWeekly, k.UsageMonthly)))
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
