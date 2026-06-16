# shaw Engine v1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the typing-trainer contents of this repo with `shaw`, a terminal arcade engine SDK: a half-block pixel canvas, a fixed-timestep game loop, input with a key-release fallback, PNG sprites, and a per-game data directory.

**Architecture:** `shaw` is the top-level Go package (`github.com/justin06lee/shaw`). Games import it and implement a small `Game` interface; `shaw.Run` owns the terminal (built on bubbletea), ticks at a fixed FPS, feeds an `Input` snapshot and `dt` into the game, and renders a `Canvas` of truecolor pixels using the upper-half-block glyph `▀` (2 vertical pixels per terminal cell).

**Tech Stack:** Go 1.26, `github.com/charmbracelet/bubbletea` (terminal/loop), standard library `image`/`image/png` (sprites). No new third-party deps.

**Convention:** every commit message ends with the trailer line `Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>` (omitted from the commands below for brevity — add it).

---

### Task 1: Gut the repo to a clean engine module

Remove the typing trainer; leave a compiling empty `shaw` package. The trainer stays recoverable in git history.

**Files:**
- Delete: `main.go`, `internal/` (whole tree), `shaw` (committed binary)
- Create: `doc.go`
- Modify: `Makefile`, `README.md`, `go.mod`

- [ ] **Step 1: Delete the trainer sources and the committed binary**

```bash
git rm -r internal
git rm main.go shaw
```

- [ ] **Step 2: Create the package doc file**

Create `doc.go`:

```go
// Package shaw is a terminal arcade engine: a half-block pixel canvas, a
// fixed-timestep game loop, input with a key-release fallback, PNG sprites,
// and per-game persistence. Games import this package and implement Game.
package shaw
```

- [ ] **Step 3: Replace the Makefile with engine targets**

Overwrite `Makefile`:

```makefile
.PHONY: test vet fmt tidy

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy
```

- [ ] **Step 4: Replace the README with an engine stub**

Overwrite `README.md`:

```markdown
# shaw

A terminal arcade engine. Games import `github.com/justin06lee/shaw` and
implement the `Game` interface; shaw owns the terminal, runs a fixed-timestep
loop, and renders a truecolor pixel canvas using half-block glyphs.

See `docs/superpowers/specs/2026-05-23-arcade-engine-design.md` for the design.

## Development

```bash
make test   # go test ./...
make vet    # go vet ./...
make tidy   # go mod tidy
```
```

- [ ] **Step 5: Verify the module compiles and is empty of the trainer**

Run: `go build ./... && go vet ./...`
Expected: no output, exit 0. `git status` shows `internal/`, `main.go`, `shaw` deleted.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "refactor!: gut typing trainer; shaw becomes the engine module

Remove the typing-trainer app (main.go, internal/, committed binary).
Repo now holds an empty package shaw, ready for the engine SDK. The
trainer remains recoverable in git history and is reborn later as a game."
```

---

### Task 2: Color and Canvas basics

**Files:**
- Create: `color.go`, `canvas.go`
- Test: `canvas_test.go`

- [ ] **Step 1: Write the failing test**

Create `canvas_test.go`:

```go
package shaw

import "testing"

func TestNewCanvasForcesEvenHeight(t *testing.T) {
	c := NewCanvas(4, 3)
	if c.Width() != 4 {
		t.Errorf("width = %d, want 4", c.Width())
	}
	if c.Height() != 4 {
		t.Errorf("height = %d, want 4 (rounded up to even)", c.Height())
	}
}

func TestSetSkipsTransparentAndOutOfBounds(t *testing.T) {
	c := NewCanvas(2, 2)
	red := Color{R: 255, A: 255}
	c.Set(0, 0, red)
	c.Set(0, 1, Color{R: 9, G: 9, B: 9}) // A==0 -> ignored
	c.Set(5, 5, red)                      // out of bounds -> ignored
	if got := c.at(0, 0); got != red {
		t.Errorf("at(0,0) = %+v, want %+v", got, red)
	}
	if got := c.at(0, 1); got != (Color{}) {
		t.Errorf("at(0,1) = %+v, want zero (transparent set ignored)", got)
	}
}

func TestClearFillsEveryPixel(t *testing.T) {
	c := NewCanvas(2, 2)
	blue := Color{B: 255, A: 255}
	c.Clear(blue)
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			if got := c.at(x, y); got != blue {
				t.Errorf("at(%d,%d) = %+v, want %+v", x, y, got, blue)
			}
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestNewCanvas -run TestSet -run TestClear ./...`
Expected: FAIL — `undefined: NewCanvas`, `undefined: Color`.

- [ ] **Step 3: Write Color**

Create `color.go`:

```go
package shaw

// Color is an RGBA color. A == 0 means fully transparent; the canvas renders
// transparent pixels as black, and Set ignores them so sprites can carry holes.
type Color struct {
	R, G, B, A uint8
}
```

- [ ] **Step 4: Write Canvas basics**

Create `canvas.go`:

```go
package shaw

// Canvas is a width x height grid of pixels. Height is always even: each
// terminal cell row holds two vertical pixels (see Render).
type Canvas struct {
	w, h   int
	pixels []Color
}

// NewCanvas allocates a w x h canvas. An odd h is rounded up to the next even
// number so it maps cleanly onto half-block cell rows.
func NewCanvas(w, h int) *Canvas {
	if h%2 != 0 {
		h++
	}
	return &Canvas{w: w, h: h, pixels: make([]Color, w*h)}
}

// Width returns the canvas width in pixels.
func (c *Canvas) Width() int { return c.w }

// Height returns the canvas height in pixels (even).
func (c *Canvas) Height() int { return c.h }

// Set paints one pixel. Transparent colors (A == 0) and out-of-bounds
// coordinates are ignored, so callers can blit freely without clipping.
func (c *Canvas) Set(x, y int, col Color) {
	if col.A == 0 || x < 0 || y < 0 || x >= c.w || y >= c.h {
		return
	}
	c.pixels[y*c.w+x] = col
}

// Clear fills every pixel with col, bypassing the transparency skip in Set.
func (c *Canvas) Clear(col Color) {
	for i := range c.pixels {
		c.pixels[i] = col
	}
}

// at returns the stored color at (x,y). Test helper; assumes in bounds.
func (c *Canvas) at(x, y int) Color { return c.pixels[y*c.w+x] }
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add color.go canvas.go canvas_test.go
git commit -m "feat: Color type and Canvas with Set/Clear/dimensions"
```

---

### Task 3: Canvas.Render (half-block ANSI)

**Files:**
- Modify: `canvas.go`
- Test: `canvas_test.go`

- [ ] **Step 1: Write the failing test**

Add to `canvas_test.go`:

```go
func TestRenderHalfBlockOneCell(t *testing.T) {
	c := NewCanvas(1, 2)
	c.Set(0, 0, Color{R: 255, A: 255}) // top pixel red
	c.Set(0, 1, Color{B: 255, A: 255}) // bottom pixel blue
	got := c.Render()
	want := "\x1b[38;2;255;0;0m\x1b[48;2;0;0;255m▀\x1b[0m"
	if got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestRenderTransparentPixelIsBlack(t *testing.T) {
	c := NewCanvas(1, 2) // nothing set -> both pixels transparent
	got := c.Render()
	want := "\x1b[38;2;0;0;0m\x1b[48;2;0;0;0m▀\x1b[0m"
	if got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestRenderTwoRowsSeparatedByNewline(t *testing.T) {
	c := NewCanvas(1, 4) // 2 cell rows
	got := c.Render()
	black := "\x1b[38;2;0;0;0m\x1b[48;2;0;0;0m▀"
	want := black + "\x1b[0m\n" + black + "\x1b[0m"
	if got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestRender ./...`
Expected: FAIL — `c.Render undefined`.

- [ ] **Step 3: Implement Render**

Add to `canvas.go` (add `"fmt"` and `"strings"` to imports):

```go
// Render draws the canvas as an ANSI truecolor string. Each terminal cell is
// the upper-half-block glyph ▀ with the top pixel as foreground and the bottom
// pixel as background, giving two vertical pixels per cell. Cell rows are joined
// by newlines with no trailing newline; each row ends with an SGR reset.
func (c *Canvas) Render() string {
	var b strings.Builder
	rows := c.h / 2
	for r := 0; r < rows; r++ {
		for x := 0; x < c.w; x++ {
			top := c.opaque(x, 2*r)
			bot := c.opaque(x, 2*r+1)
			fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀",
				top.R, top.G, top.B, bot.R, bot.G, bot.B)
		}
		b.WriteString("\x1b[0m")
		if r < rows-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// opaque returns the color at (x,y) for rendering: transparent pixels render as
// black so every cell has a concrete foreground and background color.
func (c *Canvas) opaque(x, y int) Color {
	px := c.pixels[y*c.w+x]
	if px.A == 0 {
		return Color{}
	}
	return px
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add canvas.go canvas_test.go
git commit -m "feat: Canvas.Render half-block truecolor output"
```

---

### Task 4: Sprite and LoadSprite (PNG)

**Files:**
- Create: `sprite.go`
- Test: `sprite_test.go`

- [ ] **Step 1: Write the failing test**

Create `sprite_test.go`:

```go
package shaw

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// makePNG builds a 2x1 PNG: pixel (0,0) opaque red, pixel (1,0) transparent.
func makePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.NRGBA{R: 255, A: 255})
	img.Set(1, 0, color.NRGBA{}) // A==0, transparent
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestLoadSpriteDimensionsAndTransparency(t *testing.T) {
	s, err := LoadSprite(bytes.NewReader(makePNG(t)))
	if err != nil {
		t.Fatalf("LoadSprite: %v", err)
	}
	if s.Width() != 2 || s.Height() != 1 {
		t.Fatalf("dims = %dx%d, want 2x1", s.Width(), s.Height())
	}
	if got := s.at(0, 0); got != (Color{R: 255, A: 255}) {
		t.Errorf("opaque pixel = %+v, want red A=255", got)
	}
	if got := s.at(1, 0); got.A != 0 {
		t.Errorf("transparent pixel A = %d, want 0", got.A)
	}
}

func TestLoadSpriteRejectsNonPNG(t *testing.T) {
	if _, err := LoadSprite(bytes.NewReader([]byte("not a png"))); err == nil {
		t.Fatal("expected error decoding non-PNG, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestLoadSprite ./...`
Expected: FAIL — `undefined: LoadSprite`.

- [ ] **Step 3: Implement Sprite and LoadSprite**

Create `sprite.go`:

```go
package shaw

import (
	"image/color"
	"image/png"
	"io"
)

// Sprite is a fixed pixel grid loaded from a PNG. Pixels with alpha 0 are
// transparent and are skipped when blitted onto a canvas.
type Sprite struct {
	w, h   int
	pixels []Color
}

// Width returns the sprite width in pixels.
func (s *Sprite) Width() int { return s.w }

// Height returns the sprite height in pixels.
func (s *Sprite) Height() int { return s.h }

// at returns the color at (x,y). Test helper; assumes in bounds.
func (s *Sprite) at(x, y int) Color { return s.pixels[y*s.w+x] }

// LoadSprite decodes a PNG into a Sprite. Fully transparent source pixels
// (alpha 0) become transparent Colors; all others become opaque (A = 255).
func LoadSprite(r io.Reader) (*Sprite, error) {
	img, err := png.Decode(r)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	pixels := make([]Color, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			n := color.NRGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			if n.A == 0 {
				pixels[y*w+x] = Color{} // transparent
				continue
			}
			pixels[y*w+x] = Color{R: n.R, G: n.G, B: n.B, A: 255}
		}
	}
	return &Sprite{w: w, h: h, pixels: pixels}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add sprite.go sprite_test.go
git commit -m "feat: Sprite type and LoadSprite PNG decoder"
```

---

### Task 5: Canvas.Blit

**Files:**
- Modify: `canvas.go`
- Test: `canvas_test.go`

- [ ] **Step 1: Write the failing test**

Add to `canvas_test.go` (add `"bytes"` to the test imports if not present — it is needed here):

```go
func TestBlitSkipsTransparentAndClips(t *testing.T) {
	s, err := LoadSprite(bytes.NewReader(makePNG(t))) // 2x1: red, transparent
	if err != nil {
		t.Fatalf("LoadSprite: %v", err)
	}
	c := NewCanvas(2, 2)
	c.Clear(Color{B: 255, A: 255}) // blue background
	c.Blit(s, 0, 0)
	if got := c.at(0, 0); got != (Color{R: 255, A: 255}) {
		t.Errorf("at(0,0) = %+v, want red (opaque sprite pixel)", got)
	}
	if got := c.at(1, 0); got != (Color{B: 255, A: 255}) {
		t.Errorf("at(1,0) = %+v, want blue (transparent sprite pixel skipped)", got)
	}
	c.Blit(s, 5, 5) // fully off-canvas: must not panic
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestBlit ./...`
Expected: FAIL — `c.Blit undefined`.

- [ ] **Step 3: Implement Blit**

Add to `canvas.go`:

```go
// Blit draws sprite s with its top-left corner at (x,y). Transparent sprite
// pixels are skipped and pixels outside the canvas are clipped, both via Set.
func (c *Canvas) Blit(s *Sprite, x, y int) {
	for sy := 0; sy < s.h; sy++ {
		for sx := 0; sx < s.w; sx++ {
			c.Set(x+sx, y+sy, s.pixels[sy*s.w+sx])
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add canvas.go canvas_test.go
git commit -m "feat: Canvas.Blit with transparency skip and clipping"
```

---

### Task 6: Input fallback tracker

The tracker is pure and timed by an injectable `time.Duration` (elapsed since run start), so it is deterministically testable. The loop (Task 7) feeds it real elapsed times.

**Files:**
- Create: `input.go`
- Test: `input_test.go`

- [ ] **Step 1: Write the failing test**

Create `input_test.go`:

```go
package shaw

import (
	"testing"
	"time"
)

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

func TestPressMakesKeyHeldAndPressed(t *testing.T) {
	tr := newInputTracker()
	tr.press("a", ms(0))
	in := tr.snapshot(ms(10))
	if !in.Held("a") {
		t.Error("Held(a) = false, want true within decay window")
	}
	if !in.Pressed("a") {
		t.Error("Pressed(a) = false, want true on first appearance")
	}
	if in.Released("a") {
		t.Error("Released(a) = true, want false")
	}
}

func TestPressedOnlyOnFirstFrame(t *testing.T) {
	tr := newInputTracker()
	tr.press("a", ms(0))
	_ = tr.snapshot(ms(10)) // first frame: pressed
	tr.press("a", ms(20))   // key-repeat keeps it alive
	in := tr.snapshot(ms(30))
	if !in.Held("a") {
		t.Error("Held(a) = false, want true")
	}
	if in.Pressed("a") {
		t.Error("Pressed(a) = true, want false on second held frame")
	}
}

func TestReleasedFiresOnceAfterDecay(t *testing.T) {
	tr := newInputTracker()
	tr.press("a", ms(0))
	_ = tr.snapshot(ms(10)) // held
	in := tr.snapshot(ms(300)) // far past decay window -> released
	if in.Held("a") {
		t.Error("Held(a) = true, want false after decay")
	}
	if !in.Released("a") {
		t.Error("Released(a) = false, want true on the frame it decays")
	}
	in2 := tr.snapshot(ms(310))
	if in2.Released("a") {
		t.Error("Released(a) = true again, want false (fires once)")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run "TestPress|TestReleased" ./...`
Expected: FAIL — `undefined: newInputTracker`.

- [ ] **Step 3: Implement input**

Create `input.go`:

```go
package shaw

import "time"

// Key is a normalized key name, e.g. "left", "a", "space", "esc". It matches
// bubbletea's KeyMsg.String() values.
type Key string

// decayWindow is how long after a key's last press/repeat event the key is
// still considered held. OS key-repeat re-sends a held key faster than this, so
// a genuinely held key stays Held; once repeats stop, it decays to released.
const decayWindow = 150 * time.Millisecond

// Input is an immutable per-frame snapshot of key state.
type Input struct {
	held     map[Key]bool
	pressed  map[Key]bool
	released map[Key]bool
}

// Held reports whether k is down this frame.
func (in Input) Held(k Key) bool { return in.held[k] }

// Pressed reports whether k went down on this frame (was up last frame).
func (in Input) Pressed(k Key) bool { return in.pressed[k] }

// Released reports whether k went up on this frame (was down last frame).
func (in Input) Released(k Key) bool { return in.released[k] }

// inputTracker accumulates key events and produces Input snapshots. Times are
// durations since run start, supplied by the loop (or tests).
type inputTracker struct {
	lastSeen map[Key]time.Duration
	prevHeld map[Key]bool
}

func newInputTracker() *inputTracker {
	return &inputTracker{
		lastSeen: map[Key]time.Duration{},
		prevHeld: map[Key]bool{},
	}
}

// press records that key k produced an event (initial press or OS repeat) at
// elapsed time at.
func (t *inputTracker) press(k Key, at time.Duration) {
	t.lastSeen[k] = at
}

// snapshot computes the Input for the current frame at elapsed time now, then
// records the held state for next-frame edge detection.
func (t *inputTracker) snapshot(now time.Duration) Input {
	held := map[Key]bool{}
	pressed := map[Key]bool{}
	released := map[Key]bool{}
	for k, last := range t.lastSeen {
		isHeld := now-last <= decayWindow
		if isHeld {
			held[k] = true
			if !t.prevHeld[k] {
				pressed[k] = true
			}
		} else if t.prevHeld[k] {
			released[k] = true
		}
	}
	t.prevHeld = held
	return Input{held: held, pressed: pressed, released: released}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add input.go input_test.go
git commit -m "feat: input tracker with key-release decay fallback"
```

---

### Task 7: Game loop and Run

The bubbletea model is exercised directly (no real terminal) with a fake `Game`.

**Files:**
- Create: `loop.go`
- Test: `loop_test.go`

- [ ] **Step 1: Write the failing test**

Create `loop_test.go`:

```go
package shaw

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// fakeGame records Update calls and returns a configurable Action.
type fakeGame struct {
	updates int
	lastIn  Input
	action  Action
	drawn   bool
}

func (g *fakeGame) Update(dt time.Duration, in Input) Action {
	g.updates++
	g.lastIn = in
	return g.action
}
func (g *fakeGame) Draw(c *Canvas) { g.drawn = true }

func TestFrameAdvancesGameAndContinues(t *testing.T) {
	g := &fakeGame{action: Continue}
	m := newModel(g, Options{Width: 4, Height: 4})
	next, cmd := m.Update(frameMsg(time.Now()))
	m = next.(*model)
	if g.updates != 1 {
		t.Errorf("Update calls = %d, want 1", g.updates)
	}
	if m.quit {
		t.Error("quit = true, want false on Continue")
	}
	if cmd == nil {
		t.Error("cmd = nil, want a follow-up tick command on Continue")
	}
}

func TestFrameQuitStopsLoop(t *testing.T) {
	g := &fakeGame{action: Quit}
	m := newModel(g, Options{Width: 4, Height: 4})
	next, _ := m.Update(frameMsg(time.Now()))
	if !next.(*model).quit {
		t.Error("quit = false, want true when game returns Quit")
	}
}

func TestCtrlCQuits(t *testing.T) {
	g := &fakeGame{action: Continue}
	m := newModel(g, Options{Width: 4, Height: 4})
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !next.(*model).quit {
		t.Error("quit = false, want true on Ctrl+C")
	}
}

func TestKeyEventBecomesHeldNextFrame(t *testing.T) {
	g := &fakeGame{action: Continue}
	m := newModel(g, Options{Width: 4, Height: 4})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m.Update(frameMsg(time.Now()))
	if !g.lastIn.Held("a") {
		t.Error(`Held("a") = false, want true after key event then frame`)
	}
}

func TestViewRendersCanvas(t *testing.T) {
	g := &fakeGame{action: Continue}
	m := newModel(g, Options{Width: 1, Height: 2})
	out := m.View()
	if !g.drawn {
		t.Error("game.Draw was not called by View")
	}
	if out == "" {
		t.Error("View returned empty string")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run "TestFrame|TestCtrlC|TestKeyEvent|TestView" ./...`
Expected: FAIL — `undefined: newModel`, `undefined: frameMsg`, `undefined: Options`.

- [ ] **Step 3: Implement the loop**

Create `loop.go`:

```go
package shaw

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Action is what a Game asks the loop to do after a frame.
type Action int

const (
	Continue Action = iota // keep running
	Quit                   // stop the loop and restore the terminal
)

// Game is the interface a shaw game implements. Update advances one frame given
// the time since the previous frame and the current input; Draw paints the
// frame into the canvas.
type Game interface {
	Update(dt time.Duration, in Input) Action
	Draw(c *Canvas)
}

// Options configures Run. Zero values mean: auto-size the canvas to the terminal
// (Width and Height both 0), and default to 30 FPS.
type Options struct {
	Width, Height int
	FPS           int
	Title         string
}

const defaultFPS = 30

// frameMsg is the per-frame tick.
type frameMsg time.Time

type model struct {
	game    Game
	canvas  *Canvas
	tracker *inputTracker
	fps     int
	auto    bool
	start   time.Time
	last    time.Duration
	quit    bool
}

func newModel(g Game, opts Options) *model {
	fps := opts.FPS
	if fps <= 0 {
		fps = defaultFPS
	}
	auto := opts.Width == 0 && opts.Height == 0
	w, h := opts.Width, opts.Height
	if auto {
		w, h = 80, 48
	}
	return &model{
		game:    g,
		canvas:  NewCanvas(w, h),
		tracker: newInputTracker(),
		fps:     fps,
		auto:    auto,
		start:   time.Now(),
	}
}

func (m *model) elapsed() time.Duration { return time.Since(m.start) }

func (m *model) tick() tea.Cmd {
	d := time.Second / time.Duration(m.fps)
	return tea.Tick(d, func(t time.Time) tea.Msg { return frameMsg(t) })
}

func (m *model) Init() tea.Cmd { return m.tick() }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.quit = true
			return m, tea.Quit
		}
		m.tracker.press(Key(msg.String()), m.elapsed())
		return m, nil
	case tea.WindowSizeMsg:
		if m.auto {
			m.canvas = NewCanvas(msg.Width, msg.Height*2)
		}
		return m, nil
	case frameMsg:
		now := m.elapsed()
		in := m.tracker.snapshot(now)
		dt := now - m.last
		m.last = now
		if m.game.Update(dt, in) == Quit {
			m.quit = true
			return m, tea.Quit
		}
		return m, m.tick()
	}
	return m, nil
}

func (m *model) View() string {
	m.game.Draw(m.canvas)
	return m.canvas.Render()
}

// Run starts the game loop in the alternate screen and blocks until the game
// returns Quit or the user presses Ctrl+C, then restores the terminal.
func Run(g Game, opts Options) error {
	_, err := tea.NewProgram(newModel(g, opts), tea.WithAltScreen()).Run()
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add loop.go loop_test.go
git commit -m "feat: fixed-timestep game loop and Run"
```

---

### Task 8: DataDir

**Files:**
- Create: `data.go`
- Test: `data_test.go`

- [ ] **Step 1: Write the failing test**

Create `data_test.go`:

```go
package shaw

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataDirUsesEnvAndCreates(t *testing.T) {
	base := t.TempDir()
	t.Setenv("KALAMA_DATA_DIR", base)
	dir, err := DataDir("fighter")
	if err != nil {
		t.Fatalf("DataDir: %v", err)
	}
	want := filepath.Join(base, "fighter")
	if dir != want {
		t.Errorf("dir = %q, want %q", dir, want)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Errorf("expected created directory at %q, stat err = %v", dir, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestDataDir ./...`
Expected: FAIL — `undefined: DataDir`.

- [ ] **Step 3: Implement DataDir**

Create `data.go`:

```go
package shaw

import (
	"os"
	"path/filepath"
)

// DataDir returns and creates a per-game persistence directory. It uses
// $KALAMA_DATA_DIR/<game> when the env var is set, otherwise
// ~/.kalama/data/<game>. Games store their own scores/history here.
func DataDir(game string) (string, error) {
	base := os.Getenv("KALAMA_DATA_DIR")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".kalama", "data")
	}
	dir := filepath.Join(base, game)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add data.go data_test.go
git commit -m "feat: per-game DataDir helper"
```

---

### Task 9: Tidy dependencies and smoke-test the input feel

The trainer used lipgloss; the engine does not. Tidy removes it. Then a throwaway demo confirms the loop renders and the input fallback feels acceptable for movement (the spec's required input check).

**Files:**
- Modify: `go.mod`, `go.sum`
- Create (temporary, NOT committed): `cmd/demo/main.go`

- [ ] **Step 1: Tidy modules**

Run: `go mod tidy`
Expected: `lipgloss` and now-unused indirect deps drop out of `go.mod`/`go.sum`. `bubbletea` and its indirect deps remain.

- [ ] **Step 2: Verify full build, vet, and tests**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: all pass, no output from build/vet.

- [ ] **Step 3: Write a throwaway demo (not committed)**

Create `cmd/demo/main.go`:

```go
package main

import (
	"time"

	shaw "github.com/justin06lee/shaw"
)

// box is a 6x6 sprite-free moving square driven by held arrow keys.
type box struct{ x, y int }

func (b *box) Update(dt time.Duration, in shaw.Input) shaw.Action {
	if in.Pressed("esc") {
		return shaw.Quit
	}
	if in.Held("left") {
		b.x -= 2
	}
	if in.Held("right") {
		b.x += 2
	}
	if in.Held("up") {
		b.y -= 2
	}
	if in.Held("down") {
		b.y += 2
	}
	return shaw.Continue
}

func (b *box) Draw(c *shaw.Canvas) {
	c.Clear(shaw.Color{R: 20, G: 20, B: 30, A: 255})
	for dy := 0; dy < 6; dy++ {
		for dx := 0; dx < 6; dx++ {
			c.Set(b.x+dx, b.y+dy, shaw.Color{R: 255, G: 220, A: 255})
		}
	}
}

func main() {
	_ = shaw.Run(&box{x: 20, y: 20}, shaw.Options{FPS: 30})
}
```

- [ ] **Step 4: Run the demo and confirm feel**

Run: `go run ./cmd/demo`
Manual checks:
- A yellow square renders on a dark background.
- Holding an arrow key moves the square continuously (after the brief OS key-repeat delay).
- `esc` exits cleanly and the terminal is restored.

If movement is unusable (not merely "starts after a short delay"), note it — the kitty-protocol path (deferred) is the fix, but the fallback should be good enough for movement.

- [ ] **Step 5: Remove the demo**

```bash
rm -rf cmd
```

- [ ] **Step 6: Commit the tidy**

```bash
git add go.mod go.sum
git commit -m "chore: tidy deps to engine-only set"
```

---

## Self-Review

**Spec coverage:**
- Canvas + half-block Render → Tasks 2, 3 ✓
- Sprite PNG load → Task 4 ✓; Blit → Task 5 ✓
- Input Held/Pressed/Released with decay fallback → Task 6 ✓
- Loop / Game / Run with FPS + auto-size → Task 7 ✓
- DataDir (KALAMA_DATA_DIR + ~/.kalama fallback) → Task 8 ✓
- Repo migration (gut trainer, package shaw) → Task 1 ✓
- Input spike (manual feel check) → Task 9 ✓
- Deferred items (sound, kitty protocol, launcher/kalama/hegale) → correctly absent ✓
- Manifest is documentation-only in the spec → correctly no task ✓

**Type consistency:** `Color`, `Canvas`, `Sprite`, `Key`, `Input`, `Game`, `Action`, `Options`, `Run`, `DataDir` match the spec's API and are used identically across tasks. `newInputTracker`/`press`/`snapshot` and `newModel`/`frameMsg` are internal and consistent between their defining task and the loop test.

**Placeholder scan:** no TBD/TODO; every code step is complete. The one conditional instruction (drop the `image` import if vet flags it) is explicit, not a placeholder.
```
