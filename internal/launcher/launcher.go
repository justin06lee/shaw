// Package launcher provides the styled shaw launcher UI: a sidebar plus a grid
// of installed games, and execs the chosen game so it owns the terminal.
package launcher

import (
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/justin06lee/shaw/internal/store"
)

// Exec runs the binary at binPath, inheriting the current stdio so the game owns
// the terminal; returns when the game exits.
func Exec(binPath string) error {
	c := exec.Command(binPath)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// loadBanners fetches each game's --banner art (best-effort; failures are
// silently skipped so the card falls back to the default banner).
func loadBanners(games []store.Manifest) map[string]string {
	banners := make(map[string]string, len(games))
	for _, g := range games {
		bp, err := store.BinaryPath(g.Name)
		if err != nil {
			continue
		}
		if art, err := bannerFetch(bp); err == nil {
			banners[g.Name] = art
		}
	}
	return banners
}

// Play launches a game. If name != "", it execs that installed game directly.
// Otherwise it shows the launcher UI, then execs the chosen game (or returns nil
// if the user quit without choosing).
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

	p := tea.NewProgram(NewModel(games, loadBanners(games)), tea.WithAltScreen())
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
