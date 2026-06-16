# Shaw Launcher UI (Epic-Style) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the `shaw` launcher's plain text menu with a lipgloss-styled, Epic-Games-inspired TUI: a Library/Store/Kalama sidebar and a scrollable, vim+arrow-navigable grid of game cards with ASCII-art banners fetched via a `--banner` game contract.

**Architecture:** Split `internal/launcher/` into focused files — `styles.go` (theme + layout constants), `banner.go` (fetch/fit/default, pure + an injectable exec var), `model.go` (bubbletea Model + Update: sidebar↔grid focus, 2D grid nav, scroll), `view.go` (lipgloss composition + column math), `launcher.go` (unchanged public `Play`/`Exec`, now builds the banner cache). The game side (`snake.shaw`) gains an embedded `banner.md` and a `--banner` flag.

**Tech Stack:** Go 1.26.2, charmbracelet/bubbletea, charmbracelet/lipgloss (promoted to direct), mattn/go-runewidth (promoted to direct), teatest.

**Repos:** launcher work in `/Volumes/T7/Stockpile/Workspace/github.com/justin06lee/shaw`; game work in `/Volumes/T7/Stockpile/Workspace/github.com/justin06lee/snake.shaw`.

**Commit trailer (every commit):**
```
Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>
```

## File structure (launcher repo, `internal/launcher/`)

| File | Responsibility |
|------|----------------|
| `styles.go` | Layout constants (`artW`,`artH`,`cardW`,`gap`,`sidebarW`) + all lipgloss styles (the theme). |
| `banner.go` | `bannerFetch` exec var, `fitBanner`, `bannerBlock`, `defaultBannerBlock`, `displayName` (moved here). |
| `model.go` | `Model`, section/focus enums, `NewModel`, `Init`, `Update`, `clampScroll`, `visibleRows`, accessors `Chosen`/`Quitting`. |
| `view.go` | `View`, `columns`, sidebar/main/grid rendering. |
| `launcher.go` | `Play`, `Exec` (build banner cache, run program, exec chosen). |
| `launcher_test.go` | rewritten model/nav/integration tests. |
| `banner_test.go` | `fitBanner`/`defaultBannerBlock` tests. |
| `view_test.go` | `columns` + render-substring tests. |

Constants used throughout: `artW=15`, `artH=7`, `cardW=artW+2=17` (left/right border), `gap=2`, `sidebarW=12`. Card occupies `artH+2=9` border rows + 1 label row + 1 row gap = `rowStride=11`.

---

### Task 1: Theme & layout constants (`styles.go`)

**Files:**
- Create: `internal/launcher/styles.go`
- Modify: `go.mod` (promote lipgloss + go-runewidth to direct)

- [ ] **Step 1: Create `styles.go`**

```go
package launcher

import "github.com/charmbracelet/lipgloss"

// Layout constants (character cells).
const (
	artW     = 15 // banner art inner width
	artH     = 7  // banner art inner height
	cardW    = artW + 2 // card outer width incl. left/right border
	gap      = 2  // horizontal gap between cards
	sidebarW = 12 // sidebar inner width
	rowStride = artH + 2 + 1 + 1 // card border rows + label + row gap
)

// Palette.
var (
	accent   = lipgloss.Color("#7C5CFF")
	fgBright = lipgloss.Color("#FFFFFF")
	fgDim    = lipgloss.Color("#8A8A99")
)

// Styles.
var (
	wordmarkStyle   = lipgloss.NewStyle().Foreground(fgBright).Bold(true)
	navItemStyle    = lipgloss.NewStyle().Foreground(fgDim)
	navActiveStyle  = lipgloss.NewStyle().Foreground(accent).Bold(true)
	sidebarStyle    = lipgloss.NewStyle().Width(sidebarW).Padding(1, 1)

	cardStyle      = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(fgDim)
	cardFocusStyle = lipgloss.NewStyle().Border(lipgloss.ThickBorder()).BorderForeground(accent)
	labelStyle      = lipgloss.NewStyle().Foreground(fgDim).Width(cardW).Align(lipgloss.Center)
	labelFocusStyle = lipgloss.NewStyle().Foreground(fgBright).Bold(true).Width(cardW).Align(lipgloss.Center)

	defaultBannerStyle = lipgloss.NewStyle().
				Width(artW).Height(artH).
				Align(lipgloss.Center, lipgloss.Center).
				Foreground(fgBright).Background(accent).Bold(true)

	headerStyle     = lipgloss.NewStyle().Foreground(fgBright).Bold(true)
	countStyle      = lipgloss.NewStyle().Foreground(fgDim)
	comingSoonStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	subtleStyle     = lipgloss.NewStyle().Foreground(fgDim)
	footerStyle     = lipgloss.NewStyle().Foreground(fgDim)
)
```

- [ ] **Step 2: Promote dependencies to direct and build**

```bash
cd /Volumes/T7/Stockpile/Workspace/github.com/justin06lee/shaw
go get github.com/charmbracelet/lipgloss@v1.1.0
go get github.com/mattn/go-runewidth@v0.0.16
go build ./... 2>&1 | tail -5
```
Expected: builds (styles.go compiles; unused vars are package-level so no error).

- [ ] **Step 3: Commit**

```bash
git add internal/launcher/styles.go go.mod go.sum
git commit -m "feat(launcher): theme palette and layout constants

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Banner module (`banner.go`)

**Files:**
- Create: `internal/launcher/banner.go`, `internal/launcher/banner_test.go`
- Note: `displayName` currently lives in `launcher.go`; it moves here. Remove it from `launcher.go` in Task 5.

- [ ] **Step 1: Write `banner_test.go`**

```go
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
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/launcher/ -run 'Banner|DisplayName' 2>&1 | tail -8
```
Expected: FAIL — `undefined: fitBanner`, `lineWidth`, `defaultBannerBlock`, `displayName`.

- [ ] **Step 3: Create `banner.go`**

```go
package launcher

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
)

// bannerFetch returns the raw `--banner` output for the binary at binPath.
// It is a package var so tests can stub it instead of execing real binaries.
var bannerFetch = execBanner

func execBanner(binPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, binPath, "--banner").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// displayName is the friendly name shown in the UI: the install id without the
// .shaw suffix (e.g. "snake.shaw" -> "snake").
func displayName(id string) string { return strings.TrimSuffix(id, ".shaw") }

// lineWidth is the display width of a single line of art (multibyte-safe).
func lineWidth(s string) int { return runewidth.StringWidth(s) }

// fitBanner returns exactly h lines, each exactly w display columns wide, with
// the art clipped to fit and centered horizontally and vertically.
func fitBanner(art string, w, h int) []string {
	raw := strings.Split(strings.TrimRight(art, "\n"), "\n")
	var lines []string
	for _, ln := range raw {
		lines = append(lines, runewidth.Truncate(ln, w, ""))
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	out := make([]string, h)
	for i := range out {
		out[i] = strings.Repeat(" ", w)
	}
	top := (h - len(lines)) / 2
	for i, ln := range lines {
		pad := (w - runewidth.StringWidth(ln)) / 2
		if pad < 0 {
			pad = 0
		}
		out[top+i] = runewidth.FillRight(strings.Repeat(" ", pad)+ln, w)
	}
	return out
}

// bannerBlock renders fitted art as an artW x artH block string (joined lines).
func bannerBlock(art string) string {
	return strings.Join(fitBanner(art, artW, artH), "\n")
}

// defaultBannerBlock renders the styled fallback banner (friendly name on an
// accent block) as an artW x artH block string.
func defaultBannerBlock(name string) string {
	return defaultBannerStyle.Render(displayName(name))
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/launcher/ -run 'Banner|DisplayName' 2>&1 | tail -8
```
Expected: PASS. (`defaultBannerStyle.Render` produces exactly `artH` lines because the style sets `Height(artH)`.)

- [ ] **Step 5: Commit**

```bash
git add internal/launcher/banner.go internal/launcher/banner_test.go
git commit -m "feat(launcher): banner fetch, fit/clip, and default banner

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Model & navigation (`model.go`), rewrite model tests

**Files:**
- Create: `internal/launcher/model.go`
- Rewrite: `internal/launcher/launcher_test.go` (old `Cursor()`-based tests are obsolete)
- Note: the `Model` type + `Chosen`/`Quitting` currently live in `launcher.go`. They move to `model.go`; Task 5 removes the old definitions from `launcher.go`. To avoid duplicate-symbol build errors between tasks, this task also deletes the now-conflicting parts of `launcher.go` (the `Model` struct, `NewModel`, `Init`, `Update`, `View`, accessors, `displayName`), leaving only `Exec`/`Play` (fixed up in Task 5). Do Step 4 below to gut `launcher.go` before building.

- [ ] **Step 1: Rewrite `launcher_test.go`**

```go
package launcher

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/justin06lee/shaw/internal/store"
)

func games(n int) []store.Manifest {
	out := make([]store.Manifest, n)
	for i := range out {
		out[i] = store.Manifest{Name: "g" + string(rune('a'+i)), Version: "1.0.0"}
	}
	return out
}

func key(m Model, s string) Model {
	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
	return u.(Model)
}

func size(m Model, w, h int) Model {
	u, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return u.(Model)
}

func TestStartsInSidebarOnLibrary(t *testing.T) {
	m := NewModel(games(3), nil)
	if m.focus != focusSidebar {
		t.Errorf("focus = %v, want sidebar", m.focus)
	}
	if m.section != sectionLibrary {
		t.Errorf("section = %v, want library", m.section)
	}
}

func TestSidebarNavAndEnterLibraryEntersGrid(t *testing.T) {
	m := size(NewModel(games(3), nil), 80, 40)
	m = key(m, "l") // enter Library -> grid
	if m.focus != focusGrid {
		t.Fatalf("focus = %v, want grid", m.focus)
	}
}

func TestSidebarStoreSwitchesSectionStaysSidebar(t *testing.T) {
	m := size(NewModel(games(3), nil), 80, 40)
	m = key(m, "j") // -> Store
	m = key(m, "enter")
	if m.section != sectionStore {
		t.Errorf("section = %v, want store", m.section)
	}
	if m.focus != focusSidebar {
		t.Errorf("focus = %v, want sidebar (store has no grid)", m.focus)
	}
}

func TestGridMovesAndHAtCol0ReturnsToSidebar(t *testing.T) {
	m := size(NewModel(games(6), nil), 200, 40) // wide -> several columns
	m = key(m, "l")             // enter grid
	if m.cols < 2 {
		t.Fatalf("expected >=2 columns at width 200, got %d", m.cols)
	}
	m = key(m, "l") // move right within grid
	if m.gridCursor != 1 {
		t.Fatalf("gridCursor = %d, want 1", m.gridCursor)
	}
	m = key(m, "h") // back to col 0
	if m.gridCursor != 0 || m.focus != focusGrid {
		t.Fatalf("gridCursor=%d focus=%v, want 0/grid", m.gridCursor, m.focus)
	}
	m = key(m, "h") // at col 0 -> sidebar
	if m.focus != focusSidebar {
		t.Fatalf("focus = %v, want sidebar after h at col 0", m.focus)
	}
}

func TestGridEnterChoosesGame(t *testing.T) {
	m := size(NewModel(games(3), nil), 80, 40)
	m = key(m, "l")     // grid
	m = key(m, "enter") // choose g0
	if m.Chosen() != "ga" {
		t.Errorf("Chosen = %q, want ga", m.Chosen())
	}
	if !m.Quitting() {
		t.Error("Quitting = false, want true")
	}
}

func TestQuitChoosesNothing(t *testing.T) {
	m := NewModel(games(3), nil)
	m = key(m, "q")
	if !m.Quitting() || m.Chosen() != "" {
		t.Errorf("q: Quitting=%v Chosen=%q, want true/empty", m.Quitting(), m.Chosen())
	}
}

func TestScrollKeepsFocusedRowVisible(t *testing.T) {
	// short terminal, single column -> scrolling required
	m := size(NewModel(games(10), nil), 40, 24)
	m = key(m, "l") // grid
	start := m.scroll
	for i := 0; i < 9; i++ {
		m = key(m, "j")
	}
	if m.scroll <= start {
		t.Errorf("scroll did not advance: start=%d end=%d", start, m.scroll)
	}
	focusedRow := m.gridCursor / max1(m.cols)
	vis := m.visibleRows()
	if focusedRow < m.scroll || focusedRow >= m.scroll+vis {
		t.Errorf("focused row %d not in [%d,%d)", focusedRow, m.scroll, m.scroll+vis)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/launcher/ 2>&1 | tail -8
```
Expected: FAIL/compile error — `focus`, `section`, `NewModel(…, nil)`, `cols`, `gridCursor`, `scroll`, `visibleRows`, `max1` undefined (Model still old in launcher.go).

- [ ] **Step 3: Create `model.go`**

```go
package launcher

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/justin06lee/shaw/internal/store"
)

type section int

const (
	sectionLibrary section = iota
	sectionStore
	sectionKalama
)

type focusZone int

const (
	focusSidebar focusZone = iota
	focusGrid
)

// Model is the launcher UI (pure, testable).
type Model struct {
	games   []store.Manifest
	banners map[string]string // game name -> raw --banner art

	section   section
	focus     focusZone
	navCursor int // 0..2 sidebar index
	gridCursor int
	scroll    int // top visible grid row
	cols      int
	width     int
	height    int

	chosen string
	quit   bool
}

// NewModel builds the launcher. banners maps game name -> raw banner art (may be
// nil/partial; missing games get the default banner).
func NewModel(games []store.Manifest, banners map[string]string) Model {
	return Model{games: games, banners: banners, cols: 1}
}

func (m Model) Init() tea.Cmd { return nil }

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.cols = columns(msg.Width)
		m = m.clampScroll()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		}
		if m.focus == focusSidebar {
			return m.updateSidebar(msg)
		}
		return m.updateGrid(msg)
	}
	return m, nil
}

func (m Model) updateSidebar(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.navCursor < 2 {
			m.navCursor++
		}
	case "k", "up":
		if m.navCursor > 0 {
			m.navCursor--
		}
	case "l", "right", "enter":
		m.section = section(m.navCursor)
		if m.section == sectionLibrary && len(m.games) > 0 {
			m.focus = focusGrid
		}
	}
	return m, nil
}

func (m Model) updateGrid(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cols := max1(m.cols)
	switch msg.String() {
	case "h", "left":
		if m.gridCursor%cols == 0 {
			m.focus = focusSidebar
		} else {
			m.gridCursor--
		}
	case "l", "right":
		if m.gridCursor%cols != cols-1 && m.gridCursor < len(m.games)-1 {
			m.gridCursor++
		}
	case "j", "down":
		if m.gridCursor+cols < len(m.games) {
			m.gridCursor += cols
		}
	case "k", "up":
		if m.gridCursor-cols >= 0 {
			m.gridCursor -= cols
		}
	case "enter":
		if len(m.games) > 0 {
			m.chosen = m.games[m.gridCursor].Name
			m.quit = true
			return m, tea.Quit
		}
	}
	m = m.clampScroll()
	return m, nil
}

// visibleRows is how many card rows fit in the content viewport.
func (m Model) visibleRows() int {
	// content height = total minus header (1) and footer (1) and top/bottom pad (2)
	avail := m.height - 4
	if avail < rowStride {
		return 1
	}
	return avail / rowStride
}

// clampScroll keeps the focused card's row within the visible window.
func (m Model) clampScroll() Model {
	cols := max1(m.cols)
	row := m.gridCursor / cols
	vis := m.visibleRows()
	if row < m.scroll {
		m.scroll = row
	} else if row >= m.scroll+vis {
		m.scroll = row - vis + 1
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
	return m
}

// Chosen is the selected game's install id, or "" if none.
func (m Model) Chosen() string { return m.chosen }

// Quitting reports whether the program should exit.
func (m Model) Quitting() bool { return m.quit }

// View is a temporary stub so Model satisfies tea.Model and the package builds.
// Task 4 replaces it with the real implementation in view.go (delete this stub then).
func (m Model) View() string { return "" }
```

- [ ] **Step 4: Gut the old Model out of `launcher.go`**

Open `internal/launcher/launcher.go` and delete everything EXCEPT the `Exec` and `Play` funcs and the package/import lines they need. Specifically remove: the `Model` struct, `NewModel`, `Init`, `Update`, `View`, `Cursor`/`Chosen`/`Quitting` accessors, and the `displayName` func (now in banner.go). Leave `Exec` and `Play` for now even though `Play` references the old `NewModel(games)` — it is fixed in Task 5. Temporarily, to keep the package compiling for THIS task's tests, change the `Play` body's program construction line to use the new signature:

Replace, in `launcher.go`, the line:
```go
	p := tea.NewProgram(NewModel(games))
```
with:
```go
	p := tea.NewProgram(NewModel(games, nil))
```
(Task 5 replaces this with the real banner-cache wiring.)

- [ ] **Step 5: Run tests to confirm they pass**

```bash
go build ./... && go test ./internal/launcher/ 2>&1 | tail -12
```
Expected: PASS for all model tests. (`columns` is referenced but defined in Task 4 — see note.) `columns` is referenced by `Update`/`clampScroll` but defined in Task 4's `view.go`. To keep this task building, add the REAL `columns` implementation to `model.go` now (Task 4 relocates it to `view.go` — single definition, just moved):
```go
func columns(totalWidth int) int {
	avail := totalWidth - sidebarW - 2
	if avail < cardW {
		return 1
	}
	return (avail + gap) / (cardW + gap)
}
```

- [ ] **Step 6: Commit**

```bash
git add internal/launcher/model.go internal/launcher/launcher.go internal/launcher/launcher_test.go
git commit -m "feat(launcher): model with sidebar/grid focus, 2D nav, scroll

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: View rendering (`view.go`)

**Files:**
- Create: `internal/launcher/view.go`, `internal/launcher/view_test.go`
- Modify: `internal/launcher/model.go` (remove the `columns` function — it moves to `view.go`)

- [ ] **Step 1: Write `view_test.go`**

```go
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
```

- [ ] **Step 2: Run to confirm fail**

```bash
go test ./internal/launcher/ -run 'Columns|View' 2>&1 | tail -8
```
Expected: FAIL — `m.View undefined` (View was removed from launcher.go in Task 3).

- [ ] **Step 3: Remove `columns` + the temporary `View` stub from `model.go`, then create `view.go`**

Delete the `columns` func AND the temporary `func (m Model) View() string { return "" }` stub from `model.go` (both move/are-replaced here). Create `internal/launcher/view.go`:

```go
package launcher

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// columns computes how many cards fit across the content area for a terminal width.
func columns(totalWidth int) int {
	avail := totalWidth - sidebarW - 2
	if avail < cardW {
		return 1
	}
	return (avail + gap) / (cardW + gap)
}

var navLabels = []string{"Library", "Store", "Kalama"}

func (m Model) View() string {
	height := m.height
	if height < 1 {
		height = 24
	}
	sidebar := m.renderSidebar(height)
	main := m.renderMain(height)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
}

func (m Model) renderSidebar(height int) string {
	var b strings.Builder
	b.WriteString(wordmarkStyle.Render("SHAW"))
	b.WriteString("\n\n")
	for i, label := range navLabels {
		marker := "  "
		st := navItemStyle
		if section(i) == m.section {
			st = navActiveStyle
			if m.focus == focusSidebar {
				marker = "▸ "
			}
		}
		b.WriteString(marker + st.Render(label) + "\n")
	}
	return sidebarStyle.Height(height).Render(b.String())
}

func (m Model) renderMain(height int) string {
	switch m.section {
	case sectionStore:
		return m.renderComingSoon("Store", "A marketplace for shaw games is on the way.", height)
	case sectionKalama:
		return m.renderComingSoon("Kalama", "Build your own shaw games — engine tooling is on the way.", height)
	default:
		return m.renderLibrary(height)
	}
}

func (m Model) renderComingSoon(title, subtitle string, height int) string {
	body := lipgloss.JoinVertical(lipgloss.Center,
		comingSoonStyle.Render(title+" — coming soon"),
		subtleStyle.Render(subtitle),
	)
	w := m.width - sidebarW
	if w < 1 {
		w = 40
	}
	return lipgloss.NewStyle().Width(w).Height(height).
		Align(lipgloss.Center, lipgloss.Center).Render(body)
}

func (m Model) renderLibrary(height int) string {
	header := headerStyle.Render("Library") + "   " +
		countStyle.Render(fmt.Sprintf("%d games", len(m.games)))
	footer := footerStyle.Render("hjkl/↑↓←→ move · enter play · q quit")

	var body string
	if len(m.games) == 0 {
		body = subtleStyle.Render("No games installed — run `shaw install <game>`")
	} else {
		body = m.renderGrid(height)
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(
		lipgloss.JoinVertical(lipgloss.Left, header, "", body, "", footer),
	)
}

func (m Model) renderGrid(height int) string {
	cols := max1(m.cols)
	vis := m.visibleRows()
	var rows []string
	startRow := m.scroll
	for r := startRow; r < startRow+vis; r++ {
		first := r * cols
		if first >= len(m.games) {
			break
		}
		var cells []string
		for c := 0; c < cols; c++ {
			idx := first + c
			if idx >= len(m.games) {
				break
			}
			cells = append(cells, m.renderCard(idx))
			if c < cols-1 {
				cells = append(cells, strings.Repeat(" ", gap))
			}
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m Model) renderCard(idx int) string {
	g := m.games[idx]
	art, ok := m.banners[g.Name]
	var block string
	if ok && strings.TrimSpace(art) != "" {
		block = bannerBlock(art)
	} else {
		block = defaultBannerBlock(g.Name)
	}
	focused := m.focus == focusGrid && idx == m.gridCursor
	cs, ls := cardStyle, labelStyle
	if focused {
		cs, ls = cardFocusStyle, labelFocusStyle
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		cs.Render(block),
		ls.Render(displayName(g.Name)),
	)
}
```

- [ ] **Step 4: Run to confirm pass**

```bash
go build ./... && go test ./internal/launcher/ -run 'Columns|View' 2>&1 | tail -8
```
Expected: PASS. Then run the full package: `go test ./internal/launcher/` → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/launcher/view.go internal/launcher/view_test.go internal/launcher/model.go
git commit -m "feat(launcher): lipgloss view — sidebar, grid, coming-soon panes

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Wire `Play` to build the banner cache + alt-screen; integration test

**Files:**
- Modify: `internal/launcher/launcher.go`
- Modify: `internal/launcher/launcher_test.go` (append a teatest integration test)

- [ ] **Step 1: Rewrite `launcher.go`**

Final `launcher.go` (only `Exec` + `Play`; everything else now lives in model/view/banner):

```go
// Package launcher provides the styled shaw launcher UI: a sidebar plus a grid
// of installed games, and execs the chosen game so it owns the terminal.
package launcher

import (
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/justin06lee/shaw/internal/store"
)

// Exec runs the binary at binPath, inheriting stdio so the game owns the
// terminal; returns when the game exits.
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
```

- [ ] **Step 2: Append an integration test to `launcher_test.go`**

```go
func TestProgramRendersAndQuits(t *testing.T) {
	// stub banner fetch so no real binary is exec'd
	orig := bannerFetch
	bannerFetch = func(string) (string, error) { return "PIXELART", nil }
	defer func() { bannerFetch = orig }()

	m := size(NewModel(games(4), map[string]string{"ga": "PIXELART"}), 100, 40)
	// drive: enter grid, move right, down, switch to store, back to library, quit
	for _, k := range []string{"l", "l", "j", "h", "h", "j", "enter"} {
		m = key(m, k)
	}
	// landed on Store via sidebar enter (no grid), so nothing chosen yet; quit
	m = key(m, "q")
	if !m.Quitting() {
		t.Fatal("expected quitting after q")
	}
	if out := m.View(); out == "" {
		t.Fatal("view rendered empty")
	}
}
```

- [ ] **Step 3: Build, vet, test the whole module**

```bash
cd /Volumes/T7/Stockpile/Workspace/github.com/justin06lee/shaw
go build ./... && go vet ./... && go test ./... 2>&1 | tail -15
```
Expected: all PASS.

- [ ] **Step 4: Manual smoke (TTY) — verify it renders**

```bash
SHAW_HOME=$(mktemp -d) go run ./cmd/shaw   # empty library: sidebar + "No games installed"
```
Expected: styled sidebar (SHAW / Library / Store / Kalama) and the empty-state message; `q` quits cleanly. (Document as human-verified; not auto-asserted.)

- [ ] **Step 5: Commit**

```bash
git add internal/launcher/launcher.go internal/launcher/launcher_test.go
git commit -m "feat(launcher): wire banner cache and alt-screen into Play

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Game contract — `snake.shaw` gains `banner.md` + `--banner`

**Files (in `snake.shaw` repo):**
- Create: `banner.md`
- Modify: `main.go` (embed + `--banner` flag)
- Modify: `README.md` (document the contract)

- [ ] **Step 1: Create `banner.md`** (ASCII art ≤ 15 wide × 7 tall)

```
  ___________
 /  o      o \
 \__________/
   ~~snake~~
```

(Authored to fit the 15×7 art area; the launcher centers/clips it.)

- [ ] **Step 2: Add embed + `--banner` to `main.go`**

At the top of `main.go`, add the embed import and directive (place the `import "embed"` with the other imports and the directive above `func main`):

```go
import _ "embed"

//go:embed banner.md
var bannerArt string
```

Then extend the early-flag handling in `main()`:

```go
func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version":
			fmt.Println("snake.shaw " + version)
			return
		case "--banner":
			fmt.Print(bannerArt)
			return
		}
	}
	// ... existing game startup unchanged ...
}
```

- [ ] **Step 3: Build and verify both flags**

```bash
cd /Volumes/T7/Stockpile/Workspace/github.com/justin06lee/snake.shaw
go build -o /tmp/snake.shaw . && /tmp/snake.shaw --version && echo "---" && /tmp/snake.shaw --banner && rm /tmp/snake.shaw
```
Expected: `snake.shaw 1.0.0`, then `---`, then the banner art.

- [ ] **Step 4: Document the contract in `README.md`**

Append a section:
```markdown
## Shaw game contract

Every shaw game is a single binary that supports:

- `--version` → prints `<name> <version>` and exits.
- `--banner`  → prints its ASCII-art banner (authored in `banner.md`, embedded at
  build time) and exits. The `shaw` launcher reads this to draw the game's card;
  art is sized to a 15×7 area.
```

- [ ] **Step 5: Run tests and commit**

```bash
go test ./... 2>&1 | tail -5   # expected: PASS (logic/score unaffected)
git add banner.md main.go README.md
git commit -m "feat: --banner contract; embed banner.md ASCII art

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: Cross-repo verification & release refresh

- [ ] **Step 1: Full verification (launcher repo)**

```bash
cd /Volumes/T7/Stockpile/Workspace/github.com/justin06lee/shaw
go build ./... && go vet ./... && go test ./... 2>&1 | tail -10
```
Expected: all PASS.

- [ ] **Step 2: Live banner integration (local, real binary)**

```bash
H=$(mktemp -d)
SHAW_HOME="$H" SHAW_REGISTRY=https://raw.githubusercontent.com/justin06lee/hegale/master/index.json \
  go run ./cmd/shaw install snake.shaw   # installs the CURRENTLY-released binary
"$H/games/snake.shaw/snake.shaw" --banner   # NOTE: only prints art once a banner-enabled build is released (Task 7 Step 4)
```
Expected after release: prints the banner. Before release: `--banner` is an unknown flag on the old binary and the launcher would fall back to the default banner — acceptable.

- [ ] **Step 3: Push launcher and game (CONFIRM with user before pushing)**

```bash
cd /Volumes/T7/Stockpile/Workspace/github.com/justin06lee/shaw && git push origin HEAD:master
cd /Volumes/T7/Stockpile/Workspace/github.com/justin06lee/snake.shaw && git push origin HEAD:master
```

- [ ] **Step 4: Re-release snake.shaw so the installed binary supports `--banner`**

```bash
cd /Volumes/T7/Stockpile/Workspace/github.com/justin06lee/snake.shaw
rm -rf dist && mkdir dist
for t in darwin/arm64 darwin/amd64 linux/amd64 linux/arm64; do
  os=${t%/*}; arch=${t#*/}
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -o dist/snake.shaw-$os-$arch .
done
gh release upload v1.0.0 dist/* --repo justin06lee/snake.shaw --clobber
```
(Reuses v1.0.0 — the banner is additive, no API change. If you prefer a version bump, change `version` in main.go to 1.1.0, update hegale `index.json`, and `gh release create v1.1.0` instead.)

- [ ] **Step 5: End-to-end with banners (manual, TTY)**

```bash
H=$(mktemp -d)
SHAW_HOME="$H" go run ./cmd/shaw install snake.shaw   # from launcher repo
SHAW_HOME="$H" go -C /Volumes/T7/Stockpile/Workspace/github.com/justin06lee/shaw run ./cmd/shaw
```
Expected (human-verified): the Library grid shows the `snake` card with its banner art (not the default block); focus border is the accent color; sidebar nav + scrolling work; `enter` launches snake; `q` quits.

---

## Self-review notes

- **Spec coverage:** `--banner` contract (Tasks 2,6), default banner fallback (Task 2 + view renderCard), sidebar Library/Store/Kalama with coming-soon only in main pane (Tasks 3,4), compact grid + 2D vim/arrow nav + sidebar↔grid focus + scroll (Tasks 3,4), responsive columns (Task 4), theme in one styles file (Task 1), file split (all tasks), testing strategy (model/banner/view/integration tests across Tasks 2–5), error handling (loadBanners best-effort, empty-state, BinaryPath failure skip). Covered.
- **Type consistency:** `NewModel(games, banners)`, `Model.View/Update/Chosen/Quitting`, `columns`, `visibleRows`, `clampScroll`, `bannerFetch`, `fitBanner`, `bannerBlock`, `defaultBannerBlock`, `displayName`, `max1` are used consistently across tasks. `columns` is created in model.go (Task 3, real impl) then relocated to view.go (Task 4) — single definition at all times.
- **Ordering caveat:** Task 3 intentionally leaves `launcher.go` with a temporary `NewModel(games, nil)` call so the package compiles; Task 5 finalizes it. This is called out in both tasks.
- **Reversibility:** Tasks 1–6 are local commits. Task 7 Steps 3–4 push/release and are gated on user confirmation.
