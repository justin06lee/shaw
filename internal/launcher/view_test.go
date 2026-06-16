package launcher

import (
	"strings"
	"testing"
)

func TestColumns(t *testing.T) {
	cases := []struct {
		width, want int
	}{
		{0, 1},
		{20, 1},
		{sidebarW + 2 + cardW, 1},
		{sidebarW + 2 + cardW*3 + gap*2, 3},
		{300, (300 - sidebarW - 2 + gap) / (cardW + gap)},
	}
	for _, c := range cases {
		if got := columns(c.width); got != c.want {
			t.Errorf("columns(%d) = %d, want %d", c.width, got, c.want)
		}
	}
}

func TestViewLibraryShowsSidebarAndGame(t *testing.T) {
	m := size(NewModel(games(2), nil), 100, 40)
	out := m.View()
	for _, want := range []string{"SHAW", "Library", "Store", "Kalama", "ga"} {
		if !strings.Contains(out, want) {
			t.Errorf("library view missing %q", want)
		}
	}
}

func TestViewStoreShowsComingSoon(t *testing.T) {
	m := size(NewModel(games(2), nil), 100, 40)
	m = key(m, "j")     // -> Store
	m = key(m, "enter") // activate
	out := m.View()
	if !strings.Contains(out, "coming soon") {
		t.Errorf("store view missing 'coming soon':\n%s", out)
	}
}

func TestViewEmptyLibrary(t *testing.T) {
	m := size(NewModel(nil, nil), 100, 40)
	out := m.View()
	if !strings.Contains(out, "No games installed") {
		t.Errorf("empty library missing message:\n%s", out)
	}
}
