// Package registry fetches and parses the hegale game registry index.
package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// httpClient has a timeout so a hung registry server cannot block forever.
var httpClient = &http.Client{Timeout: 60 * time.Second}

type Game struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Binary      string            `json:"binary"`
	Assets      map[string]string `json:"assets"` // "os/arch" -> download URL
}

type Index struct {
	Games []Game `json:"games"`
}

const DefaultURL = "https://raw.githubusercontent.com/justin06lee/hegale/master/index.json"

// URL returns the registry index URL: $SHAW_REGISTRY if set, else DefaultURL.
func URL() string {
	if u := os.Getenv("SHAW_REGISTRY"); u != "" {
		return u
	}
	return DefaultURL
}

// Fetch GETs and parses the registry index from URL().
func Fetch() (*Index, error) {
	return FetchFrom(URL())
}

// FetchFrom GETs and parses the index from a specific url (non-2xx -> error).
func FetchFrom(url string) (*Index, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("registry %s: unexpected status %s", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var ix Index
	if err := json.Unmarshal(body, &ix); err != nil {
		return nil, fmt.Errorf("parse registry index: %w", err)
	}
	return &ix, nil
}

// Find returns the game with the given name.
func (ix *Index) Find(name string) (*Game, bool) {
	for i := range ix.Games {
		if ix.Games[i].Name == name {
			return &ix.Games[i], true
		}
	}
	return nil, false
}

// AssetURL returns the download URL for the given GOOS/GOARCH (key "goos/goarch").
func (g *Game) AssetURL(goos, goarch string) (string, bool) {
	u, ok := g.Assets[goos+"/"+goarch]
	return u, ok
}
