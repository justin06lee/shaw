package store

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const fakeBinary = "#!/bin/sh\necho hi\n"

func serveBinary(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fakeBinary))
	}))
}

func TestInstallListBinaryPathRemove(t *testing.T) {
	t.Setenv("KALAMA_HOME", t.TempDir())
	srv := serveBinary(t)
	defer srv.Close()

	m := Manifest{Name: "luma", Description: "a glowing game", Version: "1.0.0", Binary: "luma"}
	if err := Install(m, srv.URL); err != nil {
		t.Fatalf("Install: %v", err)
	}

	gamesDir, err := GamesDir()
	if err != nil {
		t.Fatal(err)
	}
	binPath := filepath.Join(gamesDir, "luma", "luma")

	info, err := os.Stat(binPath)
	if err != nil {
		t.Fatalf("stat binary: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o755 {
		t.Errorf("binary perm = %o, want 755", perm)
	}
	got, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != fakeBinary {
		t.Errorf("binary bytes = %q, want %q", got, fakeBinary)
	}

	// manifest round-trips
	mf, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(mf) != 1 {
		t.Fatalf("List len = %d, want 1", len(mf))
	}
	if mf[0] != m {
		t.Errorf("List[0] = %+v, want %+v", mf[0], m)
	}

	bp, err := BinaryPath("luma")
	if err != nil {
		t.Fatalf("BinaryPath: %v", err)
	}
	if bp != binPath {
		t.Errorf("BinaryPath = %q, want %q", bp, binPath)
	}

	if err := Remove("luma"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	mf, err = List()
	if err != nil {
		t.Fatalf("List after remove: %v", err)
	}
	if len(mf) != 0 {
		t.Errorf("List after remove len = %d, want 0", len(mf))
	}
}

func TestInstall404(t *testing.T) {
	t.Setenv("KALAMA_HOME", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	m := Manifest{Name: "luma", Version: "1.0.0", Binary: "luma"}
	if err := Install(m, srv.URL); err == nil {
		t.Error("Install on 404 = nil error, want error")
	}
}

func TestRemoveNonInstalled(t *testing.T) {
	t.Setenv("KALAMA_HOME", t.TempDir())
	if err := Remove("ghost"); err != nil {
		t.Errorf("Remove non-installed = %v, want nil", err)
	}
}

func TestListMissingDir(t *testing.T) {
	t.Setenv("KALAMA_HOME", t.TempDir())
	mf, err := List()
	if err != nil {
		t.Fatalf("List on missing games dir: %v", err)
	}
	if len(mf) != 0 {
		t.Errorf("List = %d games, want 0", len(mf))
	}
}

func TestBinaryPathNotInstalled(t *testing.T) {
	t.Setenv("KALAMA_HOME", t.TempDir())
	if _, err := BinaryPath("ghost"); err == nil {
		t.Error("BinaryPath for non-installed = nil error, want error")
	}
}
