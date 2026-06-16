// Package launcher provides a bubbletea menu over installed games and execs the
// chosen game so it owns the terminal.
package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/justin06lee/shaw/internal/store"
)

// Model is the launcher menu (pure, testable). ↑/↓ move; enter selects; q/esc quit.
type Model struct {
	games  []store.Manifest
	cursor int
	chosen string // set to the selected game's name on enter
	quit   bool
}

func NewModel(games []store.Manifest) Model {
	return Model{games: games}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.games)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.games) > 0 {
				m.chosen = m.games[m.cursor].Name
			}
			m.quit = true
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	if len(m.games) == 0 {
		return "no games installed — try: shaw install snake.shaw\n"
	}
	var b strings.Builder
	b.WriteString("shaw arcade — pick a game\n\n")
	for i, g := range m.games {
		marker := "  "
		if i == m.cursor {
			marker = "> "
		}
		fmt.Fprintf(&b, "%s%s  %s\n", marker, displayName(g.Name), g.Description)
	}
	b.WriteString("\n↑/↓ move · enter play · q quit\n")
	return b.String()
}

// Accessors for tests/driver:
func (m Model) Cursor() int    { return m.cursor }
func (m Model) Chosen() string { return m.chosen }
func (m Model) Quitting() bool { return m.quit }

// Exec runs the binary at binPath, inheriting the current stdio so the game owns
// the terminal; returns when the game exits.
func Exec(binPath string) error {
	c := exec.Command(binPath)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// Play launches a game. If name != "", it execs that installed game directly.
// Otherwise it shows the menu, then execs the chosen game (or returns nil if the
// user quit without choosing).
func Play(name string) error {
	if name != "" {
		binPath, err := store.BinaryPath(name)
		if err != nil {
			return err
		}
		return Exec(binPath)
	}

	games, err := store.List()
	if err != nil {
		return err
	}

	p := tea.NewProgram(NewModel(games))
	final, err := p.Run()
	if err != nil {
		return err
	}
	m := final.(Model)
	if m.Chosen() == "" {
		return nil
	}
	binPath, err := store.BinaryPath(m.Chosen())
	if err != nil {
		return err
	}
	return Exec(binPath)
}
