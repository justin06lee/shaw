package launcher

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/justin06lee/kalama/internal/store"
)

func twoGames() []store.Manifest {
	return []store.Manifest{
		{Name: "luma", Version: "1.0.0"},
		{Name: "nova", Version: "2.0.0"},
	}
}

func send(m Model, key string) Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return updated.(Model)
}

func sendKey(m Model, t tea.KeyType) Model {
	updated, _ := m.Update(tea.KeyMsg{Type: t})
	return updated.(Model)
}

func TestCursorMovement(t *testing.T) {
	m := NewModel(twoGames())
	if m.Cursor() != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.Cursor())
	}

	m = sendKey(m, tea.KeyDown)
	if m.Cursor() != 1 {
		t.Errorf("after down cursor = %d, want 1", m.Cursor())
	}

	m = sendKey(m, tea.KeyDown)
	if m.Cursor() != 1 {
		t.Errorf("after down (clamp) cursor = %d, want 1", m.Cursor())
	}

	m = sendKey(m, tea.KeyUp)
	if m.Cursor() != 0 {
		t.Errorf("after up cursor = %d, want 0", m.Cursor())
	}

	m = sendKey(m, tea.KeyUp)
	if m.Cursor() != 0 {
		t.Errorf("after up (clamp) cursor = %d, want 0", m.Cursor())
	}
}

func TestEnterSelects(t *testing.T) {
	m := NewModel(twoGames())
	m = sendKey(m, tea.KeyDown)
	m = sendKey(m, tea.KeyEnter)

	if m.Chosen() != "nova" {
		t.Errorf("Chosen = %q, want nova", m.Chosen())
	}
	if !m.Quitting() {
		t.Error("Quitting = false after enter, want true")
	}
}

func TestQuitNoChoice(t *testing.T) {
	m := NewModel(twoGames())
	m = send(m, "q")

	if !m.Quitting() {
		t.Error("Quitting = false after q, want true")
	}
	if m.Chosen() != "" {
		t.Errorf("Chosen = %q after q, want empty", m.Chosen())
	}
}
