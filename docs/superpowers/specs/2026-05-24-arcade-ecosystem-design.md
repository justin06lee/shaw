# Arcade Ecosystem — Distribution Slice Design Spec

Date: 2026-05-24
Status: approved (autonomous — built overnight per explicit user instruction to "one-shot" the full slice)
Scope: a complete, working end-to-end arcade slice across four repos — engine, one real game, registry, package manager — proving install-and-play.

## Goal

Make the arcade real and usable: a user can run `kalama install luma`, have a real
game binary downloaded from the `hegale` registry onto their machine, and play it
with `kalama play`. This spec covers the first game, the registry format, the
package manager (incl. launcher), and the distribution/release flow.

## Repos involved

| Repo | Role | This spec |
|------|------|-----------|
| `shaw` | engine (built, v1 + smoke test merged) | unchanged, consumed by the game |
| `luma` (NEW) | first real game — Snake, built on shaw | full implementation |
| `hegale` | registry: `index.json` indexing per-OS game binaries | implement index + consume game releases |
| `kalama` | package manager + launcher | full implementation |

## Distribution model (decided)

- Each **game owns its binaries**: cross-compiled binaries are published as
  **GitHub Release assets on the game's own repo** (`luma` releases).
- `hegale` holds a single `index.json` (served raw via
  `https://raw.githubusercontent.com/justin06lee/hegale/master/index.json`) that
  lists games and points at those release-asset download URLs per OS/arch.
- `kalama install <game>` reads the index, selects the asset matching the host's
  `runtime.GOOS/GOARCH`, downloads it to `~/.kalama/games/<game>/<binary>`,
  sets the executable bit, and writes a `manifest.json` next to it.
- `kalama play [game]` is the launcher: with a game name it execs that binary;
  with no name it shows a bubbletea menu of installed games and execs the
  selection, returning to the menu on exit.
- `shaw` and `kalama` never import each other; games import only `shaw`.

### hegale `index.json` schema

```json
{
  "games": [
    {
      "name": "luma",
      "description": "Classic snake for the shaw terminal arcade",
      "version": "1.0.0",
      "binary": "luma",
      "assets": {
        "darwin/arm64": "https://github.com/justin06lee/luma/releases/download/v1.0.0/luma-darwin-arm64",
        "darwin/amd64": "https://github.com/justin06lee/luma/releases/download/v1.0.0/luma-darwin-amd64",
        "linux/amd64":  "https://github.com/justin06lee/luma/releases/download/v1.0.0/luma-linux-amd64",
        "linux/arm64":  "https://github.com/justin06lee/luma/releases/download/v1.0.0/luma-linux-arm64"
      }
    }
  ]
}
```

### installed layout + manifest

```
~/.kalama/games/<game>/
  manifest.json   {"name","description","version","binary"}
  <binary>        (executable)
```

`KALAMA_HOME` overrides `~/.kalama` (used by tests). `KALAMA_REGISTRY` overrides
the index URL (used by tests). `KALAMA_DATA_DIR` (already honored by shaw's
`DataDir`) is where games store save data: `~/.kalama/data/<game>/`.

## luma (the game)

Module `github.com/justin06lee/luma`, imports `github.com/justin06lee/shaw`.
Classic Snake, proving the whole engine surface: Canvas (Set/Clear), Input
(Pressed for turns), the loop (dt-accumulated fixed movement), and DataDir
(persistent high score).

**Architecture — pure logic vs rendering, like shaw's own `run`/`tui` split:**
- `game.go`: a pure `Game` state machine (no shaw imports): grid dims, snake body
  (slice of cells), direction, food cell, score, dead flag, an injectable RNG for
  food placement. Methods: `Turn(dir)` (ignores 180° reversals), `Step()`
  (advance one cell: move head, eat/grow + rescore + respawn food, or detect
  wall/self collision → dead), `Reset()`. Fully unit-testable.
- `render.go` + `main.go`: the shaw adapter — implements `shaw.Game`. `Update`
  reads `in.Pressed("up"/"down"/"left"/"right")` (and WASD) → `Turn`; accumulates
  `dt` and calls `Step()` every fixed interval (e.g. 120ms); on death waits for a
  key (`Pressed`) to `Reset`; `esc` → `Quit`. `Draw` maps the grid onto the
  canvas: clear to dark bg, draw snake cells and food as filled blocks sized to
  the canvas. Persists/loads the high score via `shaw.DataDir("luma")` as a
  corruption-tolerant JSON file (mirror shaw's history/config tolerance: a bad
  file is ignored, never crashes).
- `--version` flag: `luma --version` prints `luma <version>` and exits 0 WITHOUT
  needing a TTY. This is what kalama's end-to-end verification calls.

**Tests:** pure `game.go` logic — growth on eating, score increments, wall
collision death, self collision death, reversal guard, deterministic food via
seeded RNG; high-score load/save round-trip via a temp `KALAMA_DATA_DIR`.

## kalama (package manager + launcher)

Module `github.com/justin06lee/kalama`. `cmd/kalama/main.go` dispatches on
`os.Args[1]`. Internal packages with clear boundaries:
- `registry`: fetch + parse `index.json` (honors `KALAMA_REGISTRY`); look up a
  game; select the asset URL for `runtime.GOOS/GOARCH`; clear error when the host
  platform is unsupported.
- `store`: the local install dir (honors `KALAMA_HOME`); install (download a URL
  to `<home>/games/<game>/<binary>`, chmod 0755, write `manifest.json`), remove,
  list (read manifests). Download via `net/http` with a non-2xx → error.
- `launcher`: a bubbletea menu model over installed games (↑/↓ select, enter
  launch, q/esc quit) — pure model, unit-testable; and an `exec` that runs a
  game binary with inherited stdio so the game owns the terminal, returning on
  exit.

**Commands:**
```
kalama install <game>   # fetch index, download host asset, install + manifest
kalama remove  <game>   # delete ~/.kalama/games/<game>
kalama list             # installed games (name, version) from manifests
kalama play [game]      # exec a game, or menu over installed games
kalama --help
```

**Tests:**
- `registry`: `httptest` server serving a fixed `index.json`; assert game lookup
  and GOOS/GOARCH asset selection + unsupported-platform error.
- `store`: `httptest` server serving fake binary bytes; `KALAMA_HOME` = temp dir;
  assert install writes binary (exec bit set) + correct manifest; list returns it;
  remove deletes it.
- `launcher`: menu model selection/quit transitions (no real exec in tests).

## Release / distribution flow (executed during the build)

1. Build `luma` for `darwin/arm64`, `darwin/amd64`, `linux/amd64`, `linux/arm64`
   (`GOOS`/`GOARCH`, `CGO_ENABLED=0`). Name assets `luma-<os>-<arch>`.
2. `gh release create v1.0.0` on the `luma` repo, uploading the four binaries.
3. Write `hegale/index.json` pointing at those release URLs; push.
4. End-to-end verify on this host (darwin/arm64):
   `KALAMA_REGISTRY=<live hegale raw url> kalama install luma` →
   `~/.kalama/games/luma/luma --version` prints `luma 1.0.0` →
   `kalama list` shows it. This proves index→download→install→native-exec for real.

## Honest verification boundary

- **Fully verified on darwin/arm64:** game logic (unit tests), kalama (unit tests
  + a real install against the live hegale index), the downloaded binary is a
  working native executable (`--version`).
- **Built but not run here:** the linux + darwin/amd64 binaries (no such host
  available to execute them).
- **Not automatically verifiable (needs an interactive TTY + human):** the actual
  playing of luma and the `kalama play` menu rendering. The loop machinery is
  already covered by shaw's teatest smoke test; luma's interactive feel and the
  launcher menu are left for a human to eyeball.

## Deferred (still out of scope)

- The Street Fighter–style fighter (the north-star game; Snake proves the stack first).
- Multiple games (the pipeline supports N; we ship one excellent one).
- True key-up via kitty protocol; sound.
- `kalama update`, version pinning, checksums/signatures (note: a future hardening — verify asset integrity).
