// Package store manages locally installed games under KALAMA_HOME (~/.kalama).
package store

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
)

type Manifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Binary      string `json:"binary"`
}

// Home returns $KALAMA_HOME or ~/.kalama.
func Home() (string, error) {
	if h := os.Getenv("KALAMA_HOME"); h != "" {
		return h, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kalama"), nil
}

// GamesDir returns Home()/games.
func GamesDir() (string, error) {
	h, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, "games"), nil
}

// Install downloads assetURL into GamesDir()/m.Name/m.Binary (chmod 0755) and
// writes manifest.json next to it. The download is atomic (temp file + rename)
// and a non-2xx HTTP status is an error. Creates directories as needed.
func Install(m Manifest, assetURL string) error {
	gamesDir, err := GamesDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(gamesDir, m.Name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	resp, err := http.Get(assetURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download %s: unexpected status %s", assetURL, resp.Status)
	}

	tmp, err := os.CreateTemp(dir, ".download-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after successful rename

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return err
	}

	binPath := filepath.Join(dir, m.Binary)
	if err := os.Rename(tmpName, binPath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o644)
}

// Remove deletes GamesDir()/name. Removing a non-installed game is not an error.
func Remove(name string) error {
	gamesDir, err := GamesDir()
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(gamesDir, name))
}

// List returns the manifest of every installed game (reads each
// GamesDir()/<name>/manifest.json), sorted by name. A missing games dir -> empty.
func List() ([]Manifest, error) {
	gamesDir, err := GamesDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(gamesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Manifest{}, nil
		}
		return nil, err
	}

	var out []Manifest
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(gamesDir, e.Name(), "manifest.json"))
		if err != nil {
			continue
		}
		var m Manifest
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// BinaryPath returns the absolute path to an installed game's executable,
// erroring if the game is not installed.
func BinaryPath(name string) (string, error) {
	gamesDir, err := GamesDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(gamesDir, name)
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return "", fmt.Errorf("%s is not installed", name)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return "", fmt.Errorf("%s: corrupt manifest: %w", name, err)
	}
	binPath := filepath.Join(dir, m.Binary)
	if _, err := os.Stat(binPath); err != nil {
		return "", fmt.Errorf("%s: binary missing: %w", name, err)
	}
	return binPath, nil
}
