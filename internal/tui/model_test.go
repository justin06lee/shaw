package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/justin06lee/shaw/internal/history"
	"github.com/justin06lee/shaw/internal/run"
)

// fixedSource is a WordSource that returns the same word forever, so model
// tests need no filesystem.
type fixedSource struct{ word string }

func (f fixedSource) Next() (string, bool) { return f.word, true }

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func newModel() Model {
	return New(fixedSource{word: "alpha"}, run.ModeTime, 30, 80, 24)
}

func TestModelStartsIdle(t *testing.T) {
	m := newModel()
	if m.State() != StateIdle {
		t.Errorf("got %v, want StateIdle", m.State())
	}
}

func TestConfigBarChangesModeWhenIdle(t *testing.T) {
	m := newModel()
	// Right arrow on the mode control moves time -> words.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(Model)
	if m.Mode() != run.ModeWords {
		t.Errorf("got mode %v, want ModeWords", m.Mode())
	}
}

func TestTypingTransitionsToActive(t *testing.T) {
	m := newModel()
	updated, _ := m.Update(keyMsg("a")) // first char of "alpha"
	m = updated.(Model)
	if m.State() != StateActive {
		t.Errorf("got %v, want StateActive", m.State())
	}
}

func TestConfigBarIgnoredWhileActive(t *testing.T) {
	m := newModel()
	updated, _ := m.Update(keyMsg("a")) // -> active
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight}) // arrow ignored
	m = updated.(Model)
	if m.Mode() != run.ModeTime {
		t.Errorf("mode changed during active run: got %v", m.Mode())
	}
}

func TestEscDuringActiveReturnsToIdle(t *testing.T) {
	m := newModel()
	updated, _ := m.Update(keyMsg("a"))
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.State() != StateIdle {
		t.Errorf("got %v, want StateIdle after Esc", m.State())
	}
}

func TestWordsRunReachingGoalShowsResult(t *testing.T) {
	// 1-word target "alpha": typing all 5 chars correctly hits the goal.
	m := New(fixedSource{word: "alpha"}, run.ModeWords, 1, 80, 24)
	for _, ch := range "alpha" {
		updated, _ := m.Update(keyMsg(string(ch)))
		m = updated.(Model)
	}
	if m.State() != StateResult {
		t.Errorf("got %v, want StateResult", m.State())
	}
}

func TestTickFinishesExpiredTimeRun(t *testing.T) {
	m := New(fixedSource{word: "alpha"}, run.ModeTime, 30, 80, 24)
	updated, _ := m.Update(keyMsg("a")) // start the run
	m = updated.(Model)
	// Force the run's clock so the goal is met, then deliver a tick.
	m.Run().Now = func() time.Time { return time.Now().Add(time.Hour) }
	updated, _ = m.Update(tickMsg(time.Now()))
	m = updated.(Model)
	if m.State() != StateResult {
		t.Errorf("got %v, want StateResult after expiry tick", m.State())
	}
}

func TestViewRendersWithoutPanic(t *testing.T) {
	m := newModel()
	if m.View() == "" {
		t.Error("idle view is empty")
	}
	updated, _ := m.Update(keyMsg("a"))
	m = updated.(Model)
	if m.View() == "" {
		t.Error("active view is empty")
	}
}

func TestFooterShowsRemainingTime(t *testing.T) {
	m := New(fixedSource{word: "alpha"}, run.ModeTime, 30, 80, 24)
	base := time.Unix(0, 0)
	clock := base
	m.Run().Now = func() time.Time { return clock }
	updated, _ := m.Update(keyMsg("a")) // first keystroke sets run start = base
	m = updated.(Model)
	clock = base.Add(10 * time.Second)
	if !strings.Contains(m.footer(), "20s") {
		t.Errorf("footer should show 20s remaining at +10s of a 30s run, got %q", m.footer())
	}
}

func TestFooterWordsProgress(t *testing.T) {
	m := New(fixedSource{word: "alpha"}, run.ModeWords, 10, 80, 24)
	// Type "alpha " — one full word plus its trailing space.
	for _, ch := range "alpha " {
		updated, _ := m.Update(keyMsg(string(ch)))
		m = updated.(Model)
	}
	if !strings.Contains(m.footer(), "1/10 words") {
		t.Errorf("footer should show 1/10 words after one word, got %q", m.footer())
	}
}

func TestFinishWritesHistory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New(fixedSource{word: "alpha"}, run.ModeWords, 1, 80, 24)
	for _, ch := range "alpha" {
		updated, _ := m.Update(keyMsg(string(ch)))
		m = updated.(Model)
	}
	if m.State() != StateResult {
		t.Fatalf("run did not finish: state %v", m.State())
	}
	recs, err := history.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d history records, want 1", len(recs))
	}
}
