package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/lwlee2608/tokentop/pkg/codex"

	tea "github.com/charmbracelet/bubbletea"
)

const codexLoginHintLine = "  Hint: " + codex.LoginHint

func (m Model) codexElapsedPercent(w *codex.UsageWindow) float64 {
	if !m.codexUIConfig.PaceTick || w == nil || w.LimitWindowSeconds <= 0 {
		return -1
	}
	return elapsedPercent(w.ResetTime(), time.Duration(w.LimitWindowSeconds)*time.Second)
}

func codexWindowLabel(w *codex.UsageWindow) string {
	switch time.Duration(w.LimitWindowSeconds) * time.Second {
	case 5 * time.Hour:
		return "5h Limit"
	case 7 * 24 * time.Hour:
		return "Weekly  "
	case 30 * 24 * time.Hour:
		return "Monthly "
	default:
		return "Limit   "
	}
}

func (m Model) codexSection() string {
	return sectionBox("Codex", m.codexSectionBody(), m.width)
}

func (m Model) codexSectionBody() string {
	var b strings.Builder

	if m.codexAuth == nil && m.codexEnabled {
		b.WriteString(pctStyle(red).Render("  ⚠️  credentials not found"))
		b.WriteByte('\n')
		b.WriteString(dimStyle.Render(codexLoginHintLine))
		b.WriteByte('\n')
		return b.String()
	}

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
		if m.codexAuthFailed {
			b.WriteString(dimStyle.Render(codexLoginHintLine))
			b.WriteByte('\n')
		}
		if m.codexUsage == nil {
			return b.String()
		}
	}

	u := m.codexUsage
	bw := m.barWidth()

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

	if w := u.RateLimit.PrimaryWindow; w != nil {
		b.WriteString(m.renderCompactBar(codexWindowLabel(w), w.UsedPercent, m.codexElapsedPercent(w), bw, m.timeUntil(w.ResetTime())))
	}
	if w := u.RateLimit.SecondaryWindow; w != nil {
		b.WriteString(m.renderCompactBar(codexWindowLabel(w), w.UsedPercent, m.codexElapsedPercent(w), bw, m.timeUntil(w.ResetTime())))
	}
	if m.codexUIConfig.CodeReview && u.CodeReviewRateLimit != nil {
		if w := u.CodeReviewRateLimit.PrimaryWindow; w != nil {
			b.WriteString(m.renderCompactBar("Code Review", w.UsedPercent, m.codexElapsedPercent(w), bw, m.timeUntil(w.ResetTime())))
		}
	}

	b.WriteByte('\n')
	return b.String()
}

func fetchCodexUsage(auth *codex.Auth, gen uint64) tea.Cmd {
	return func() tea.Msg {
		usage, err := codex.FetchUsage(auth)
		return codexUsageMsg{usage: usage, err: err, gen: gen}
	}
}
