package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/lwlee2608/tokentop/pkg/claude"

	tea "github.com/charmbracelet/bubbletea"
)

const claudeLoginHintLine = "  Hint: " + claude.LoginHint

func (m Model) claudeElapsedPercent(resetAt time.Time, window time.Duration) float64 {
	if !m.claudeUIConfig.PaceTick {
		return -1
	}
	return elapsedPercent(resetAt, window)
}

func (m Model) claudeSection() string {
	return sectionBox("Claude", m.claudeSectionBody(), m.width)
}

func (m Model) claudeSectionBody() string {
	var b strings.Builder

	if m.claudeAuth == nil && m.claudeEnabled {
		b.WriteString(pctStyle(red).Render("  ⚠️  credentials not found"))
		b.WriteByte('\n')
		b.WriteString(dimStyle.Render(claudeLoginHintLine))
		b.WriteByte('\n')
		return b.String()
	}

	if m.claudeUsage == nil && m.claudeErr == "" {
		b.WriteString(dimStyle.Render("  Loading..."))
		b.WriteByte('\n')
		return b.String()
	}

	if m.claudeErr != "" {
		c := yellow
		if m.claudeUsage == nil {
			c = red
		}
		b.WriteString(pctStyle(c).Render(fmt.Sprintf("  ⚠️  %s (retry %d/%d)", m.claudeErr, m.claudeRetries, maxRetries)))
		b.WriteByte('\n')
		if m.claudeAuthFailed {
			b.WriteString(dimStyle.Render(claudeLoginHintLine))
			b.WriteByte('\n')
		}
		if m.claudeUsage == nil {
			return b.String()
		}
	}

	u := m.claudeUsage
	bw := m.barWidth()

	b.WriteString(dimStyle.Render(fmt.Sprintf("  Plan: %s | Tier: %s", u.SubscriptionType, formatClaudeTier(u.RateLimitTier))))
	b.WriteByte('\n')

	labels := make([]string, len(u.Limits))
	labelWidth := 0
	for i, l := range u.Limits {
		labels[i] = claudeLimitLabel(l)
		labelWidth = max(labelWidth, len(labels[i]))
	}

	for i, l := range u.Limits {
		resetInfo := ""
		if !l.ResetAt.IsZero() {
			resetInfo = timeUntil(l.ResetAt)
		}
		pace := -1.0
		if d := claudeLimitWindow(l); d > 0 {
			pace = m.claudeElapsedPercent(l.ResetAt, d)
		}
		b.WriteString(renderCompactBar(fmt.Sprintf("%-*s", labelWidth, labels[i]), l.Percent, pace, bw, resetInfo))
	}

	b.WriteByte('\n')
	return b.String()
}

func claudeLimitLabel(l claude.Limit) string {
	label := l.Group
	switch l.Group {
	case "session":
		label = "5h Limit"
	case "weekly":
		label = "Weekly"
	}
	if l.Model != "" {
		label += " (" + l.Model + ")"
	}
	return label
}

func claudeLimitWindow(l claude.Limit) time.Duration {
	switch l.Group {
	case "session":
		return 5 * time.Hour
	case "weekly":
		return 7 * 24 * time.Hour
	}
	return 0
}

type claudeTier string

const (
	tierMax20x claudeTier = "default_claude_max_20x"
	tierMax5x  claudeTier = "default_claude_max_5x"
	tierAI     claudeTier = "default_claude_ai"
	tierZero   claudeTier = "default_claude_zero"
)

func formatClaudeTier(tier string) string {
	switch claudeTier(tier) {
	case tierMax20x:
		return "Max (20x)"
	case tierMax5x:
		return "Max (5x)"
	case tierAI:
		return "Pro"
	case tierZero:
		return "Zero"
	}
	return tier
}

func fetchClaudeUsage(auth *claude.Auth, gen uint64) tea.Cmd {
	return func() tea.Msg {
		usage, err := claude.FetchUsage(auth)
		return claudeUsageMsg{usage: usage, err: err, gen: gen}
	}
}
