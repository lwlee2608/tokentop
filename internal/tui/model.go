package tui

import (
	"errors"
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
type countdownMsg time.Time

type codexUsageMsg struct {
	usage *codex.Usage
	err   error
	gen   uint64
}

type codexRetryMsg struct{ gen uint64 }

type orUsageMsg struct {
	usage *openrouter.Usage
	err   error
	gen   uint64
}

type orRetryMsg struct{ gen uint64 }

type claudeUsageMsg struct {
	usage *claude.Usage
	err   error
	gen   uint64
}

type claudeRetryMsg struct{ gen uint64 }

type Model struct {
	version        string
	width          int
	lastFetch      time.Time
	nextRefresh    time.Time
	gen            uint64
	codexUIConfig  config.CodexUIConfig
	claudeUIConfig config.ClaudeUIConfig
	orUIConfig     config.OpenRouterUIConfig

	codexAuth       *codex.Auth
	codexUsage      *codex.Usage
	codexErr        string
	codexRetries    int
	codexEnabled    bool
	codexAuthFailed bool

	orAuth       *openrouter.Auth
	orUsage      *openrouter.Usage
	orErr        string
	orRetries    int
	orMetric     orMetric
	orEnabled    bool
	orAuthFailed bool

	claudeAuth       *claude.Auth
	claudeUsage      *claude.Usage
	claudeErr        string
	claudeRetries    int
	claudeEnabled    bool
	claudeAuthFailed bool
}

type CodexProvider struct {
	Auth    *codex.Auth
	Enabled bool
	UI      config.CodexUIConfig
}

type OpenRouterProvider struct {
	Auth    *openrouter.Auth
	Enabled bool
	UI      config.OpenRouterUIConfig
}

type ClaudeProvider struct {
	Auth    *claude.Auth
	Enabled bool
	UI      config.ClaudeUIConfig
}

func New(cx CodexProvider, or OpenRouterProvider, cl ClaudeProvider, version string) Model {
	return Model{
		codexAuth:      cx.Auth,
		codexEnabled:   cx.Enabled,
		orAuth:         or.Auth,
		orEnabled:      or.Enabled,
		claudeAuth:     cl.Auth,
		claudeEnabled:  cl.Enabled,
		codexUIConfig:  cx.UI,
		claudeUIConfig: cl.UI,
		orUIConfig:     or.UI,
		orMetric:       parseMetric(or.UI.Metric),
		version:        version,
		nextRefresh:    time.Now().Add(refreshInterval),
	}
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tick(), countdown()}
	if m.codexAuth != nil {
		cmds = append(cmds, fetchCodexUsage(m.codexAuth, m.gen))
	}
	if m.orAuth != nil {
		cmds = append(cmds, fetchORUsage(m.orAuth, m.gen))
	}
	if m.claudeAuth != nil {
		cmds = append(cmds, fetchClaudeUsage(m.claudeAuth, m.gen))
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			return m.refresh()
		case "m":
			m.orMetric = m.orMetric.next()
			return m, nil
		}

	case countdownMsg:
		return m, countdown()

	case tickMsg:
		return m.refresh()

	case codexRetryMsg:
		if msg.gen != m.gen {
			return m, nil
		}
		return m, fetchCodexUsage(m.codexAuth, m.gen)

	case orRetryMsg:
		if msg.gen != m.gen {
			return m, nil
		}
		return m, fetchORUsage(m.orAuth, m.gen)

	case claudeRetryMsg:
		if msg.gen != m.gen {
			return m, nil
		}
		return m, fetchClaudeUsage(m.claudeAuth, m.gen)

	case codexUsageMsg:
		if msg.gen != m.gen {
			return m, nil
		}
		if msg.err != nil {
			m.codexErr = msg.err.Error()
			m.codexAuthFailed = errors.Is(msg.err, codex.ErrUnauthorized)
			if cmd := m.scheduleRetry("codex", &m.codexRetries, msg.err, func(g uint64) tea.Msg { return codexRetryMsg{gen: g} }); cmd != nil {
				return m, cmd
			}
		} else {
			m.codexUsage = msg.usage
			m.codexErr = ""
			m.codexAuthFailed = false
			m.codexRetries = 0
			m.lastFetch = time.Now()
			slog.Debug("codex usage refresh succeeded")
		}

	case orUsageMsg:
		if msg.gen != m.gen {
			return m, nil
		}
		if msg.err != nil {
			m.orErr = msg.err.Error()
			m.orAuthFailed = errors.Is(msg.err, openrouter.ErrUnauthorized)
			if cmd := m.scheduleRetry("openrouter", &m.orRetries, msg.err, func(g uint64) tea.Msg { return orRetryMsg{gen: g} }); cmd != nil {
				return m, cmd
			}
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
			m.orAuthFailed = false
			m.orRetries = 0
			m.lastFetch = time.Now()
			slog.Debug("openrouter usage refresh succeeded")
		}

	case claudeUsageMsg:
		if msg.gen != m.gen {
			return m, nil
		}
		if msg.err != nil {
			m.claudeErr = msg.err.Error()
			m.claudeAuthFailed = errors.Is(msg.err, claude.ErrUnauthorized)
			if cmd := m.scheduleRetry("claude", &m.claudeRetries, msg.err, func(g uint64) tea.Msg { return claudeRetryMsg{gen: g} }); cmd != nil {
				return m, cmd
			}
		} else {
			m.claudeUsage = msg.usage
			m.claudeErr = ""
			m.claudeAuthFailed = false
			m.claudeRetries = 0
			m.lastFetch = time.Now()
			slog.Debug("claude usage refresh succeeded")
		}
	}
	return m, nil
}

// scheduleRetry handles the common retry-with-backoff bookkeeping. Returns a
// tea.Cmd to fire the retry, or nil when retries are exhausted.
func (m Model) scheduleRetry(provider string, retries *int, err error, mkMsg func(gen uint64) tea.Msg) tea.Cmd {
	slog.Warn(provider+" usage refresh failed", "error", err, "retry", *retries, "max_retries", maxRetries)
	if *retries >= maxRetries {
		slog.Error(provider+" usage refresh exhausted retries", "error", err, "max_retries", maxRetries)
		return nil
	}
	*retries++
	slog.Debug("scheduling "+provider+" usage retry", "retry", *retries, "delay", retryDelay.String())
	gen := m.gen
	return tea.Tick(retryDelay, func(time.Time) tea.Msg { return mkMsg(gen) })
}

func (m Model) refresh() (tea.Model, tea.Cmd) {
	m.nextRefresh = time.Now().Add(refreshInterval)
	m.gen++
	m.codexRetries = 0
	m.orRetries = 0
	m.claudeRetries = 0
	slog.Debug("starting usage refresh", "gen", m.gen)
	var cmds []tea.Cmd
	if m.codexAuth != nil {
		cmds = append(cmds, fetchCodexUsage(m.codexAuth, m.gen))
	}
	if m.orAuth != nil {
		cmds = append(cmds, fetchORUsage(m.orAuth, m.gen))
	}
	if m.claudeAuth != nil {
		cmds = append(cmds, fetchClaudeUsage(m.claudeAuth, m.gen))
	}
	cmds = append(cmds, tick())
	return m, tea.Batch(cmds...)
}

func (m Model) barWidth() int {
	w := m.width - barPadding
	w = max(w, 10)
	return w
}

func (m Model) View() string {
	var b strings.Builder

	if m.claudeAuth != nil || m.claudeEnabled {
		b.WriteString(m.claudeSection())
	}
	if m.codexAuth != nil || m.codexEnabled {
		b.WriteString(m.codexSection())
	}
	if m.orAuth != nil || m.orEnabled {
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
	remaining := max(time.Until(m.nextRefresh), 0)
	var orHint string
	if m.orAuth != nil && (m.orUIConfig.DailySpend || m.orUIConfig.TopModels) {
		orHint = fmt.Sprintf(" | metric: %s (m)", m.orMetric)
	}
	info := fmt.Sprintf(" refresh: %ds | updated: %s | r to refresh%s | q to quit", int(remaining.Seconds()), ts, orHint)
	return dimStyle.Render(info)
}

func elapsedCells(elapsedPercent float64, barWidth int) int {
	if elapsedPercent <= 0 {
		return 0
	}
	n := min(int(math.Round(elapsedPercent/100*float64(barWidth))), barWidth)
	return n
}

type barCellState uint8

const (
	barCellEmpty barCellState = iota
	barCellSlack
	barCellFilled
	barCellOverPace
)

func buildBarCells(usedPercent, elapsedPercent float64, barWidth int) []barCellState {
	if barWidth <= 0 {
		return nil
	}

	used := math.Min(usedPercent, 100)
	filledCount := int(math.Round(used / 100 * float64(barWidth)))

	showElapsed := elapsedPercent >= 0
	eCount := 0
	if showElapsed {
		eCount = elapsedCells(elapsedPercent, barWidth)
	}

	cells := make([]barCellState, barWidth)
	for i := range barWidth {
		switch {
		case i < filledCount && (!showElapsed || i < eCount):
			cells[i] = barCellFilled
		case i < filledCount:
			cells[i] = barCellOverPace
		case showElapsed && i < eCount:
			cells[i] = barCellSlack
		default:
			cells[i] = barCellEmpty
		}
	}
	return cells
}

func renderBar(label string, usedPercent, elapsedPercent float64, barWidth int, resetInfo string) string {
	used := math.Min(usedPercent, 100)
	remaining := 100 - used

	c := usageColor(used)
	cells := buildBarCells(usedPercent, elapsedPercent, barWidth)

	var bar strings.Builder
	for _, cell := range cells {
		switch cell {
		case barCellFilled:
			bar.WriteString(barFilledStyle(c).Render(" "))
		case barCellOverPace:
			bar.WriteString(barOverPaceStyle(c).Render(" "))
		case barCellSlack:
			bar.WriteString(barSlackStyle.Render(" "))
		case barCellEmpty:
			bar.WriteString(barEmptyStyle.Render(" "))
		}
	}

	var b strings.Builder
	b.WriteString("  " + labelStyle.Render(label) + "\n")
	fmt.Fprintf(&b, "   Used:%s  %s  %s\n",
		pctStyle(c).Render(fmt.Sprintf("%4.0f%%", used)),
		bar.String(),
		pctStyle(c).Render(fmt.Sprintf("%4.0f%% free", remaining)),
	)
	b.WriteString("   " + dimStyle.Render(resetInfo) + "\n")
	return b.String()
}

const compactResetWidth = 8 // fixed width for reset info, e.g. "168h 59m"

func renderCompactBar(label string, usedPercent, elapsedPercent float64, barWidth int, resetInfo string) string {
	used := math.Min(usedPercent, 100)

	// Shrink bar to fit label, pct, and fixed-width reset info on one line
	overhead := barPadding + compactResetWidth - 2
	compactBarWidth := barWidth - overhead
	compactBarWidth = max(compactBarWidth, 10)

	c := usageColor(used)
	cells := buildBarCells(usedPercent, elapsedPercent, compactBarWidth)

	var bar strings.Builder
	for _, cell := range cells {
		switch cell {
		case barCellFilled:
			bar.WriteString(compactBarFilledStyle(c).Render("▄"))
		case barCellOverPace:
			bar.WriteString(compactBarOverPaceStyle(c).Render("▄"))
		case barCellSlack:
			bar.WriteString(compactBarSlackStyle.Render("▄"))
		case barCellEmpty:
			bar.WriteString(compactBarEmptyStyle.Render("▄"))
		}
	}

	reset := fmt.Sprintf("%*s", compactResetWidth, resetInfo)

	var b strings.Builder
	fmt.Fprintf(&b, "  %s%s  %s  %s\n",
		labelStyle.Render(label),
		pctStyle(c).Render(fmt.Sprintf("%4.0f%%", used)),
		bar.String(),
		dimStyle.Render(reset),
	)
	return b.String()
}

func fetchORUsage(auth *openrouter.Auth, gen uint64) tea.Cmd {
	return func() tea.Msg {
		usage, err := openrouter.FetchUsage(auth)
		return orUsageMsg{usage: usage, err: err, gen: gen}
	}
}

func tick() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func countdown() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return countdownMsg(t)
	})
}

func elapsedPercent(resetAt time.Time, windowDuration time.Duration) float64 {
	if resetAt.IsZero() || windowDuration <= 0 {
		return -1
	}
	remaining := time.Until(resetAt)
	if remaining <= 0 {
		return 100
	}
	if remaining >= windowDuration {
		return 0
	}
	return float64(windowDuration-remaining) / float64(windowDuration) * 100
}

func timeUntil(t time.Time) string {
	d := time.Until(t)
	if d < 0 {
		return "expired"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
