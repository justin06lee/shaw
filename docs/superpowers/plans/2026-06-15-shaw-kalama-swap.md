# Shaw/Kalama Role Swap — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Swap the roles of the `shaw` and `kalama` repos (shaw → launcher you type; kalama → engine library you import), rename `luma` → `snake.shaw`, establish the `<name>.shaw` game-naming standard, and re-brand all user-facing paths/env to `shaw`.

**Architecture:** A content/remote swap. The engine code (currently the `shaw` repo) becomes module `github.com/justin06lee/kalama`; the launcher/package-manager code (currently the `kalama` repo) becomes module `github.com/justin06lee/shaw` with a binary `shaw` whose no-arg invocation opens the game menu. The game imports the new `kalama` engine. Phase A is all local (rename, build, test, commit). Phase B is the outward cutover (force-pushes, new repo, release) and is confirmed with the user before running.

**Tech Stack:** Go 1.26.2, charmbracelet/bubbletea, teatest, `gh` CLI, GitHub Releases.

**Working dirs at start** (siblings under `/Volumes/T7/Stockpile/Workspace/github.com/justin06lee/`):
- `shaw/` — engine code → becomes module `kalama`, pushed to github `kalama`, local dir renamed to `kalama/`
- `kalama/` — launcher code → becomes module `shaw`, pushed to github `shaw`, local dir renamed to `shaw/`
- `luma/` — game → becomes module `snake.shaw`, new github repo, local dir renamed to `snake.shaw/`
- `hegale/` — registry, unchanged location

**Convention:** all commits end with the trailer:
```
Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>
```

---

## Phase A — Local rename, build, test, commit (no network)

### Task 1: Engine → module `kalama`, package `kalama`, `SHAW_*` data dir

**Files (all in `shaw/`):**
- Modify: `go.mod`, `doc.go`, `data.go`, `README.md`
- Modify (package line only): `canvas.go`, `color.go`, `input.go`, `loop.go`, `sprite.go`, `canvas_test.go`, `data_test.go`, `input_test.go`, `loop_test.go`, `loop_smoke_test.go`

- [ ] **Step 1: Rename the Go package in every file**

In all 12 `.go` files listed above, change the first line `package shaw` → `package kalama`.
Run (from `shaw/`):
```bash
cd /Volumes/T7/Stockpile/Workspace/github.com/justin06lee/shaw
sed -i '' 's/^package shaw$/package kalama/' *.go
grep -L '^package kalama' *.go   # expected: no output
```

- [ ] **Step 2: Set the module path**

Edit `go.mod` line 1: `module github.com/justin06lee/shaw` → `module github.com/justin06lee/kalama`.
```bash
go mod edit -module github.com/justin06lee/kalama
head -1 go.mod   # expected: module github.com/justin06lee/kalama
```

- [ ] **Step 3: Re-brand `DataDir` to `SHAW_*` / `~/.shaw/data`**

In `data.go`, replace the doc comment and body env/path:
- Comment: `$KALAMA_DATA_DIR/<game>` → `$SHAW_DATA_DIR/<game>`, `~/.kalama/data/<game>` → `~/.shaw/data/<game>`.
- `os.Getenv("KALAMA_DATA_DIR")` → `os.Getenv("SHAW_DATA_DIR")`.
- `filepath.Join(home, ".kalama", "data")` → `filepath.Join(home, ".shaw", "data")`.
```bash
sed -i '' -e 's/KALAMA_DATA_DIR/SHAW_DATA_DIR/g' -e 's/\.kalama/.shaw/g' data.go
grep -n 'SHAW_DATA_DIR\|".shaw"' data.go   # expected: env line + Join line
```

- [ ] **Step 4: Update `doc.go` package comment**

In `doc.go`, change `// Package shaw is a terminal arcade engine:` → `// Package kalama is a terminal arcade engine:` (keep the rest of the comment text).
```bash
sed -i '' 's/Package shaw is/Package kalama is/' doc.go
```

- [ ] **Step 5: Rewrite `README.md`** to describe the engine library

Replace `README.md` contents with:
```markdown
# kalama

The engine library behind the [shaw](https://github.com/justin06lee/shaw) terminal
arcade. Games import `github.com/justin06lee/kalama` and implement the `Game`
interface; kalama owns the terminal, runs a fixed-timestep loop, and renders a
truecolor pixel canvas using half-block glyphs.

Players never type `kalama` — they use the [`shaw`](https://github.com/justin06lee/shaw)
launcher. This module is for game developers.

## Use it

```go
import "github.com/justin06lee/kalama"
```

Implement `kalama.Game` (`Update(dt, in) Action` + `Draw(*Canvas)`) and call
`kalama.Run(game, kalama.Options{})`.

## Persistence

`kalama.DataDir("<game>.shaw")` returns a per-game directory under
`~/.shaw/data/` (or `$SHAW_DATA_DIR`).

## Develop

```
make test
make vet
make tidy
```

## Related

- [shaw](https://github.com/justin06lee/shaw) — the launcher/package manager players run.
- [hegale](https://github.com/justin06lee/hegale) — the game registry.
```

- [ ] **Step 6: Build, vet, test (existing suite is the safety net)**

```bash
go build ./... && go vet ./... && go test ./...
```
Expected: all PASS (canvas/color/input/loop/sprite/data + the teatest smoke test). No `shaw` identifiers remain:
```bash
grep -rn 'package shaw\|KALAMA_DATA_DIR' . --include=*.go   # expected: no output
```

- [ ] **Step 7: Commit** (do NOT push yet — push happens in Phase B)

```bash
git add -A
git commit -m "refactor!: rename engine package/module shaw -> kalama; SHAW_* data dir

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Launcher → module `shaw`, binary `shaw`, no-arg opens menu, `SHAW_*`

**Files (all in `kalama/`):**
- Modify: `go.mod`, `cmd/kalama/main.go` (→ move to `cmd/shaw/main.go`), `internal/store/store.go`, `internal/registry/registry.go`, `internal/launcher/launcher.go`, `README.md`, `.gitignore`
- Modify (import paths + fixtures): `internal/store/store_test.go`, `internal/registry/registry_test.go`, `internal/launcher/launcher_test.go`

- [ ] **Step 1: Set module path and rename the command dir**

```bash
cd /Volumes/T7/Stockpile/Workspace/github.com/justin06lee/kalama
go mod edit -module github.com/justin06lee/shaw
git mv cmd/kalama cmd/shaw
```

- [ ] **Step 2: Rewrite internal import paths**

Every `github.com/justin06lee/kalama/internal/...` → `github.com/justin06lee/shaw/internal/...` across all `.go` files.
```bash
grep -rl 'justin06lee/kalama/internal' . --include=*.go | xargs sed -i '' 's#justin06lee/kalama/internal#justin06lee/shaw/internal#g'
grep -rn 'justin06lee/kalama' . --include=*.go   # expected: no output
```

- [ ] **Step 3: Re-brand env vars and paths**

`KALAMA_HOME`→`SHAW_HOME`, `KALAMA_REGISTRY`→`SHAW_REGISTRY`, `~/.kalama`→`~/.shaw`, `.kalama`→`.shaw` across non-comment code and comments in `internal/store/store.go`, `internal/registry/registry.go`, and tests.
```bash
grep -rl 'KALAMA_\|\.kalama' . --include=*.go | xargs sed -i '' -e 's/KALAMA_HOME/SHAW_HOME/g' -e 's/KALAMA_REGISTRY/SHAW_REGISTRY/g' -e 's/\.kalama/.shaw/g'
grep -rn 'KALAMA_\|"\.kalama"' . --include=*.go   # expected: no output
```
Note: `registry.DefaultURL` (the hegale raw URL) is unchanged — verify it still reads `https://raw.githubusercontent.com/justin06lee/hegale/master/index.json`.

- [ ] **Step 4: Rewrite `cmd/shaw/main.go` user-facing strings + no-arg behavior**

Apply these exact edits to `cmd/shaw/main.go`:
- Line 1 comment: `// Command kalama is the package manager and launcher for the shaw terminal arcade.` → `// Command shaw is the launcher and package manager for the shaw terminal arcade.`
- `const usage` block → replace with:
```go
const usage = `shaw — launcher and package manager for the shaw terminal arcade

Usage:
  shaw                    open the game menu
  shaw install <game>     fetch from hegale and install to ~/.shaw/games/<game>/
  shaw remove  <game>     delete an installed game
  shaw list               list installed games
  shaw play [game]        launch a game directly, or open the menu
  shaw help               show this help

Environment:
  SHAW_HOME      install location (default ~/.shaw)
  SHAW_REGISTRY  registry index URL (default hegale on GitHub)
`
```
- Error prefix in `main()`: `fmt.Fprintf(os.Stderr, "kalama: %v\n", err)` → `fmt.Fprintf(os.Stderr, "shaw: %v\n", err)`
- No-arg behavior in `run()`: change
```go
	if len(args) == 0 {
		fmt.Print(usage)
		return nil
	}
```
to
```go
	if len(args) == 0 {
		return launcher.Play("")
	}
```

- [ ] **Step 5: Friendly display name in the menu (trim `.shaw`)**

In `internal/launcher/launcher.go`:
- Empty-state string: `"no games installed — try: kalama install luma\n"` → `"no games installed — try: shaw install snake.shaw\n"`
- Add a helper (package already imports `strings`):
```go
// displayName is the friendly name shown in the menu: the install id without
// the .shaw suffix (e.g. "snake.shaw" -> "snake").
func displayName(id string) string { return strings.TrimSuffix(id, ".shaw") }
```
- In `View()`, change the per-game line from `g.Name` to `displayName(g.Name)`:
```go
		fmt.Fprintf(&b, "%s%s  %s\n", marker, displayName(g.Name), g.Description)
```

- [ ] **Step 6: Update `.gitignore` and `README.md`**

- `.gitignore`: change the ignored binary `/kalama` → `/shaw` (leave `/dist/`, `*.test` if present).
- Rewrite `README.md`:
```markdown
# shaw

The launcher and package manager for the shaw terminal arcade. Type `shaw` to
pick a game and play; install more with `shaw install`.

Games are built on the [kalama](https://github.com/justin06lee/kalama) engine and
pulled from the [hegale](https://github.com/justin06lee/hegale) registry. shaw
drops the matching per-OS binary plus a `manifest.json` into `~/.shaw/games/<game>/`.

## Install

```
go install github.com/justin06lee/shaw/cmd/shaw@latest
```

## Quickstart

```
shaw install snake.shaw   # download and install the snake game
shaw                      # open the menu and play
```

## Commands

```
shaw                    open the game menu
shaw install <game>     fetch from hegale and install to ~/.shaw/games/<game>/
shaw remove  <game>     delete an installed game
shaw list               list installed games
shaw play [game]        launch a game directly, or open the menu
shaw help               show usage
```

In the menu: `↑`/`↓` (or `k`/`j`) move, `enter` plays the highlighted game,
`q`/`esc` quits. Games are listed by their friendly name (the `.shaw` suffix is
hidden), e.g. `snake.shaw` shows as `snake`.

## Environment

| Variable        | Default                           | Purpose                   |
| --------------- | --------------------------------- | ------------------------- |
| `SHAW_HOME`     | `~/.shaw`                         | where games are installed |
| `SHAW_REGISTRY` | the hegale `index.json` on GitHub | registry index URL        |

## Install layout

```
~/.shaw/
  games/
    snake.shaw/
      manifest.json   {"name","description","version","binary"}
      snake.shaw      the executable (chmod 0755)
```

## Related

- [kalama](https://github.com/justin06lee/kalama) — the engine games are built on.
- [hegale](https://github.com/justin06lee/hegale) — the registry shaw pulls games from.
```

- [ ] **Step 7: Build, vet, test**

```bash
go build ./... && go vet ./... && go test ./...
```
Expected: all PASS. If a test hardcodes `luma` as a fixture game name it still passes (arbitrary string); only fix a test if it asserts an env var name or path you changed — update those fixtures to `SHAW_*`/`.shaw` to match.

- [ ] **Step 8: Commit** (no push yet)

```bash
git add -A
git commit -m "refactor!: rename launcher module kalama -> shaw; bare 'shaw' opens menu; SHAW_* env

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Game → module `snake.shaw`, imports `kalama` (local replace for now)

**Files (all in `luma/`):**
- Modify: `go.mod`, `main.go`, `internal/score/score.go`, `README.md`, `.gitignore`
- Modify (import paths): `internal/score/score_test.go` if it imports the module path

- [ ] **Step 1: Set module path and switch the engine dependency**

```bash
cd /Volumes/T7/Stockpile/Workspace/github.com/justin06lee/luma
go mod edit -module github.com/justin06lee/snake.shaw
go mod edit -droprequire github.com/justin06lee/shaw
go mod edit -require github.com/justin06lee/kalama@v0.0.0-00010101000000-000000000000
go mod edit -replace github.com/justin06lee/kalama=../shaw
```
(The `../shaw` dir still holds the engine in Phase A. The temporary `replace` lets the game build locally before the engine is pushed; Phase B swaps it for a real version.)

- [ ] **Step 2: Rewrite import paths and `shaw.` → `kalama.`**

In `main.go`:
- `shaw "github.com/justin06lee/shaw"` → `kalama "github.com/justin06lee/kalama"`
- `"github.com/justin06lee/luma/internal/game"` → `"github.com/justin06lee/snake.shaw/internal/game"`
- `"github.com/justin06lee/luma/internal/score"` → `"github.com/justin06lee/snake.shaw/internal/score"`
- every `shaw.` → `kalama.` (Input, Action, Quit, Continue, Canvas, Color, Run, Options)
- line 1 comment `Command luma is a Snake game` → `Command snake.shaw is a Snake game`
- `--version` output `fmt.Println("luma " + version)` → `fmt.Println("snake.shaw " + version)`
- `shaw.Options{Title: "luma", FPS: 60}` → `kalama.Options{Title: "snake", FPS: 60}`
- error prefix `"luma:"` → `"snake.shaw:"`

In `internal/score/score.go`:
- import `shaw "github.com/justin06lee/shaw"` → `kalama "github.com/justin06lee/kalama"`
- `shaw.DataDir("luma")` → `kalama.DataDir("snake.shaw")`
- package comment `luma high score` → `snake.shaw high score`

```bash
sed -i '' \
  -e 's#shaw "github.com/justin06lee/shaw"#kalama "github.com/justin06lee/kalama"#' \
  -e 's#justin06lee/luma/internal#justin06lee/snake.shaw/internal#g' \
  -e 's/\bshaw\./kalama./g' \
  main.go internal/score/score.go
grep -rn 'justin06lee/shaw\|shaw\.' . --include=*.go   # expected: no output
```
Then hand-edit the remaining literal strings in `main.go`/`score.go` (the `luma` strings, Title, version line, comments) per the list above — `sed` above only handled imports and `shaw.` qualifiers.

- [ ] **Step 3: Re-resolve the module graph**

```bash
go mod tidy
go build ./... && go vet ./... && go test ./...
```
Expected: all PASS (game logic + score tests). `go build -o snake.shaw .` produces a binary.

- [ ] **Step 4: Update `.gitignore` and `README.md`**

- `.gitignore`: `/luma` → `/snake.shaw`.
- Rewrite `README.md`:
```markdown
# snake.shaw

Snake for the shaw terminal arcade. Built on the
[kalama](https://github.com/justin06lee/kalama) engine.

Guide the snake, eat the food, grow longer, and don't run into the walls or
yourself.

## Build & run

```
go run .
```

Or build a binary:

```
go build -o snake.shaw .
./snake.shaw
```

## Controls

- Move: arrow keys or `WASD`
- Quit: `esc`
- Restart (after game over): `enter` or `space`

## High scores

Your best score is saved under `~/.shaw/data/snake.shaw/` (or `$SHAW_DATA_DIR/snake.shaw`)
and persists across runs.

## Install

Install and play via the [`shaw`](https://github.com/justin06lee/shaw) launcher:

```
shaw install snake.shaw
shaw
```
```

- [ ] **Step 5: Commit** (no push yet; the `replace` directive is committed for now and removed in Phase B)

```bash
git add -A
git commit -m "refactor!: rename luma -> snake.shaw; import kalama engine

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Registry — `hegale/index.json` + README

**Files (in `hegale/`):** `index.json`, `README.md`

- [ ] **Step 1: Replace `index.json`** with the `snake.shaw` entry

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
        "linux/amd64": "https://github.com/justin06lee/snake.shaw/releases/download/v1.0.0/snake.shaw-linux-amd64",
        "linux/arm64": "https://github.com/justin06lee/snake.shaw/releases/download/v1.0.0/snake.shaw-linux-arm64"
      }
    }
  ]
}
```

- [ ] **Step 2: Validate JSON**

```bash
cd /Volumes/T7/Stockpile/Workspace/github.com/justin06lee/hegale
python3 -m json.tool index.json > /dev/null && echo OK   # expected: OK
```

- [ ] **Step 3: Update `README.md`** — replace mentions of `kalama` (package manager) with `shaw`, `luma` with `snake.shaw`, and add a note that the menu shows the friendly name. Key replacements:
- "The [kalama](...) package manager reads this index" → "The [shaw](https://github.com/justin06lee/shaw) launcher reads this index"
- "`kalama install <name>`" → "`shaw install <name>`"
- Catalog entry `luma` → `snake.shaw`
- Add: "Install ids keep the `.shaw` suffix; the launcher menu shows the friendly name (e.g. `snake.shaw` → `snake`)."
- Related links: keep shaw, change "kalama — the package manager" to "kalama — the engine".

- [ ] **Step 4: Commit** (no push yet)

```bash
git add -A
git commit -m "feat: registry entry snake.shaw; shaw is the launcher

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Move ecosystem docs to the launcher (shaw) repo

The `docs/superpowers/` tree currently lives in the engine working tree (`shaw/`, soon module `kalama`). The launcher is the public face, so the ecosystem docs move there (the `kalama/` working tree, soon module `shaw`).

- [ ] **Step 1: Copy docs into the launcher tree and remove from the engine tree**

```bash
cd /Volumes/T7/Stockpile/Workspace/github.com/justin06lee
mkdir -p kalama/docs
cp -R shaw/docs/superpowers kalama/docs/superpowers
git -C kalama add docs
git -C shaw rm -r docs
```

- [ ] **Step 2: Commit both**

```bash
git -C kalama commit -m "docs: home the arcade ecosystem specs/plans in the shaw launcher repo

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
git -C shaw commit -m "docs: move ecosystem specs/plans to the shaw launcher repo

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Phase B — Cutover (outward, irreversible) — CONFIRM WITH USER BEFORE STARTING

> Everything below force-pushes / creates GitHub repos / publishes a release. Pause and get an explicit go-ahead before Step 1. Verify `gh auth status` first.

### Task 6: Publish the engine to github `kalama`

- [ ] **Step 1:** Confirm GitHub auth: `gh auth status` (expected: logged in as justin06lee).
- [ ] **Step 2:** Repoint the engine remote and force-push.
```bash
cd /Volumes/T7/Stockpile/Workspace/github.com/justin06lee/shaw
git remote set-url origin https://github.com/justin06lee/kalama.git
git push --force origin HEAD:master
```
- [ ] **Step 3:** Capture the engine commit for pinning: `git rev-parse HEAD`.

### Task 7: Pin the game to the real engine, create repo, push

- [ ] **Step 1:** Remove the local replace and pin the published engine.
```bash
cd /Volumes/T7/Stockpile/Workspace/github.com/justin06lee/luma
go mod edit -dropreplace github.com/justin06lee/kalama
GOFLAGS=-mod=mod go get github.com/justin06lee/kalama@master
go mod tidy
go build ./... && go test ./...   # expected: PASS against the published engine
```
- [ ] **Step 2:** Commit the dependency pin.
```bash
git add go.mod go.sum
git commit -m "build: pin published kalama engine; drop local replace

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```
- [ ] **Step 3:** Create the GitHub repo and push.
```bash
gh repo create justin06lee/snake.shaw --public --source=. --remote=origin --push
```
(If the remote already exists, instead: `git remote set-url origin https://github.com/justin06lee/snake.shaw.git && git push --force origin HEAD:master`.)

### Task 8: Publish the launcher to github `shaw`

- [ ] **Step 1:** Repoint the launcher remote and force-push.
```bash
cd /Volumes/T7/Stockpile/Workspace/github.com/justin06lee/kalama
git remote set-url origin https://github.com/justin06lee/shaw.git
git push --force origin HEAD:master
```

### Task 9: Publish the registry

- [ ] **Step 1:** Push hegale.
```bash
cd /Volumes/T7/Stockpile/Workspace/github.com/justin06lee/hegale
git push origin HEAD:master
```

### Task 10: Build + release `snake.shaw` binaries

- [ ] **Step 1:** Cross-compile the four targets.
```bash
cd /Volumes/T7/Stockpile/Workspace/github.com/justin06lee/luma
rm -rf dist && mkdir dist
for t in darwin/arm64 darwin/amd64 linux/amd64 linux/arm64; do
  os=${t%/*}; arch=${t#*/}
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -o dist/snake.shaw-$os-$arch .
done
ls dist   # expected: snake.shaw-darwin-arm64 ...-darwin-amd64 ...-linux-amd64 ...-linux-arm64
```
- [ ] **Step 2:** Create the release with assets.
```bash
gh release create v1.0.0 dist/* --repo justin06lee/snake.shaw \
  --title "snake.shaw v1.0.0" --notes "Classic snake for the shaw terminal arcade."
```

### Task 11: Rename local working dirs to match roles

Done from the parent dir so no shell is inside a dir being renamed. (Run from a shell whose cwd is the parent.)

- [ ] **Step 1:** Three-way swap + game rename.
```bash
cd /Volumes/T7/Stockpile/Workspace/github.com/justin06lee
mv shaw shaw.engine.tmp      # engine (module kalama, remote kalama)
mv kalama shaw              # launcher (module shaw, remote shaw)
mv shaw.engine.tmp kalama   # engine lands at kalama/
mv luma snake.shaw          # game
ls -d shaw kalama snake.shaw hegale   # expected: all four present
```
Note: the second additional working directory at `/Users/huiyunlee/Workspace/github.com/justin06lee/shaw` is a separate checkout — leave it; it is not part of this swap.

### Task 12: End-to-end verification (darwin/arm64)

- [ ] **Step 1:** Install from the live registry into a throwaway home.
```bash
SHAW_HOME=$(mktemp -d) SHAW_REGISTRY=https://raw.githubusercontent.com/justin06lee/hegale/master/index.json \
  go -C /Volumes/T7/Stockpile/Workspace/github.com/justin06lee/shaw run ./cmd/shaw install snake.shaw
```
Expected: `installed snake.shaw 1.0.0`.
- [ ] **Step 2:** Run the installed binary's version and list it (reuse the same `SHAW_HOME`).
```bash
SHAW_HOME=<dir from step 1>; "$SHAW_HOME/games/snake.shaw/snake.shaw" --version   # expected: snake.shaw 1.0.0
SHAW_HOME="$SHAW_HOME" go -C .../shaw run ./cmd/shaw list                          # expected: snake.shaw row
```
- [ ] **Step 3:** Confirm bare `shaw` opens the menu (manual / TTY): `go -C .../shaw run ./cmd/shaw` shows "shaw arcade — pick a game" with `snake` listed (friendly name). Document as human-verified; the menu UI is not auto-testable.

---

## Self-review notes

- **Spec coverage:** swap (Tasks 1,2 + 6,8), no-arg menu (Task 2 Step 4), `<name>.shaw` standard + friendly menu name (Task 3, Task 2 Step 5), luma→snake.shaw (Task 3,7,10), `~/.shaw`+`SHAW_*` (Tasks 1,2), hegale update (Task 4,9), docs move (Task 5), dep graph/no cycles (launcher never imports engine — preserved), verification boundary (Task 12). All covered.
- **Reversibility:** all of Phase A is local commits (resettable). Phase B is the irreversible cutover, gated on user confirmation.
- **Old repos:** the old `luma` GitHub repo is left as-is (not deleted — deletion is irreversible and out of scope). Mention it to the user; archive/delete only on explicit request.
```
