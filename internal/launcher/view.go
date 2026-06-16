package launcher

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// columns computes how many cards fit across the content area for a terminal width.
func columns(totalWidth int) int {
	avail := totalWidth - sidebarW - 2
	if avail < cardW {
		return 1
	}
	return (avail + gap) / (cardW + gap)
}

var navLabels = []string{"Library", "Store", "Kalama"}

func (m Model) View() string {
	height := m.height
	if height < 1 {
		height = 24
	}
	sidebar := m.renderSidebar(height)
	main := m.renderMain(height)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
}

func (m Model) renderSidebar(height int) string {
	var b strings.Builder
	b.WriteString(wordmarkStyle.Render("SHAW"))
	b.WriteString("\n\n")
	for i, label := range navLabels {
		marker := "  "
		st := navItemStyle
		if section(i) == m.section {
			st = navActiveStyle
			if m.focus == focusSidebar {
				marker = "▸ "
			}
		}
		b.WriteString(marker + st.Render(label) + "\n")
	}
	return sidebarStyle.Height(height).Render(b.String())
}

func (m Model) renderMain(height int) string {
	switch m.section {
	case sectionStore:
		return m.renderComingSoon("Store", "A marketplace for shaw games is on the way.", height)
	case sectionKalama:
		return m.renderComingSoon("Kalama", "Build your own shaw games — engine tooling is on the way.", height)
	default:
		return m.renderLibrary(height)
	}
}

func (m Model) renderComingSoon(title, subtitle string, height int) string {
	body := lipgloss.JoinVertical(lipgloss.Center,
		comingSoonStyle.Render(title+" — coming soon"),
		subtleStyle.Render(subtitle),
	)
	w := m.width - sidebarW
	if w < 1 {
		w = 40
	}
	return lipgloss.NewStyle().Width(w).Height(height).
		Align(lipgloss.Center, lipgloss.Center).Render(body)
}

func (m Model) renderLibrary(height int) string {
	header := headerStyle.Render("Library") + "   " +
		countStyle.Render(fmt.Sprintf("%d games", len(m.games)))
	footer := footerStyle.Render("hjkl/↑↓←→ move · enter play · q quit")

	var body string
	if len(m.games) == 0 {
		body = subtleStyle.Render("No games installed — run `shaw install <game>`")
	} else {
		body = m.renderGrid(height)
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(
		lipgloss.JoinVertical(lipgloss.Left, header, "", body, "", footer),
	)
}

func (m Model) renderGrid(height int) string {
	cols := max1(m.cols)
	vis := m.visibleRows()
	var rows []string
	startRow := m.scroll
	for r := startRow; r < startRow+vis; r++ {
		first := r * cols
		if first >= len(m.games) {
			break
		}
		var cells []string
		for c := 0; c < cols; c++ {
			idx := first + c
			if idx >= len(m.games) {
				break
			}
			cells = append(cells, m.renderCard(idx))
			if c < cols-1 {
				cells = append(cells, strings.Repeat(" ", gap))
			}
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m Model) renderCard(idx int) string {
	g := m.games[idx]
	art, ok := m.banners[g.Name]
	var block string
	if ok && strings.TrimSpace(art) != "" {
		block = bannerBlock(art)
	} else {
		block = defaultBannerBlock(g.Name)
	}
	focused := m.focus == focusGrid && idx == m.gridCursor
	cs, ls := cardStyle, labelStyle
	if focused {
		cs, ls = cardFocusStyle, labelFocusStyle
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		cs.Render(block),
		ls.Render(displayName(g.Name)),
	)
}
