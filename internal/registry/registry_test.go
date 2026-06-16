package registry

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const sampleIndex = `{"games":[{"name":"luma","description":"a glowing game","version":"1.0.0","binary":"luma","assets":{"darwin/arm64":"https://example.com/luma-darwin-arm64","linux/amd64":"https://example.com/luma-linux-amd64"}}]}`

func TestFetchFromParsesIndex(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleIndex))
	}))
	defer srv.Close()

	ix, err := FetchFrom(srv.URL)
	if err != nil {
		t.Fatalf("FetchFrom: %v", err)
	}
	if len(ix.Games) != 1 {
		t.Fatalf("got %d games, want 1", len(ix.Games))
	}

	g, ok := ix.Find("luma")
	if !ok {
		t.Fatal("Find(luma) = false, want true")
	}
	if g.Version != "1.0.0" || g.Binary != "luma" || g.Description != "a glowing game" {
		t.Errorf("unexpected game: %+v", g)
	}

	if _, ok := ix.Find("nope"); ok {
		t.Error("Find(nope) = true, want false")
	}
}

func TestAssetURL(t *testing.T) {
	g := &Game{
		Name: "luma",
		Assets: map[string]string{
			"darwin/arm64": "https://example.com/luma-darwin-arm64",
		},
	}
	u, ok := g.AssetURL("darwin", "arm64")
	if !ok {
		t.Fatal("AssetURL(darwin,arm64) = false, want true")
	}
	if u != "https://example.com/luma-darwin-arm64" {
		t.Errorf("AssetURL = %q", u)
	}
	if _, ok := g.AssetURL("plan9", "386"); ok {
		t.Error("AssetURL(plan9,386) = true, want false")
	}
}

func TestURL(t *testing.T) {
	t.Run("env override", func(t *testing.T) {
		t.Setenv("SHAW_REGISTRY", "https://custom.example.com/index.json")
		if got := URL(); got != "https://custom.example.com/index.json" {
			t.Errorf("URL() = %q", got)
		}
	})
	t.Run("default", func(t *testing.T) {
		t.Setenv("SHAW_REGISTRY", "")
		if got := URL(); got != DefaultURL {
			t.Errorf("URL() = %q, want default %q", got, DefaultURL)
		}
	})
}

func TestFetchFrom404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	if _, err := FetchFrom(srv.URL); err == nil {
		t.Error("FetchFrom on 404 = nil error, want error")
	}
}
