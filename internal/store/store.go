// Package store manages locally installed games under SHAW_HOME (~/.shaw).
package store

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// maxDownloadBytes caps the size of a downloaded asset so a hostile or broken
// asset cannot fill the disk. It is a var (not const) so tests can override it.
var maxDownloadBytes int64 = 200 << 20 // 200 MiB

// httpClient has a timeout so a hung server cannot block forever.
var httpClient = &http.Client{Timeout: 60 * time.Second}

// validComponent rejects anything that isn't a single, safe path element, so a
// malicious registry entry cannot escape SHAW_HOME via name/binary.
func validComponent(s string) error {
	if s == "" || s == "." || s == ".." ||
		strings.ContainsAny(s, `/\`) || filepath.Base(s) != s {
		return fmt.Errorf("invalid path component %q", s)
	}
	return nil
}

type Manifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Binary      string `json:"binary"`
}

// Home returns $SHAW_HOME or ~/.shaw.
func Home() (string, error) {
	if h := os.Getenv("SHAW_HOME"); h != "" {
		return h, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".shaw"), nil
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
	if err := validComponent(m.Name); err != nil {
		return err
	}
	if err := validComponent(m.Binary); err != nil {
		return err
	}
	gamesDir, err := GamesDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(gamesDir, m.Name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	resp, err := httpClient.Get(assetURL)
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

	n, err := io.Copy(tmp, io.LimitReader(resp.Body, maxDownloadBytes+1))
	if err != nil {
		tmp.Close()
		return err
	}
	if n > maxDownloadBytes {
		tmp.Close()
		return fmt.Errorf("download exceeds %d bytes", maxDownloadBytes)
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
	if err := validComponent(name); err != nil {
		return err
	}
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
	if err := validComponent(name); err != nil {
		return "", err
	}
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
	if err := validComponent(m.Binary); err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	binPath := filepath.Join(dir, m.Binary)
	if _, err := os.Stat(binPath); err != nil {
		return "", fmt.Errorf("%s: binary missing: %w", name, err)
	}
	return binPath, nil
}
