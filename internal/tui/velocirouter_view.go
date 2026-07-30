package tui

import (
	"fmt"
	"strings"

	"github.com/lwlee2608/tokentop/pkg/velocirouter"

	tea "github.com/charmbracelet/bubbletea"
)

const vrLoginHintLine = "  Hint: " + velocirouter.LoginHint

func (m Model) vrElapsedPercent(w velocirouter.Window) float64 {
	if !m.vrUIConfig.PaceTick || w.Duration <= 0 {
		return -1
	}
	return elapsedPercent(w.ResetAt, w.Duration)
}

func vrWindowLabel(name string) string {
	switch strings.ToLower(name) {
	case "5h":
		return "5h Limit"
	case "daily":
		return "Daily"
	case "weekly":
		return "Weekly"
	case "monthly":
		return "Monthly"
	}
	return name
}

func (m Model) renderVRKeyHeader() string {
	parts := make([]string, 0, 2)
	if m.vrUsage != nil {
		if m.vrUsage.Key.Name != "" {
			parts = append(parts, m.vrUsage.Key.Name)
		}
		if m.vrUsage.Key.Prefix != "" {
			parts = append(parts, m.vrUsage.Key.Prefix)
		}
	}
	if len(parts) == 0 && m.vrAuth != nil && m.vrAuth.APIKey != "" {
		parts = append(parts, redactAPIKey(m.vrAuth.APIKey))
	}
	if len(parts) == 0 {
		return ""
	}
	return dimStyle.Render("  Key: "+strings.Join(parts, " · ")) + "\n"
}

func (m Model) vrSection() string {
	return sectionBox("VelociRouter", m.vrSectionBody(), m.width)
}

func (m Model) vrSectionBody() string {
	var b strings.Builder

	if m.vrAuth == nil && m.vrEnabled {
		b.WriteString(pctStyle(red).Render("  ⚠️  credentials not found"))
		b.WriteByte('\n')
		b.WriteString(dimStyle.Render(vrLoginHintLine))
		b.WriteByte('\n')
		return b.String()
	}

	if m.vrUsage == nil && m.vrErr == "" {
		b.WriteString(dimStyle.Render("  Loading..."))
		b.WriteByte('\n')
		return b.String()
	}

	if m.vrErr != "" {
		c := yellow
		if m.vrUsage == nil {
			c = red
		}
		b.WriteString(pctStyle(c).Render(fmt.Sprintf("  ⚠️  %s (retry %d/%d)", m.vrErr, m.vrRetries, maxRetries)))
		b.WriteByte('\n')
		if m.vrAuthFailed {
			b.WriteString(dimStyle.Render(vrLoginHintLine))
			b.WriteByte('\n')
		}
		if m.vrUsage == nil {
			return b.String()
		}
	}

	u := m.vrUsage
	bw := m.barWidth()

	if header := m.renderVRKeyHeader(); header != "" {
		b.WriteString(header)
	}

	labels := make([]string, len(u.Windows))
	labelWidth := len("Lifetime")
	for i, w := range u.Windows {
		labels[i] = vrWindowLabel(w.Name)
		labelWidth = max(labelWidth, len(labels[i]))
	}

	for i, w := range u.Windows {
		label := fmt.Sprintf("%-*s", labelWidth, labels[i])
		if !w.HasLimit() {
			b.WriteString("  " + labelStyle.Render(label) +
				dimStyle.Render(fmt.Sprintf("%5s  $%.2f spent (no limit)", "—", w.SpentUSD)) + "\n")
			continue
		}
		resetInfo := ""
		if !w.ResetAt.IsZero() {
			resetInfo = m.timeUntil(w.ResetAt)
		}
		b.WriteString(m.renderCompactBar(label, w.UsedPercent(), m.vrElapsedPercent(w), bw, resetInfo))
	}

	if m.vrUIConfig.Lifetime {
		l := u.Lifetime
		b.WriteString("  " + labelStyle.Render(fmt.Sprintf("%-*s", labelWidth, "Lifetime")) +
			dimStyle.Render(fmt.Sprintf("  %.0f reqs | %s tok | $%.2f", l.Requests, formatTokens(l.TotalTokens), l.CostUSD)) + "\n")
	}

	b.WriteByte('\n')
	return b.String()
}

func fetchVRUsage(auth *velocirouter.Auth, gen uint64) tea.Cmd {
	return func() tea.Msg {
		usage, err := velocirouter.FetchUsage(auth)
		return vrUsageMsg{usage: usage, err: err, gen: gen}
	}
}
