package launcher

import (
	"strings"
	"testing"
)

func TestFitBannerClipsAndPads(t *testing.T) {
	art := "abcdefghijklmnopqrstuvwxyz\nshort"
	lines := fitBanner(art, artW, artH)
	if len(lines) != artH {
		t.Fatalf("got %d lines, want %d", len(lines), artH)
	}
	for i, ln := range lines {
		if w := lineWidth(ln); w != artW {
			t.Errorf("line %d width = %d, want %d (%q)", i, w, artW, ln)
		}
	}
	// the long line must have been clipped to artW
	if !strings.Contains(strings.Join(lines, "\n"), "abcdefghijklmno") {
		t.Errorf("clipped long line missing; got:\n%s", strings.Join(lines, "\n"))
	}
}

func TestFitBannerCentersSmallArt(t *testing.T) {
	lines := fitBanner("hi", artW, artH)
	// first and last rows are blank padding (vertical centering)
	if strings.TrimSpace(lines[0]) != "" {
		t.Errorf("top row not blank: %q", lines[0])
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "hi") {
		t.Errorf("centered art missing 'hi':\n%s", joined)
	}
}

func TestDefaultBannerBlockHasName(t *testing.T) {
	block := defaultBannerBlock("snake.shaw")
	if !strings.Contains(block, "snake") {
		t.Errorf("default banner missing friendly name; got:\n%s", block)
	}
	if n := strings.Count(block, "\n"); n != artH-1 {
		t.Errorf("default banner has %d newlines, want %d", n, artH-1)
	}
}

func TestDisplayNameTrimsSuffix(t *testing.T) {
	if got := displayName("snake.shaw"); got != "snake" {
		t.Errorf("displayName = %q, want snake", got)
	}
	if got := displayName("nova"); got != "nova" {
		t.Errorf("displayName = %q, want nova", got)
	}
}
