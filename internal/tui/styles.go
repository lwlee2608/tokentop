package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

const barPadding = 12

var (
	green     = lipgloss.Color("2")
	cyan      = lipgloss.Color("6")
	blue      = lipgloss.Color("12")
	orange    = lipgloss.Color("208")
	pink      = lipgloss.Color("205")
	yellow    = lipgloss.Color("3")
	red       = lipgloss.Color("1")
	brightRed = lipgloss.Color("9")
	white     = lipgloss.Color("15")
	gray      = lipgloss.Color("237")
	slack     = lipgloss.Color("244")

	dimStyle   = lipgloss.NewStyle().Faint(true)
	labelStyle = lipgloss.NewStyle().Bold(true).Foreground(white)

	sectionBorderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(slack)
)

func barFilledStyle(c lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Background(c).Foreground(white)
}

var barEmptyStyle = lipgloss.NewStyle().Background(gray)

func compactBarFilledStyle(c lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(c)
}

var compactBarEmptyStyle = lipgloss.NewStyle().Foreground(gray)

var compactBarSlackStyle = lipgloss.NewStyle().Foreground(slack)

var barSlackStyle = lipgloss.NewStyle().Background(slack)

func compactBarOverPaceStyle(usage lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(overPaceColor(usage))
}

func barOverPaceStyle(usage lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Background(overPaceColor(usage))
}

// overPaceColor returns a color one step more alarming than the usage color.
func overPaceColor(c lipgloss.Color) lipgloss.Color {
	switch c {
	case green:
		return yellow
	case yellow:
		return red
	case red:
		return brightRed
	default:
		return red
	}
}

var modelBarEmptyStyle = lipgloss.NewStyle().Foreground(gray)

var modelBarColors = []lipgloss.Color{cyan, blue, green, yellow, orange, pink, red}

func modelBarFilledStyle(i int) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(modelBarColors[i%len(modelBarColors)])
}

func pctStyle(c lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(c)
}

// sectionBox wraps body in the rounded section border with title spliced into the top edge.
// Body lines wider than width-2 are clipped so the right border stays visible.
func sectionBox(title, body string, width int) string {
	body = clipLines(body, width-2)
	box := sectionBorderStyle.Render(strings.TrimRight(body, "\n"))
	return injectBorderTitle(box, labelStyle.Render(" "+title+" ")) + "\n"
}

// clipLines truncates each line of s to at most maxW visible cells, preserving ANSI
// escape sequences (they have zero visible width) and closing any open style at the cut.
func clipLines(s string, maxW int) string {
	if maxW <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if lipgloss.Width(line) > maxW {
			lines[i] = clipLine(line, maxW)
		}
	}
	return strings.Join(lines, "\n")
}

func clipLine(line string, maxW int) string {
	var b strings.Builder
	visible := 0
	for i := 0; i < len(line); {
		if line[i] == 0x1b && i+1 < len(line) && line[i+1] == '[' {
			j := i + 2
			for j < len(line) && !(line[j] >= '@' && line[j] <= '~') {
				j++
			}
			if j < len(line) {
				j++
			}
			b.WriteString(line[i:j])
			i = j
			continue
		}
		if visible >= maxW {
			break
		}
		r, size := utf8.DecodeRuneInString(line[i:])
		_ = r
		b.WriteString(line[i : i+size])
		i += size
		visible++
	}
	b.WriteString("\x1b[0m")
	return b.String()
}

// injectBorderTitle splices a title onto the top edge of a rendered rounded-border box,
// replacing a portion of the top ─ run with `title`. Expects the top line to be a
// single styled run: "<ansi>╭───╮<reset>".
func injectBorderTitle(rendered, title string) string {
	nl := strings.Index(rendered, "\n")
	if nl == -1 {
		return rendered
	}
	top, rest := rendered[:nl], rendered[nl:]

	left := strings.Index(top, "╭")
	right := strings.LastIndex(top, "╮")
	if left == -1 || right == -1 || left >= right {
		return rendered
	}

	ansiOpen := top[:left]
	ansiClose := top[right+len("╮"):]
	dashes := utf8.RuneCountInString(top[left+len("╭") : right])

	const lead = 2
	titleW := lipgloss.Width(title)
	if dashes < lead+titleW+1 {
		return rendered
	}
	trail := dashes - lead - titleW

	var b strings.Builder
	b.WriteString(ansiOpen)
	b.WriteString("╭")
	b.WriteString(strings.Repeat("─", lead))
	b.WriteString(ansiClose)
	b.WriteString(title)
	b.WriteString(ansiOpen)
	b.WriteString(strings.Repeat("─", trail))
	b.WriteString("╮")
	b.WriteString(ansiClose)
	b.WriteString(rest)
	return b.String()
}

func usageColor(pct float64) lipgloss.Color {
	switch {
	case pct >= 90:
		return red
	case pct >= 70:
		return yellow
	default:
		return green
	}
}
