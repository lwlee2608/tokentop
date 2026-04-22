package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/lwlee2608/tokentop/pkg/codex"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) codexElapsedPercent(w *codex.UsageWindow) float64 {
	if !m.codexUIConfig.PaceTick || w == nil || w.LimitWindowSeconds <= 0 {
		return -1
	}
	return elapsedPercent(w.ResetTime(), time.Duration(w.LimitWindowSeconds)*time.Second)
}

func (m Model) codexSection() string {
	return sectionBox("Codex", m.codexSectionBody())
}

func (m Model) codexSectionBody() string {
	var b strings.Builder

	if m.codexUsage == nil && m.codexErr == "" {
		b.WriteString(dimStyle.Render("  Loading..."))
		b.WriteByte('\n')
		return b.String()
	}

	if m.codexErr != "" {
		c := yellow
		if m.codexUsage == nil {
			c = red
		}
		b.WriteString(pctStyle(c).Render(fmt.Sprintf("  ⚠️  %s (retry %d/%d)", m.codexErr, m.codexRetries, maxRetries)))
		b.WriteByte('\n')
		if m.codexUsage == nil {
			return b.String()
		}
	}

	u := m.codexUsage
	bw := m.barWidth()
	render := renderBar
	if m.codexUIConfig.Compact {
		render = renderCompactBar
	}

	credits := ""
	if u.Credits.HasCredits {
		bal := "n/a"
		if u.Credits.Balance != nil {
			bal = *u.Credits.Balance
		}
		credits = fmt.Sprintf(" | Credits: %s (unlimited: %v)", bal, u.Credits.Unlimited)
	}
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Plan: %s%s", u.PlanType, credits)))
	b.WriteByte('\n')
	if !m.codexUIConfig.Compact {
		b.WriteByte('\n')
	}

	if w := u.RateLimit.PrimaryWindow; w != nil {
		resetInfo := fmt.Sprintf("resets %s (%s)", w.ResetTime().Local().Format("3:04 PM"), timeUntil(w.ResetTime()))
		if m.codexUIConfig.Compact {
			resetInfo = timeUntil(w.ResetTime())
		}
		b.WriteString(render("5h Limit", w.UsedPercent, m.codexElapsedPercent(w), bw, resetInfo))
		if !m.codexUIConfig.Compact {
			b.WriteByte('\n')
		}
	}
	if w := u.RateLimit.SecondaryWindow; w != nil {
		resetInfo := fmt.Sprintf("resets %s (%s)", w.ResetTime().Local().Format("Mon Jan 2 3:04 PM"), timeUntil(w.ResetTime()))
		if m.codexUIConfig.Compact {
			resetInfo = timeUntil(w.ResetTime())
		}
		b.WriteString(render("Weekly  ", w.UsedPercent, m.codexElapsedPercent(w), bw, resetInfo))
		if !m.codexUIConfig.Compact {
			b.WriteByte('\n')
		}
	}
	if m.codexUIConfig.CodeReview {
		if w := u.CodeReviewRateLimit.PrimaryWindow; w != nil {
			resetInfo := fmt.Sprintf("resets %s (%s)", w.ResetTime().Local().Format("Mon Jan 2 3:04 PM"), timeUntil(w.ResetTime()))
			if m.codexUIConfig.Compact {
				resetInfo = timeUntil(w.ResetTime())
			}
			b.WriteString(render("Code Review", w.UsedPercent, m.codexElapsedPercent(w), bw, resetInfo))
			if !m.codexUIConfig.Compact {
				b.WriteByte('\n')
			}
		}
	}

	b.WriteByte('\n')
	return b.String()
}

func fetchCodexUsage(auth *codex.Auth) tea.Cmd {
	return func() tea.Msg {
		usage, err := codex.FetchUsage(auth)
		return codexUsageMsg{usage: usage, err: err}
	}
}
