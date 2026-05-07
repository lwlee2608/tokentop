package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/lwlee2608/tokentop/pkg/claude"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	claudeSessionWindow = 5 * time.Hour
	claudeWeeklyWindow  = 7 * 24 * time.Hour
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

	b.WriteString(dimStyle.Render(fmt.Sprintf("  Plan: %s | Tier: %s", u.SubscriptionType, u.RateLimitTier)))
	b.WriteByte('\n')

	if w := u.SessionLimit; w != nil {
		resetInfo := ""
		if !w.ResetAt.IsZero() {
			resetInfo = timeUntil(w.ResetAt)
		}
		b.WriteString(renderCompactBar("5h Limit", w.Utilization*100, m.claudeElapsedPercent(w.ResetAt, claudeSessionWindow), bw, resetInfo))
	}

	if w := u.WeeklyLimit; w != nil {
		resetInfo := ""
		if !w.ResetAt.IsZero() {
			resetInfo = timeUntil(w.ResetAt)
		}
		b.WriteString(renderCompactBar("Weekly  ", w.Utilization*100, m.claudeElapsedPercent(w.ResetAt, claudeWeeklyWindow), bw, resetInfo))
	}

	b.WriteByte('\n')
	return b.String()
}

func fetchClaudeUsage(auth *claude.Auth, gen uint64) tea.Cmd {
	return func() tea.Msg {
		usage, err := claude.FetchUsage(auth)
		return claudeUsageMsg{usage: usage, err: err, gen: gen}
	}
}
