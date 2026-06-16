# Shaw Launcher UI — Epic-Style Redesign Design Spec

Date: 2026-06-16
Status: approved (interactive)
Scope: redesign the `shaw` launcher's interactive UI from a plain text menu into a styled, Epic-Games-Launcher-inspired TUI using lipgloss — a left sidebar (Library / Store / Kalama) and a scrollable, vim-navigable grid of game cards with ASCII-art banners. Establish a `--banner` game contract for banner art.

## Motivation

`shaw` is the face of the arcade. Today its menu is a bare vertical text list. The user wants it to feel like the Epic Games Launcher: a persistent left sidebar for navigation and a main pane showing installed games as a grid of cards, each with a banner, packed compactly, navigable with both arrows and vim keys, and scrollable. "Store" and "Kalama" are placeholders for a future marketplace and engine/dev section.

## Non-goals

- No real raster image rendering (no sixel/kitty/half-block image decoding). Banners are ASCII/text art only.
- No actual Store/marketplace or Kalama/engine pages — those render a "coming soon" placeholder.
- No change to the package-manager commands (`install`/`remove`/`list`) or to the registry/install flow.
- The launcher still does NOT import the kalama engine.

## Banner art — the `--banner` game contract

Every `.shaw` game follows a tiny CLI contract (extends the existing `--version`):

- **`<binary> --version`** → prints `<name> <version>` and exits 0 (already implemented).
- **`<binary> --banner`** → prints the game's ASCII-art banner to stdout and exits 0. The art is authored as a `banner.md` file in the game's repo and embedded into the binary via `go:embed`.

The launcher obtains banner art by executing `<installed-binary> --banner` once per game (fast, no TTY), with a short timeout, and caching the result for the session. This keeps the banner versioned with and bundled inside the binary — zero changes to the registry or install flow.

**Banner canvas.** Banner art is authored to fit a card's inner art area of **15 columns × 7 rows** (the recommended canvas; documented in the game contract). The launcher fits art to this area:
- Lines longer than the inner width are clipped (truncated by display width, multibyte-safe).
- Art with more rows than the inner height is clipped to the top rows.
- Smaller art is centered horizontally and vertically within the area.

**Default banner (fallback).** If a game ships no banner, `--banner` is unsupported (exits non-zero / unknown flag), the call errors/times out, or output is empty, the launcher renders a **default banner**: the game's friendly name (`.shaw` trimmed) centered on a lipgloss gradient/accent block sized to the art area. The library is always fully usable without any game providing art.

`snake.shaw` ships a `banner.md` and implements `--banner` as the reference implementation.

## UI structure

### Sidebar (left, fixed width)

- A `SHAW` wordmark at top.
- Three nav items, in order: **Library**, **Store**, **Kalama**. No "coming soon" text next to the buttons.
- The active item is highlighted (accent color + `▸` marker); inactive items are soft grey.

### Main pane (right, fills remaining width)

Depends on the active section:

- **Library** — a header row (`Library` title + `N games` count) and a grid of game cards (below). Default section on launch.
- **Store** — centered placeholder: "Store — coming soon" with a one-line subtitle ("A marketplace for shaw games is on the way.").
- **Kalama** — centered placeholder: "Kalama — coming soon" with a one-line subtitle ("Build your own shaw games — engine tooling is on the way.").

### Game card

Each installed game is a bordered card:
- Banner art area (15×7) at top, holding fitted banner art or the default banner.
- The game's friendly name (`.shaw` trimmed) on the line below the card.
- The **focused** card has a bright accent border; unfocused cards have a dim border.
- Empty-library state: a friendly centered message ("No games installed — run `shaw install <game>`").

## Navigation & input

Two focus zones: the **sidebar** and the **content grid** (Library only).

- **Grid focus** (Library):
  - `l`/`→`, `h`/`←`, `j`/`↓`, `k`/`↑` move the focused card across the 2D grid.
  - Pressing `h`/`←` while in the leftmost column moves focus to the sidebar.
  - `enter` launches the focused game.
  - The grid scrolls vertically when it overflows the viewport; the focused card is always kept fully visible (scroll offset adjusts on move).
- **Sidebar focus:**
  - `j`/`↓`, `k`/`↑` move between Library / Store / Kalama.
  - `l`/`→`/`enter` activates the highlighted item: Library moves focus into the grid (if non-empty); Store/Kalama switch the main pane to their placeholder (focus stays in sidebar).
- **Global:** `q`, `esc`, `ctrl+c` quit the launcher without launching.

Layout is responsive: on `WindowSizeMsg` the model stores width/height; the number of grid columns is computed from the available content width (fixed card width + gap). A terminal too small to show one card renders a graceful minimal message.

## Theme

Dark, Epic-inspired:
- Background: near-black.
- Inactive text/borders: soft grey.
- Active text: bright white.
- One **accent color** (default: a vivid cyan-violet) for the selected nav item and the focused card border.

All colors and lipgloss styles live in one `styles.go` so the theme is tweakable in a single place.

## Architecture & files

All changes to the launcher live under `internal/launcher/` in the `shaw` repo. The single `launcher.go` is split by responsibility:

| File | Responsibility |
|------|----------------|
| `launcher.go` | Public API: `Play(name)` and `Exec(binPath)` orchestration (unchanged signatures). Builds the program with the games list and runs it; on exit, execs the chosen game. |
| `model.go` | The bubbletea `Model`: state (active section, sidebar cursor, grid cursor, scroll offset, terminal width/height, games, cached banners) + `Init`/`Update` (all key handling, focus transitions, scroll math). Pure and testable. |
| `view.go` | `View()` rendering: composes sidebar + main pane with lipgloss; computes columns and visible rows; assembles the grid. |
| `styles.go` | The palette and all lipgloss `Style` values (the theme). |
| `banner.go` | `bannerFor(game)` — exec `<binary> --banner` (behind an injectable function var for tests), cache, fit/clip to the art area, and the default-banner generator. |

`Model` exposes the same accessors used today (`Chosen()`, etc.) plus whatever the new tests need. lipgloss is promoted to a direct dependency in `go.mod`.

**Data dependencies.** The model is constructed from `[]store.Manifest` (already provides `Name`, `Description`, `Version`, `Binary`) and resolves each game's installed binary path via `store.BinaryPath(name)` for the `--banner` exec and for launching. No registry or manifest schema changes.

**Game side (`snake.shaw` repo).** Add `banner.md` (ASCII art sized to 15×7), embed it, and handle a `--banner` flag in `main.go` that prints the embedded art and exits 0. Document the `--version` + `--banner` contract in the README.

## Error handling

- `--banner` exec fails, times out, returns non-zero, or empty → default banner (no error surfaced to the user).
- `store.BinaryPath` fails for a game → that card shows the default banner and, if launched, surfaces the existing launch error.
- Empty library → friendly empty-state message in the Library pane.
- Terminal smaller than one card → minimal "window too small" message; resizing recovers.
- `enter` with no games / no focused card → no-op.

## Testing

- **Model (unit, pure):**
  - Sidebar navigation: `j`/`k` cycle items; `enter`/`l` on Library enters grid; on Store/Kalama switches section and stays in sidebar.
  - Grid navigation: `hjkl`/arrows move the cursor within bounds; `h` at column 0 returns focus to sidebar; `enter` sets the chosen game.
  - Scroll: moving below the visible viewport increases the scroll offset and keeps the focused card visible; moving up decreases it.
  - Column math: given a width and card/gap sizes, the computed column count is correct (including the 1-column and over-wide cases).
  - Quit keys set the quit flag and choose nothing.
- **Banner (unit):**
  - Fit/clip: long lines truncated to inner width; too-many-rows clipped; smaller art centered.
  - Default banner generated when the injected fetch returns error/empty; contains the friendly name.
  - Multibyte-safe width handling (no panic on wide runes).
- **Integration (teatest):** drive the full program through a window-size msg + key sequence (enter grid, move, switch to Store, back to Library, quit) and assert it renders without panic and the final chosen value is correct. Banner fetch is stubbed via the injectable function var so tests do not exec real binaries.
- **Game (`snake.shaw`):** `--banner` prints non-empty output and exits 0; `--version` unchanged.

## Out of scope (deferred)

- Returning to the launcher menu after a game exits. As today, `enter` sets the chosen game and quits the program; `Play` then execs the game and the `shaw` process ends when the game exits (control returns to the shell). Looping back into the launcher after a game quits is deferred.
- Real marketplace (Store) and engine/dev (Kalama) pages.
- Per-game metadata beyond what the manifest already carries (e.g., genre, art tags).
- Mouse support; search/filter; sorting.
- Animated or color banner art beyond what plain text + ANSI in `banner.md` already allows.
