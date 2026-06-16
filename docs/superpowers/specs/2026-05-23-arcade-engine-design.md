# shaw — Arcade Engine Design Spec

Date: 2026-05-23
Status: approved for planning
Scope: the **shaw engine** — this repo, gutted and replaced with the engine SDK

## Decision: shaw IS the engine

`shaw` stops being a typing trainer and becomes the **arcade engine itself** —
the foundation every game is built on. The current contents of this repo (the
typing-trainer app: `main.go`, `internal/*`) are removed and replaced with the
engine SDK. The trainer is not lost; it stays in git history and is reborn later
as a *game* built on shaw.

Games import the engine as the top-level package:

```go
import "github.com/justin06lee/shaw"

func main() {
    shaw.Run(myGame, shaw.Options{FPS: 30})
}
```

There is no separate engine repo. shaw is the engine.

## The arcade ecosystem

| Module | Repo | Role |
|--------|------|------|
| **shaw** | this repo | the engine SDK — canvas, loop, input, sprite, data. Games import it. |
| games | per-game repos | standalone binaries that import shaw (typing trainer reborn first, fighter later) |
| launcher | TBD (own repo, or folded into kalama) | scans installed games, draws a menu, exec's the pick, returns on exit |
| kalama | own repo | package manager: pull games from hegale, install/remove/list locally |
| hegale | own repo | registry: a JSON index + per-OS binaries |

**Execution model (decided): separate binaries.** Each game is a standalone
compiled program that imports shaw. The package manager installs prebuilt
binaries per OS into `~/.kalama/games/<game>/`. A launcher scans that directory
and `exec`s the chosen binary; the game owns the terminal while running and
returns control to the launcher on exit.

**Game ↔ launcher contract (decided): games own their data.** The launcher only
launches and returns — no score back-channel. Each game persists its own
scores/history under a standard data directory the engine provides. The launcher
and kalama never call each other; they meet at the filesystem.

**Note on the launcher:** it is no longer shaw (shaw is the engine now). Its home
is deferred — likely its own small repo or a subcommand of kalama. Not part of
this spec.

**Build order:** ship the shaw engine → rebuild the typing trainer as the first
game on it (proves the engine real) → launcher → kalama → hegale. A playable
local arcade exists before any networking.

This spec covers only the **shaw engine**. Each later module gets its own spec.

## Engine purpose

A Go package that lets a game be written as a small `Game` implementation while
shaw owns the terminal, a fixed-timestep loop, input state, a pixel canvas, and
per-game persistence. Built on `bubbletea` (reuses its terminal setup,
alt-screen, resize handling, and input decoding).

Both paradigms are served by the same engine:
- **Text games** (typing trainer) use loop + input + datadir, rendering with
  lipgloss as they like.
- **Action games** (fighter) additionally use the pixel canvas + sprites.

## Public API

Package name is `shaw`; callers reference `shaw.Canvas`, `shaw.Run`, etc.

```go
package shaw

// ---- Canvas: pixel buffer drawn with half-blocks (2 px per terminal cell) ----
type Color struct{ R, G, B, A uint8 } // A == 0 means transparent

type Canvas struct{ /* width, height, pixels */ }

func NewCanvas(w, h int) *Canvas          // h must be even; one cell row = 2 px rows
func (c *Canvas) Set(x, y int, col Color) // out-of-bounds is a no-op
func (c *Canvas) Clear(col Color)
func (c *Canvas) Blit(s *Sprite, x, y int) // draws sprite; skips transparent pixels
func (c *Canvas) Render() string           // ANSI truecolor string, one ▀ per cell
func (c *Canvas) Width() int
func (c *Canvas) Height() int

// ---- Sprite: a pixel grid loaded from PNG; alpha 0 = transparent ----
type Sprite struct{ /* width, height, pixels */ }

func LoadSprite(r io.Reader) (*Sprite, error) // decode PNG; games embed.FS their assets

// ---- Input ----
type Key string // normalized names: "left","right","up","down","a".."z","space","esc", etc.

type Input struct{ /* per-frame snapshot */ }

func (in Input) Held(k Key) bool     // key is down right now
func (in Input) Pressed(k Key) bool  // key went down during this frame
func (in Input) Released(k Key) bool // key went up during this frame

// ---- Loop ----
type Action int

const (
    Continue Action = iota
    Quit
)

type Game interface {
    Update(dt time.Duration, in Input) Action // advance one frame; return Quit to exit
    Draw(c *Canvas)                            // render current frame into the canvas
}

type Options struct {
    Width, Height int // pixel canvas size; 0 = auto-size from terminal (cols, rows*2)
    FPS           int // 0 = default 30
    Title        string
}

func Run(g Game, opts Options) error // blocks until Quit/Ctrl-C; restores terminal on return

// ---- Data ----
func DataDir(game string) (string, error) // $KALAMA_DATA_DIR/<game> or ~/.kalama/data/<game>; created
```

## How the pieces work

### Canvas (half-block rendering)

A `Canvas` is a `Width × Height` grid of `Color`, where `Height` is even. It
renders to terminal cells using the upper-half-block glyph `▀`:

- For cell at column `x`, cell-row `r`: top pixel = `(x, 2r)`, bottom pixel =
  `(x, 2r+1)`.
- Emit `▀` with **foreground** = top pixel color, **background** = bottom pixel
  color, using ANSI truecolor SGR (`\x1b[38;2;R;G;Bm\x1b[48;2;R;G;Bm▀`).
- This yields 2 vertical pixels per cell, near-square at typical font ratios. An
  80×24 terminal becomes an 80×48 pixel canvas.

Transparency applies only to sprites/blitting. The canvas itself always holds
concrete colors (cleared to some background). `Set` with an `A == 0` color is
treated as "no change" / skip, so `Blit` naturally skips transparent sprite
pixels.

`Render()` is a pure function of canvas state — unit-testable by asserting the
produced ANSI byte string for small canvases.

### Loop

`Run` wraps the game's `Game` implementation inside a bubbletea model:

1. Set up alt-screen + raw input via bubbletea.
2. Tick at `FPS` (default 30). On each tick, compute `dt`, build an `Input`
   snapshot from key events accumulated since the last frame, call
   `Update(dt, in)`.
3. If `Update` returns `Quit` (or Ctrl-C arrives), stop and restore the
   terminal, returning from `Run`. For an exec'd game this hands control back to
   the launcher.
4. `View` calls `Draw(canvas)` then `canvas.Render()`.

Sizing: when `Width`/`Height` are 0, the canvas auto-sizes to the terminal
(`width = cols`, `height = rows*2`) and resizes on `WindowSizeMsg`. The game's
`Draw` must tolerate a variable canvas size; a game wanting a fixed playfield
letterboxes itself.

### Input and the key-release problem

A terminal program reads a byte stream from stdin, not OS key events. A key press
delivers a character; **a key release delivers nothing**. Holding a key relies on
OS keyboard-repeat re-sending the character. Therefore release must be either
inferred or obtained from a terminal extension.

Two-tier strategy, behind the same `Held/Pressed/Released` API:

- **v1 — timeout-decay fallback (ships first, all terminals, no extra deps):** a
  key is considered `Held` for a decay window (~150 ms) after its most recent
  press/repeat event. `Pressed` is true on the frame a key first appears after
  being absent; `Released` is true on the frame the decay window expires. This is
  playable for directional movement; charge/hold-precise inputs feel mushy due to
  OS repeat delay and rate.
- **later — kitty keyboard protocol (deferred):** terminals such as
  kitty/ghostty/wezterm/foot can emit true press/release escape sequences.
  Obtained via bubbletea v2 or `charmbracelet/x/input`. Same public API, so games
  do not change. v1 does **not** depend on this.

This is the riskiest component. The implementation plan must include an early
input spike that confirms the fallback feels acceptable for movement before the
rest of the engine is finished.

### Sprite

`LoadSprite` decodes a PNG into a `Sprite` (a `Color` grid). Pixels with alpha 0
become transparent (`A == 0`); others carry their RGB with `A = 255`. Games embed
their PNG assets with `embed.FS` and load at startup. `Canvas.Blit` draws a
sprite at an offset, skipping transparent pixels and clipping at canvas edges.

### Data

`DataDir(game)` returns and creates `$KALAMA_DATA_DIR/<game>/`, falling back to
`~/.kalama/data/<game>/` when the env var is unset. This generalizes the
XDG-aware, corruption-tolerant persistence pattern the old trainer used in its
`history`/`config` packages; games store their own scores/history here.

## The manifest (contract, not engine code)

The launcher discovers games and `kalama` installs them by agreeing on a JSON
file the package manager writes alongside each installed binary:

```
~/.kalama/games/<game>/
  manifest.json
  <binary>
```

```json
{
  "name": "typing-trainer",
  "description": "monkeytype-style touch-typing trainer",
  "version": "1.0.0",
  "binary": "typing-trainer"
}
```

The launcher reads each `manifest.json` to build its menu and `exec`s
`<dir>/<binary>`. This schema is **documented here but is not part of the shaw
engine** — the launcher doesn't import shaw, only games do. The schema is owned
by the kalama/hegale specs; it appears here only to make the engine's place in
the system clear.

## v1 scope

**In:** `Canvas` (+ `Render`), `Sprite` (PNG load + `Blit`), `Run` loop,
`Input` with the timeout-decay fallback, `DataDir`.

**The repo migration is part of v1:** delete the existing trainer
(`main.go`, `internal/`, trainer-specific Makefile/README sections) and replace
the module's contents with the engine package. Keep the module path
`github.com/justin06lee/shaw`. The trainer remains recoverable from git history.

**Deferred (explicitly out of v1):**
- Sound.
- True key press/release via kitty protocol (fallback ships first).
- The launcher, games, kalama, hegale (separate modules / specs).

## Testing strategy

- **Canvas:** pure unit tests — construct a small canvas, `Set`/`Clear`/`Blit`,
  assert the exact ANSI string from `Render`. Cover even-height requirement,
  out-of-bounds no-ops, transparent-skip in `Blit`.
- **Sprite:** decode a tiny in-memory PNG, assert dimensions and that
  alpha-0 pixels become transparent.
- **Input:** drive the fallback with a synthetic, injectable clock (mirror the
  old `run` package's injectable `Now`) — feed press/repeat events and assert
  `Held/Pressed/Released` transitions across frames and across the decay
  boundary.
- **DataDir:** point `KALAMA_DATA_DIR` at a temp dir, assert path + creation.
- **Loop:** the `Game` interface is pure (no I/O); a fake `Game` can be ticked
  directly to assert `Update`/`Draw` ordering and `Quit` handling without a real
  terminal.

## Open questions (resolve during planning)

1. Exact decay window for the input fallback (start ~150 ms; tune in the spike).
2. Whether `Run` should expose raw terminal size to the game beyond canvas
   dimensions (probably not in v1).
3. Home of the launcher (own repo vs kalama subcommand) — out of scope here, but
   affects how the typing trainer is run during early development (likely a
   throwaway `go run` harness until the launcher exists).
```
