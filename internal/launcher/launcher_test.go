package launcher

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/justin06lee/shaw/internal/store"
)

func games(n int) []store.Manifest {
	out := make([]store.Manifest, n)
	for i := range out {
		out[i] = store.Manifest{Name: "g" + string(rune('a'+i)), Version: "1.0.0"}
	}
	return out
}

func key(m Model, s string) Model {
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
	return u.(Model)
}

func size(m Model, w, h int) Model {
	u, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return u.(Model)
}

func TestStartsInSidebarOnLibrary(t *testing.T) {
	m := NewModel(games(3), nil)
	if m.focus != focusSidebar {
		t.Errorf("focus = %v, want sidebar", m.focus)
	}
	if m.section != sectionLibrary {
		t.Errorf("section = %v, want library", m.section)
	}
}

func TestSidebarNavAndEnterLibraryEntersGrid(t *testing.T) {
	m := size(NewModel(games(3), nil), 80, 40)
	m = key(m, "l") // enter Library -> grid
	if m.focus != focusGrid {
		t.Fatalf("focus = %v, want grid", m.focus)
	}
}

func TestSidebarStoreSwitchesSectionStaysSidebar(t *testing.T) {
	m := size(NewModel(games(3), nil), 80, 40)
	m = key(m, "j") // -> Store
	m = key(m, "enter")
	if m.section != sectionStore {
		t.Errorf("section = %v, want store", m.section)
	}
	if m.focus != focusSidebar {
		t.Errorf("focus = %v, want sidebar (store has no grid)", m.focus)
	}
}

func TestGridMovesAndHAtCol0ReturnsToSidebar(t *testing.T) {
	m := size(NewModel(games(6), nil), 200, 40) // wide -> several columns
	m = key(m, "l")                             // enter grid
	if m.cols < 2 {
		t.Fatalf("expected >=2 columns at width 200, got %d", m.cols)
	}
	m = key(m, "l") // move right within grid
	if m.gridCursor != 1 {
		t.Fatalf("gridCursor = %d, want 1", m.gridCursor)
	}
	m = key(m, "h") // back to col 0
	if m.gridCursor != 0 || m.focus != focusGrid {
		t.Fatalf("gridCursor=%d focus=%v, want 0/grid", m.gridCursor, m.focus)
	}
	m = key(m, "h") // at col 0 -> sidebar
	if m.focus != focusSidebar {
		t.Fatalf("focus = %v, want sidebar after h at col 0", m.focus)
	}
}

func TestGridEnterChoosesGame(t *testing.T) {
	m := size(NewModel(games(3), nil), 80, 40)
	m = key(m, "l")     // grid
	m = key(m, "enter") // choose g0
	if m.Chosen() != "ga" {
		t.Errorf("Chosen = %q, want ga", m.Chosen())
	}
	if !m.Quitting() {
		t.Error("Quitting = false, want true")
	}
}

func TestQuitChoosesNothing(t *testing.T) {
	m := NewModel(games(3), nil)
	m = key(m, "q")
	if !m.Quitting() || m.Chosen() != "" {
		t.Errorf("q: Quitting=%v Chosen=%q, want true/empty", m.Quitting(), m.Chosen())
	}
}

func TestScrollKeepsFocusedRowVisible(t *testing.T) {
	// short terminal, single column -> scrolling required
	m := size(NewModel(games(10), nil), 40, 24)
	m = key(m, "l") // grid
	start := m.scroll
	for i := 0; i < 9; i++ {
		m = key(m, "j")
	}
	if m.scroll <= start {
		t.Errorf("scroll did not advance: start=%d end=%d", start, m.scroll)
	}
	focusedRow := m.gridCursor / max1(m.cols)
	vis := m.visibleRows()
	if focusedRow < m.scroll || focusedRow >= m.scroll+vis {
		t.Errorf("focused row %d not in [%d,%d)", focusedRow, m.scroll, m.scroll+vis)
	}
}

func TestProgramRendersAndQuits(t *testing.T) {
	// stub banner fetch so no real binary is exec'd
	orig := bannerFetch
	bannerFetch = func(string) (string, error) { return "PIXELART", nil }
	defer func() { bannerFetch = orig }()

	m := size(NewModel(games(4), map[string]string{"ga": "PIXELART"}), 100, 40)
	// drive: enter grid, move right, down, back to sidebar, to store, quit
	for _, k := range []string{"l", "l", "j", "h", "h", "j", "enter"} {
		m = key(m, k)
	}
	// landed on Store via sidebar enter (no grid), so nothing chosen yet; quit
	m = key(m, "q")
	if !m.Quitting() {
		t.Fatal("expected quitting after q")
	}
	if out := m.View(); out == "" {
		t.Fatal("view rendered empty")
	}
}
