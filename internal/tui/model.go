package tui

import (
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lwlee2608/tokentop/pkg/codex"
	"github.com/lwlee2608/tokentop/pkg/openrouter"
)

const (
	refreshInterval = 30 * time.Second
	retryDelay      = 5 * time.Second
	maxRetries      = 3
)

type tickMsg time.Time

type codexUsageMsg struct {
	usage *codex.Usage
	err   error
}

type codexRetryMsg struct{}

type orUsageMsg struct {
	usage *openrouter.Usage
	err   error
}

type orRetryMsg struct{}

type Model struct {
	version   string
	width     int
	lastFetch time.Time

	codexAuth    *codex.Auth
	codexUsage   *codex.Usage
	codexErr     string
	codexRetries int

	orAuth    *openrouter.Auth
	orUsage   *openrouter.Usage
	orErr     string
	orRetries int
}

func New(codexAuth *codex.Auth, orAuth *openrouter.Auth, version string) Model {
	return Model{
		codexAuth: codexAuth,
		orAuth:    orAuth,
		version:   version,
	}
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tick()}
	if m.codexAuth != nil {
		cmds = append(cmds, fetchCodexUsage(m.codexAuth))
	}
	if m.orAuth != nil {
		cmds = append(cmds, fetchORUsage(m.orAuth))
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case tickMsg:
		m.codexRetries = 0
		m.orRetries = 0
		slog.Debug("starting usage refresh")
		var cmds []tea.Cmd
		if m.codexAuth != nil {
			cmds = append(cmds, fetchCodexUsage(m.codexAuth))
		}
		if m.orAuth != nil {
			cmds = append(cmds, fetchORUsage(m.orAuth))
		}
		cmds = append(cmds, tick())
		return m, tea.Batch(cmds...)

	case codexRetryMsg:
		return m, fetchCodexUsage(m.codexAuth)

	case orRetryMsg:
		return m, fetchORUsage(m.orAuth)

	case codexUsageMsg:
		if msg.err != nil {
			m.codexErr = msg.err.Error()
			slog.Warn("codex usage refresh failed", "error", msg.err, "retry", m.codexRetries, "max_retries", maxRetries)
			if m.codexRetries < maxRetries {
				m.codexRetries++
				slog.Debug("scheduling codex usage retry", "retry", m.codexRetries, "delay", retryDelay.String())
				return m, tea.Tick(retryDelay, func(time.Time) tea.Msg { return codexRetryMsg{} })
			}
			slog.Error("codex usage refresh exhausted retries", "error", msg.err, "max_retries", maxRetries)
		} else {
			m.codexUsage = msg.usage
			m.codexErr = ""
			m.codexRetries = 0
			m.lastFetch = time.Now()
			slog.Debug("codex usage refresh succeeded")
		}

	case orUsageMsg:
		if msg.err != nil {
			m.orErr = msg.err.Error()
			slog.Warn("openrouter usage refresh failed", "error", msg.err, "retry", m.orRetries, "max_retries", maxRetries)
			if m.orRetries < maxRetries {
				m.orRetries++
				slog.Debug("scheduling openrouter usage retry", "retry", m.orRetries, "delay", retryDelay.String())
				return m, tea.Tick(retryDelay, func(time.Time) tea.Msg { return orRetryMsg{} })
			}
			slog.Error("openrouter usage refresh exhausted retries", "error", msg.err, "max_retries", maxRetries)
		} else {
			m.orUsage = msg.usage
			m.orErr = ""
			m.orRetries = 0
			m.lastFetch = time.Now()
			slog.Debug("openrouter usage refresh succeeded")
		}
	}
	return m, nil
}

func (m Model) barWidth() int {
	w := m.width - barPadding
	if w < 10 {
		w = 10
	}
	return w
}

func (m Model) View() string {
	var b strings.Builder

	// Header
	label := fmt.Sprintf(" tokentop %s ", m.version)
	sideLen := (m.width - len(label) - 2) / 2
	if sideLen < 0 {
		sideLen = 0
	}
	rightLen := m.width - 2 - sideLen - len(label)
	if rightLen < 0 {
		rightLen = 0
	}
	title := "┌" + strings.Repeat("─", sideLen) + label + strings.Repeat("─", rightLen) + "┐"
	b.WriteString(headerStyle.Render(title))
	b.WriteByte('\n')

	if m.codexAuth != nil {
		b.WriteString(m.codexSection())
	}
	if m.orAuth != nil {
		b.WriteString(m.orSection())
	}

	// TODO: anthropic section (not yet implemented)

	b.WriteString(m.footer())
	return b.String()
}

func (m Model) codexSection() string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render(" Codex"))
	b.WriteByte('\n')

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

	b.WriteString(dimStyle.Render(fmt.Sprintf("  Plan: %s", u.PlanType)))
	b.WriteByte('\n')
	b.WriteByte('\n')

	if w := u.RateLimit.PrimaryWindow; w != nil {
		b.WriteString(renderBar("5h Limit", w.UsedPercent, bw,
			fmt.Sprintf("resets %s (%s)", w.ResetTime().Local().Format("3:04 PM"), timeUntil(w.ResetTime())),
		))
		b.WriteByte('\n')
	}
	if w := u.RateLimit.SecondaryWindow; w != nil {
		b.WriteString(renderBar("Weekly", w.UsedPercent, bw,
			fmt.Sprintf("resets %s (%s)", w.ResetTime().Local().Format("Mon Jan 2 3:04 PM"), timeUntil(w.ResetTime())),
		))
		b.WriteByte('\n')
	}
	if w := u.CodeReviewRateLimit.PrimaryWindow; w != nil {
		b.WriteString(renderBar("Code Review", w.UsedPercent, bw,
			fmt.Sprintf("resets %s (%s)", w.ResetTime().Local().Format("Mon Jan 2 3:04 PM"), timeUntil(w.ResetTime())),
		))
		b.WriteByte('\n')
	}

	if u.Credits.HasCredits {
		bal := "n/a"
		if u.Credits.Balance != nil {
			bal = *u.Credits.Balance
		}
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Credits: %s (unlimited: %v)", bal, u.Credits.Unlimited)))
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	return b.String()
}

func (m Model) footer() string {
	var ts string
	if !m.lastFetch.IsZero() {
		ts = m.lastFetch.Local().Format("3:04:05 PM")
	} else {
		ts = "..."
	}
	info := fmt.Sprintf(" refresh: %ds | updated: %s | q to quit", int(refreshInterval.Seconds()), ts)
	return dimStyle.Render(info) + "\n" + dimStyle.Render(strings.Repeat("─", m.width))
}

func renderBar(label string, usedPercent float64, barWidth int, resetInfo string) string {
	used := math.Min(usedPercent, 100)
	remaining := 100 - used

	filledCount := int(math.Round(used / 100 * float64(barWidth)))
	emptyCount := barWidth - filledCount

	c := usageColor(used)

	filled := barFilledStyle(c).Render(strings.Repeat(" ", filledCount))
	empty := barEmptyStyle.Render(strings.Repeat(" ", emptyCount))

	var b strings.Builder
	b.WriteString("  " + labelStyle.Render(label) + "\n")
	b.WriteString(fmt.Sprintf("   Used:%s  %s%s  %s\n",
		pctStyle(c).Render(fmt.Sprintf("%4.0f%%", used)),
		filled, empty,
		pctStyle(c).Render(fmt.Sprintf("%4.0f%% free", remaining)),
	))
	b.WriteString("   " + dimStyle.Render(resetInfo) + "\n")
	return b.String()
}

func fetchCodexUsage(auth *codex.Auth) tea.Cmd {
	return func() tea.Msg {
		usage, err := codex.FetchUsage(auth)
		return codexUsageMsg{usage: usage, err: err}
	}
}

func fetchORUsage(auth *openrouter.Auth) tea.Cmd {
	return func() tea.Msg {
		usage, err := openrouter.FetchUsage(auth)
		return orUsageMsg{usage: usage, err: err}
	}
}

func tick() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func timeUntil(t time.Time) string {
	d := time.Until(t)
	if d < 0 {
		return "expired"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("in %dh %dm", h, m)
	}
	return fmt.Sprintf("in %dm", m)
}
