package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/justin06lee/shaw/internal/run"
	"github.com/justin06lee/shaw/internal/stats"
)

const minWidth = 40

var (
	styleDim      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	styleCorrect  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleWrong    = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	styleActive   = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	styleInactive = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

// View renders the whole screen for the current state.
func (m Model) View() string {
	if m.width < minWidth {
		return "terminal too narrow — widen to at least 40 columns"
	}
	switch m.state {
	case StateResult:
		return m.resultView()
	default:
		return m.typingView()
	}
}

// typingView renders the config bar, three-line text area, and footer.
func (m Model) typingView() string {
	bar := m.configBar()
	text := m.textArea()
	footer := m.footer()
	return strings.Join([]string{"", bar, "", text, "", footer}, "\n")
}

// configBar renders the mode and target segmented controls.
func (m Model) configBar() string {
	dim := m.state == StateActive
	modes := []string{"time", "words", "zen"}
	var parts []string
	for i, name := range modes {
		parts = append(parts, segment(name, i == m.modeIdx, dim, m.barFocus == 0))
	}
	modeCtl := strings.Join(parts, " ")

	var tparts []string
	for i, tv := range targetOptions[m.Mode()] {
		if m.Mode() == run.ModeZen {
			continue
		}
		tparts = append(tparts, segment(fmt.Sprintf("%d", tv),
			i == m.targetIdx, dim, m.barFocus == 1))
	}
	targetCtl := strings.Join(tparts, " ")
	return "  " + modeCtl + "    " + targetCtl
}

// segment styles one config-bar option.
func segment(label string, selected, dim, focused bool) string {
	st := styleInactive
	if selected {
		st = styleActive
	}
	if dim {
		st = styleDim
	}
	if focused && selected && !dim {
		label = "[" + label + "]"
	}
	return st.Render(label)
}

// textArea renders the 3-line scrolling viewport of the target text.
func (m Model) textArea() string {
	text := m.run.Text()
	width := m.width - 4
	lines := WrapLines(text, width)
	cursorLine := LineOfCursor(lines, m.run.Cursor())
	start, count := Viewport(lines, cursorLine)
	states := m.run.States()
	cursor := m.run.Cursor()

	var out []string
	for i := start; i < start+count; i++ {
		ln := lines[i]
		var b strings.Builder
		for j := ln.Start; j < ln.End; j++ {
			ch := string(text[j])
			switch {
			case j == cursor:
				b.WriteString(styleActive.Render(ch))
			case states[j] == run.Correct:
				b.WriteString(styleCorrect.Render(ch))
			case states[j] == run.Incorrect:
				b.WriteString(styleWrong.Render(ch))
			default:
				b.WriteString(styleDim.Render(ch))
			}
		}
		out = append(out, "  "+b.String())
	}
	return strings.Join(out, "\n")
}

// footer renders the live progress indicator and key hints.
func (m Model) footer() string {
	var status string
	switch m.Mode() {
	case run.ModeTime:
		remaining := m.Target() - int(m.run.Elapsed().Seconds())
		if remaining < 0 {
			remaining = 0
		}
		status = fmt.Sprintf("%ds", remaining)
	case run.ModeWords:
		done := 0
		text := m.run.Text()
		for _, ch := range text[:m.run.Cursor()] {
			if ch == ' ' {
				done++
			}
		}
		status = fmt.Sprintf("%d/%d words", done, m.Target())
	default: // zen
		status = fmt.Sprintf("%ds", int(m.run.Elapsed().Seconds()))
	}
	return styleDim.Render("  " + status + "   ·   esc to restart")
}

// resultView renders metrics, the WPM chart, and the error breakdown.
func (m Model) resultView() string {
	r := m.result
	var b strings.Builder
	b.WriteString(styleActive.Render(
		fmt.Sprintf("\n  %.0f wpm   %.0f%% acc\n", r.NetWPM, r.Accuracy*100)))
	b.WriteString(styleDim.Render(fmt.Sprintf(
		"  raw %.0f   consistency %.0f%%\n\n", r.RawWPM, r.Consistency)))
	chart := stats.RenderChart(r.Samples, m.width-4, 8)
	for _, ln := range strings.Split(chart, "\n") {
		b.WriteString("  " + styleCorrect.Render(ln) + "\n")
	}
	b.WriteString("\n")
	if len(r.MissedChars) > 0 {
		var parts []string
		for _, mc := range r.MissedChars {
			parts = append(parts, fmt.Sprintf("%q×%d", mc.Char, mc.Count))
		}
		b.WriteString(styleDim.Render("  missed: " + strings.Join(parts, "  ") + "\n"))
	}
	b.WriteString(styleDim.Render("  saved to history   ·   enter for a new run\n"))
	return b.String()
}
