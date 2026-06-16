# Arcade Rename & Role Swap — Design Spec

Date: 2026-06-15
Status: approved (interactive)
Scope: swap the roles of the `shaw` and `kalama` repos, rename the first game `luma` → `snake.shaw`, establish `<name>.shaw` as the game-naming standard, and re-brand all user-facing paths/env to `shaw`. Update every README and design doc across all repos.

## Motivation

`shaw` is the name a player should type. Today `shaw` is the engine library (never typed by players) and `kalama` is the launcher/package-manager CLI (typed constantly). That's backwards. We swap them so the one command a player runs is `shaw`, and the engine that game developers import becomes `kalama`.

The mental model after this change:

- **`shaw`** — the launcher / arcade. The only command players type. `shaw` → game menu; `shaw install <game>`, `shaw list`, etc.
- **`kalama`** — the underlying engine library. Game *developers* import it. Players never type it.
- **`hegale`** — the registry (`index.json` catalog) `shaw` reads to install games. Plumbing.
- **games** — `*.shaw` binaries built on the `kalama` engine. `snake.shaw` is the first.

The brand boundary: everything *player-facing* is "shaw" (the command, `~/.shaw`, `SHAW_*` env, the `.shaw` game suffix). "kalama" is only the engine library name that game *developers* import. A `.shaw` game importing a package called `kalama` is expected and accepted.

## Target end state

### `shaw` (the launcher — was the `kalama` repo's code)

- Module `github.com/justin06lee/shaw`; binary **`shaw`**.
- **`shaw` with no args opens the selection menu** (today `kalama` with no args prints usage — this is the headline behavior change).
- Subcommands retained, renamed in help/output to `shaw`:
  - `shaw install <id>` — fetch from hegale, install to `~/.shaw/games/<id>/`
  - `shaw remove <id>`
  - `shaw list`
  - `shaw play [id]` — exec a game directly, or show the menu (same as bare `shaw`)
  - `shaw help`
- Install tree: `~/.shaw/games/<id>/` with `manifest.json` + binary.
- Env: `SHAW_HOME` (default `~/.shaw`), `SHAW_REGISTRY` (default hegale raw URL).
- Dependencies: bubbletea only (unchanged). Does **not** import the engine.

### `kalama` (the engine — was the `shaw` repo's code)

- Module `github.com/justin06lee/kalama`; Go package renamed `shaw` → **`kalama`**.
- Public API unchanged in shape, re-qualified: `kalama.Run`, `kalama.Canvas`, `kalama.Color`, `kalama.Sprite`, `kalama.LoadSprite`, `kalama.Input`, `kalama.Game`, `kalama.Action`, `kalama.Continue`, `kalama.Quit`, `kalama.Options`, `kalama.DataDir`.
- `DataDir(game)` writes to `~/.shaw/data/<game>/`; env `SHAW_DATA_DIR` (default `~/.shaw/data`).
- Dependencies: bubbletea + teatest (unchanged).

### `snake.shaw` (the game — was the `luma` repo)

- Repo + module `github.com/justin06lee/snake.shaw`; binary **`snake.shaw`**.
- Registry id `snake.shaw`; the launcher menu displays the friendly name **"snake"** (derived by trimming the `.shaw` suffix).
- Imports the `kalama` engine; uses `kalama.*` throughout.
- High score under `~/.shaw/data/snake.shaw/highscore.json` via `kalama.DataDir("snake.shaw")`.
- `--version` prints `snake.shaw 1.0.0` (no TTY needed — this is what `shaw` install-verification calls).
- Release assets named `snake.shaw-<os>-<arch>` for `darwin/arm64`, `darwin/amd64`, `linux/amd64`, `linux/arm64`.

### `hegale` (the registry — shape unchanged)

- `index.json`: the `luma` entry becomes `snake.shaw`:
  ```json
  {
    "games": [
      {
        "name": "snake.shaw",
        "description": "Classic snake for the shaw terminal arcade",
        "version": "1.0.0",
        "binary": "snake.shaw",
        "assets": {
          "darwin/arm64": "https://github.com/justin06lee/snake.shaw/releases/download/v1.0.0/snake.shaw-darwin-arm64",
          "darwin/amd64": "https://github.com/justin06lee/snake.shaw/releases/download/v1.0.0/snake.shaw-darwin-amd64",
          "linux/amd64":  "https://github.com/justin06lee/snake.shaw/releases/download/v1.0.0/snake.shaw-linux-amd64",
          "linux/arm64":  "https://github.com/justin06lee/snake.shaw/releases/download/v1.0.0/snake.shaw-linux-arm64"
        }
      }
    ]
  }
  ```
- README reworded: shaw = launcher/package-manager, kalama = engine. The menu shows the friendly name; the install id keeps the `.shaw` suffix.

### Resulting dependency graph

```
shaw (launcher)   → bubbletea
kalama (engine)   → bubbletea, teatest
snake.shaw (game) → kalama
hegale            → (data only)
```

No cycles. The launcher never imports the engine; the game imports only the engine.

## Game-naming standard (`<name>.shaw`)

Every shaw game uses the `.shaw` suffix consistently across: GitHub repo, Go module path, compiled binary, and registry id. The launcher derives the display name by trimming the trailing `.shaw`. This makes "this is a shaw game" unambiguous everywhere a name appears, while players still see clean names ("snake") in the menu.

## Migration mechanism

The swap is a content/remote swap plus module-path edits; force-push is acceptable (no other collaborators).

1. **Engine → `kalama`**: in the current `shaw` working tree, set module to `github.com/justin06lee/kalama`, rename package `shaw` → `kalama`, update `doc.go`/tests/internal references, repoint `DataDir` to `~/.shaw/data` + `SHAW_DATA_DIR`, update README + `docs/`. Point the remote at `github.com/justin06lee/kalama` and force-push.
2. **Launcher → `shaw`**: in the current `kalama` working tree, set module to `github.com/justin06lee/shaw`, rename `cmd/kalama` → `cmd/shaw`, change bare-invocation to open the menu, rebrand `KALAMA_*` → `SHAW_*` and `~/.kalama` → `~/.shaw`, update strings/help/README. Point the remote at `github.com/justin06lee/shaw` and force-push.
3. **Game → `snake.shaw`**: in the `luma` working tree, set module to `github.com/justin06lee/snake.shaw`, switch imports to `kalama`, rename binary/version string/`--version` output, update `.gitignore` + README. Create the `snake.shaw` GitHub repo and push.
4. **Registry**: update `hegale/index.json` + README; push.
5. **Local dirs**: rename working directories to match new roles so `../shaw` holds the launcher and `../kalama` holds the engine.
6. **Release & verify**: cross-compile `snake.shaw` for the four targets, `gh release create v1.0.0` on the `snake.shaw` repo with the assets, then end-to-end verify on darwin/arm64: `SHAW_REGISTRY=<live> shaw install snake.shaw` → `~/.shaw/games/snake.shaw/snake.shaw --version` prints `snake.shaw 1.0.0` → `shaw list` shows it.

## Testing & verification

- Each repo's existing unit tests pass after the rename (engine canvas/sprite/input/loop/data; launcher registry/store/launcher; game logic/score).
- `go vet` clean; `go build` clean in all three Go modules.
- The engine smoke test (teatest) still drives the loop without panics.
- Install-and-play happy path verified live on darwin/arm64 as above. linux + darwin/amd64 binaries are built but not run (honest boundary, same as the prior slice).
- `shaw` with no args is confirmed to open the menu (and shows the empty-state hint when no games are installed).

## Honest verification boundary

- **Fully verified on darwin/arm64**: all unit tests, `go vet`/`build`, real `shaw install` against the live hegale index, downloaded binary runs (`--version`), `shaw list`.
- **Built but not run**: linux + darwin/amd64 binaries.
- **Not auto-verifiable**: interactive play and the menu UI (need a TTY + human).

## Out of scope (unchanged from prior slices)

- The fighter / additional games (pipeline supports N; shipping one).
- True key-up via the kitty keyboard protocol; sound.
- `shaw update`, version pinning, checksums/signatures.
- Preserving old git history on the swapped repos (force-push clobbers it, per decision).
