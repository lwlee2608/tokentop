package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lwlee2608/tokenburn/pkg/codex"
)

const (
	refreshInterval = 30 * time.Second
	retryDelay      = 5 * time.Second
	maxRetries      = 3
)

type tickMsg time.Time
type retryMsg struct{}

type usageMsg struct {
	usage *codex.Usage
	err   error
}

type Model struct {
	auth      *codex.Auth
	usage     *codex.Usage
	fetchErr  string
	retries   int
	lastFetch time.Time
	version   string
	width     int
}

func New(auth *codex.Auth, version string) Model {
	return Model{auth: auth, version: version}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchUsage(m.auth), tick())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tickMsg:
		m.retries = 0
		return m, tea.Batch(fetchUsage(m.auth), tick())
	case retryMsg:
		return m, fetchUsage(m.auth)
	case usageMsg:
		if msg.err != nil {
			m.fetchErr = msg.err.Error()
			if m.retries < maxRetries {
				m.retries++
				return m, tea.Tick(retryDelay, func(time.Time) tea.Msg { return retryMsg{} })
			}
		} else {
			m.usage = msg.usage
			m.fetchErr = ""
			m.retries = 0
			m.lastFetch = time.Now()
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

	// Header — center "tokenburn <version>"
	label := fmt.Sprintf(" tokenburn %s ", m.version)
	sideLen := (m.width - len(label) - 2) / 2 // -2 for ┌ and ┐
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

	if m.usage == nil && m.fetchErr == "" {
		b.WriteString(dimStyle.Render(" Loading..."))
		b.WriteByte('\n')
		return b.String()
	}

	if m.usage == nil && m.fetchErr != "" {
		b.WriteString(pctStyle(red).Render(fmt.Sprintf(" ⚠️  %s (retry %d/%d)", m.fetchErr, m.retries, maxRetries)))
		b.WriteByte('\n')
		b.WriteByte('\n')
		b.WriteString(m.footer())
		return b.String()
	}

	u := m.usage

	if m.fetchErr != "" {
		b.WriteString(pctStyle(yellow).Render(fmt.Sprintf(" ⚠️  %s (retry %d/%d)", m.fetchErr, m.retries, maxRetries)))
		b.WriteByte('\n')
	}
	b.WriteString(dimStyle.Render(fmt.Sprintf(" Plan: %s", u.PlanType)))
	b.WriteByte('\n')
	b.WriteByte('\n')

	bw := m.barWidth()

	// 5-hour window
	b.WriteString(renderBar(
		"5h Limit", u.PrimaryUsedPercent, bw,
		fmt.Sprintf("resets %s (%s)", u.PrimaryResetAt.Local().Format("3:04 PM"), timeUntil(u.PrimaryResetAt)),
	))
	b.WriteByte('\n')

	// Weekly window
	b.WriteString(renderBar(
		"Weekly", u.SecondaryUsedPercent, bw,
		fmt.Sprintf("resets %s (%s)", u.SecondaryResetAt.Local().Format("Mon Jan 2 3:04 PM"), timeUntil(u.SecondaryResetAt)),
	))

	if u.CreditsHasCredits {
		b.WriteByte('\n')
		b.WriteString(dimStyle.Render(fmt.Sprintf(" Credits: %s (unlimited: %v)", u.CreditsBalance, u.CreditsUnlimited)))
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	b.WriteString(m.footer())
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
	foot := dimStyle.Render(info) + "\n" + dimStyle.Render(strings.Repeat("─", m.width))
	return foot
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
	b.WriteString(" " + labelStyle.Render(label) + "\n")
	b.WriteString(fmt.Sprintf("  Used:%s  %s%s  %s\n",
		pctStyle(c).Render(fmt.Sprintf("%4.0f%%", used)),
		filled, empty,
		pctStyle(c).Render(fmt.Sprintf("%4.0f%% free", remaining)),
	))
	b.WriteString("  " + dimStyle.Render(resetInfo) + "\n")
	return b.String()
}

func fetchUsage(auth *codex.Auth) tea.Cmd {
	return func() tea.Msg {
		usage, err := codex.FetchUsage(auth)
		return usageMsg{usage: usage, err: err}
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
