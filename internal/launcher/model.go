package launcher

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/justin06lee/shaw/internal/store"
)

type section int

const (
	sectionLibrary section = iota
	sectionStore
	sectionKalama
)

type focusZone int

const (
	focusSidebar focusZone = iota
	focusGrid
)

// Model is the launcher UI (pure, testable).
type Model struct {
	games   []store.Manifest
	banners map[string]string // game name -> raw --banner art

	section    section
	focus      focusZone
	navCursor  int // 0..2 sidebar index
	gridCursor int
	scroll     int // top visible grid row
	cols       int
	width      int
	height     int

	chosen string
	quit   bool
}

// NewModel builds the launcher. banners maps game name -> raw banner art (may be
// nil/partial; missing games get the default banner).
func NewModel(games []store.Manifest, banners map[string]string) Model {
	return Model{games: games, banners: banners, cols: 1}
}

func (m Model) Init() tea.Cmd { return nil }

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.cols = columns(msg.Width)
		m = m.clampScroll()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		}
		if m.focus == focusSidebar {
			return m.updateSidebar(msg)
		}
		return m.updateGrid(msg)
	}
	return m, nil
}

func (m Model) updateSidebar(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.navCursor < 2 {
			m.navCursor++
		}
	case "k", "up":
		if m.navCursor > 0 {
			m.navCursor--
		}
	case "l", "right", "enter":
		m.section = section(m.navCursor)
		if m.section == sectionLibrary && len(m.games) > 0 {
			m.focus = focusGrid
		}
	}
	return m, nil
}

func (m Model) updateGrid(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cols := max1(m.cols)
	switch msg.String() {
	case "h", "left":
		if m.gridCursor%cols == 0 {
			m.focus = focusSidebar
		} else {
			m.gridCursor--
		}
	case "l", "right":
		if m.gridCursor%cols != cols-1 && m.gridCursor < len(m.games)-1 {
			m.gridCursor++
		}
	case "j", "down":
		if m.gridCursor+cols < len(m.games) {
			m.gridCursor += cols
		}
	case "k", "up":
		if m.gridCursor-cols >= 0 {
			m.gridCursor -= cols
		}
	case "enter":
		if len(m.games) > 0 {
			m.chosen = m.games[m.gridCursor].Name
			m.quit = true
			return m, tea.Quit
		}
	}
	m = m.clampScroll()
	return m, nil
}

// visibleRows is how many card rows fit in the content viewport.
func (m Model) visibleRows() int {
	// content height = total minus header (1) and footer (1) and top/bottom pad (2)
	avail := m.height - 4
	if avail < rowStride {
		return 1
	}
	return avail / rowStride
}

// clampScroll keeps the focused card's row within the visible window.
func (m Model) clampScroll() Model {
	cols := max1(m.cols)
	row := m.gridCursor / cols
	vis := m.visibleRows()
	if row < m.scroll {
		m.scroll = row
	} else if row >= m.scroll+vis {
		m.scroll = row - vis + 1
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
	return m
}

// Chosen is the selected game's install id, or "" if none.
func (m Model) Chosen() string { return m.chosen }

// Quitting reports whether the program should exit.
func (m Model) Quitting() bool { return m.quit }
