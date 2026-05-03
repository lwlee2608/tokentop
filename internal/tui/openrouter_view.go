package tui

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
)

func redactAPIKey(key string) string {
	if len(key) <= 8 {
		return "…"
	}
	return key[:4] + "…" + key[len(key)-4:]
}

func (m Model) renderORKeyHeader() string {
	var label, redacted string
	if m.orUsage != nil {
		label = m.orUsage.Key.Label
	}
	if m.orAuth != nil && m.orAuth.APIKey != "" {
		redacted = redactAPIKey(m.orAuth.APIKey)
	}
	// OpenRouter returns the redacted key as Label for non-management keys; dedupe.
	if strings.HasPrefix(label, "sk-or-") {
		label = ""
	}
	parts := make([]string, 0, 2)
	if label != "" {
		parts = append(parts, label)
	}
	if redacted != "" {
		parts = append(parts, redacted)
	}
	if len(parts) == 0 {
		return ""
	}
	suffix := " (standard key)"
	if m.orUsage != nil && m.orUsage.Key.IsManagementKey {
		suffix = " (mgnt key)"
	}
	return dimStyle.Render("  Key: "+strings.Join(parts, " · ")+suffix) + "\n"
}

func parseLimitResetPeriod(s string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "daily":
		return "Daily", true
	case "weekly":
		return "Weekly", true
	case "monthly":
		return "Monthly", true
	}
	return "", false
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

	if header := m.renderORKeyHeader(); header != "" {
		b.WriteString(header)
	}

	if m.orUsage.Key.IsManagementKey {
		b.WriteString(m.renderORManagementBody(m.orUsage))
	} else {
		b.WriteString(m.renderORStandardBody(m.orUsage))
	}

	b.WriteByte('\n')
	return b.String()
}

func truncate(s string, max int) string {
	if runewidth.StringWidth(s) <= max {
		return s
	}
	return runewidth.Truncate(s, max, "…")
}
