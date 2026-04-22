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

func (m Model) claudeElapsedPercent(resetAt time.Time, window time.Duration) float64 {
	if !m.claudeUIConfig.PaceTick {
		return -1
	}
	return elapsedPercent(resetAt, window)
}

func (m Model) claudeSection() string {
	box := sectionBorderStyle.Render(strings.TrimRight(m.claudeSectionBody(), "\n"))
	return injectBorderTitle(box, labelStyle.Render(" Claude ")) + "\n"
}

func (m Model) claudeSectionBody() string {
	var b strings.Builder

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
		if m.claudeUsage == nil {
			return b.String()
		}
	}

	u := m.claudeUsage
	bw := m.barWidth()
	render := renderBar
	if m.claudeUIConfig.Compact {
		render = renderCompactBar
	}

	b.WriteString(dimStyle.Render(fmt.Sprintf("  Plan: %s | Tier: %s", u.SubscriptionType, u.RateLimitTier)))
	b.WriteByte('\n')
	if !m.claudeUIConfig.Compact {
		b.WriteByte('\n')
	}

	if w := u.SessionLimit; w != nil {
		resetInfo := ""
		if !w.ResetAt.IsZero() {
			if m.claudeUIConfig.Compact {
				resetInfo = timeUntil(w.ResetAt)
			} else {
				resetInfo = fmt.Sprintf("resets %s (%s)", w.ResetAt.Local().Format("3:04 PM"), timeUntil(w.ResetAt))
			}
		}
		b.WriteString(render("5h Limit", w.Utilization*100, m.claudeElapsedPercent(w.ResetAt, claudeSessionWindow), bw, resetInfo))
		if !m.claudeUIConfig.Compact {
			b.WriteByte('\n')
		}
	}

	if w := u.WeeklyLimit; w != nil {
		resetInfo := ""
		if !w.ResetAt.IsZero() {
			if m.claudeUIConfig.Compact {
				resetInfo = timeUntil(w.ResetAt)
			} else {
				resetInfo = fmt.Sprintf("resets %s (%s)", w.ResetAt.Local().Format("Mon Jan 2 3:04 PM"), timeUntil(w.ResetAt))
			}
		}
		b.WriteString(render("Weekly  ", w.Utilization*100, m.claudeElapsedPercent(w.ResetAt, claudeWeeklyWindow), bw, resetInfo))
		if !m.claudeUIConfig.Compact {
			b.WriteByte('\n')
		}
	}

	b.WriteByte('\n')
	return b.String()
}

func fetchClaudeUsage(auth *claude.Auth) tea.Cmd {
	return func() tea.Msg {
		usage, err := claude.FetchUsage(auth)
		return claudeUsageMsg{usage: usage, err: err}
	}
}
