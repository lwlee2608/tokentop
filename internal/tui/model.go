package tui

import (
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lwlee2608/tokentop/internal/config"
	"github.com/lwlee2608/tokentop/pkg/claude"
	"github.com/lwlee2608/tokentop/pkg/codex"
	"github.com/lwlee2608/tokentop/pkg/openrouter"
)

const (
	refreshInterval = 5 * time.Minute
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

type claudeUsageMsg struct {
	usage *claude.Usage
	err   error
}

type claudeRetryMsg struct{}

type Model struct {
	version        string
	width          int
	lastFetch      time.Time
	codexUIConfig  config.CodexUIConfig
	claudeUIConfig config.ClaudeUIConfig
	orUIConfig     config.OpenRouterUIConfig

	codexAuth    *codex.Auth
	codexUsage   *codex.Usage
	codexErr     string
	codexRetries int

	orAuth    *openrouter.Auth
	orUsage   *openrouter.Usage
	orErr     string
	orRetries int

	claudeAuth    *claude.Auth
	claudeUsage   *claude.Usage
	claudeErr     string
	claudeRetries int
}

func New(codexAuth *codex.Auth, orAuth *openrouter.Auth, claudeAuth *claude.Auth, codexUIConfig config.CodexUIConfig, claudeUIConfig config.ClaudeUIConfig, orUIConfig config.OpenRouterUIConfig, version string) Model {
	return Model{
		codexAuth:      codexAuth,
		orAuth:         orAuth,
		claudeAuth:     claudeAuth,
		codexUIConfig:  codexUIConfig,
		claudeUIConfig: claudeUIConfig,
		orUIConfig:     orUIConfig,
		version:        version,
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
	if m.claudeAuth != nil {
		cmds = append(cmds, fetchClaudeUsage(m.claudeAuth))
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
		m.claudeRetries = 0
		slog.Debug("starting usage refresh")
		var cmds []tea.Cmd
		if m.codexAuth != nil {
			cmds = append(cmds, fetchCodexUsage(m.codexAuth))
		}
		if m.orAuth != nil {
			cmds = append(cmds, fetchORUsage(m.orAuth))
		}
		if m.claudeAuth != nil {
			cmds = append(cmds, fetchClaudeUsage(m.claudeAuth))
		}
		cmds = append(cmds, tick())
		return m, tea.Batch(cmds...)

	case codexRetryMsg:
		return m, fetchCodexUsage(m.codexAuth)

	case orRetryMsg:
		return m, fetchORUsage(m.orAuth)

	case claudeRetryMsg:
		return m, fetchClaudeUsage(m.claudeAuth)

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
			if m.orUsage == nil {
				keyType := "standard"
				if msg.usage.Key.IsFreeTier {
					keyType = "free tier"
				} else if msg.usage.Key.IsManagementKey {
					keyType = "management"
				}
				slog.Info("openrouter key info", "label", msg.usage.Key.Label, "type", keyType)
			}
			m.orUsage = msg.usage
			m.orErr = ""
			m.orRetries = 0
			m.lastFetch = time.Now()
			slog.Debug("openrouter usage refresh succeeded")
		}

	case claudeUsageMsg:
		if msg.err != nil {
			m.claudeErr = msg.err.Error()
			slog.Warn("claude usage refresh failed", "error", msg.err, "retry", m.claudeRetries, "max_retries", maxRetries)
			if m.claudeRetries < maxRetries {
				m.claudeRetries++
				slog.Debug("scheduling claude usage retry", "retry", m.claudeRetries, "delay", retryDelay.String())
				return m, tea.Tick(retryDelay, func(time.Time) tea.Msg { return claudeRetryMsg{} })
			}
			slog.Error("claude usage refresh exhausted retries", "error", msg.err, "max_retries", maxRetries)
		} else {
			m.claudeUsage = msg.usage
			m.claudeErr = ""
			m.claudeRetries = 0
			m.lastFetch = time.Now()
			slog.Debug("claude usage refresh succeeded")
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

	if m.claudeAuth != nil {
		b.WriteString(m.claudeSection())
	}
	if m.codexAuth != nil {
		b.WriteString(m.codexSection())
	}
	if m.orAuth != nil {
		b.WriteString(m.orSection())
	}

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

func renderCompactBar(label string, usedPercent float64, barWidth int, resetInfo string) string {
	used := math.Min(usedPercent, 100)

	// Shrink bar to fit label, pct, and reset info on one line
	// Layout: "  {label}{pct}  {bar}  {resetInfo}"
	//          2 + labelWidth + 5(pct) + 2 + bar + 2 + len(resetInfo)
	overhead := barPadding + len(resetInfo)
	compactBarWidth := barWidth - overhead
	if compactBarWidth < 10 {
		compactBarWidth = 10
	}

	filledCount := int(math.Round(used / 100 * float64(compactBarWidth)))
	emptyCount := compactBarWidth - filledCount

	c := usageColor(used)

	filled := barFilledStyle(c).Render(strings.Repeat(" ", filledCount))
	empty := barEmptyStyle.Render(strings.Repeat(" ", emptyCount))

	var b strings.Builder
	reset := ""
	if resetInfo != "" {
		reset = "  " + dimStyle.Render(resetInfo)
	}
	b.WriteString(fmt.Sprintf("  %s%s  %s%s%s\n",
		labelStyle.Render(label),
		pctStyle(c).Render(fmt.Sprintf("%4.0f%%", used)),
		filled, empty,
		reset,
	))
	return b.String()
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
