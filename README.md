# shaw

A monkeytype-style touch-typing trainer that lives in your terminal and sources
text from a folder of your own `.txt` files. Three run modes (timed, word-count,
zen), live WPM and accuracy, an ASCII WPM chart, error breakdown, and a
persistent history across sessions.

## Install

A single command, once Go is installed:

```bash
go install github.com/justin06lee/shaw@latest
```

This drops a `shaw` binary into `$(go env GOBIN)` (or `$(go env GOPATH)/bin`
when `GOBIN` is unset). Make sure that directory is on your `PATH`:

```bash
# bash / zsh
export PATH="$(go env GOPATH)/bin:$PATH"
```

After that, `shaw` runs from anywhere.

### Building from a clone

```bash
git clone https://github.com/justin06lee/shaw
cd shaw
make install        # installs to $GOBIN / $GOPATH/bin
# or
make build          # just builds ./shaw in the working tree
```

`make help` lists every target.

## Quick start

```bash
# Save a default corpus directory once. After this, plain `shaw` uses it.
shaw --set-dir ~/typing-corpus

shaw                       # uses the saved default, opens the config bar
shaw --time 60             # start straight into a 60-second run
shaw ~/some/other/folder   # one-off folder, ignoring the default
shaw --history             # cross-run WPM progress chart, then exit
```

`~/typing-corpus` is anything you want — paste in some plain `.txt` files
(books, articles, code comments) and shaw will pick from them at random.

## How the corpus directory is resolved

In order of priority:

1. A positional argument: `shaw /path/to/folder`.
2. The `SHAW_DIR` environment variable.
3. The `default_dir` value saved by `shaw --set-dir`.

If none of those resolve to a folder with at least one `.txt` file, shaw exits
with a message telling you how to set one.

## Modes

| Mode  | Goal                                | Ends when             |
|-------|-------------------------------------|-----------------------|
| time  | 15, 30, 60, or 120 seconds          | timer expires         |
| words | 10, 25, 50, or 100 correct words    | target count reached  |
| zen   | type until you press `Esc`          | you press `Esc`       |

A `.txt` file is selected at random, read from the top, and tokenised into a
single stream. Newlines collapse to spaces; consecutive whitespace is one
space; non-UTF-8 files are skipped. When a file runs out, another is appended
so timed and zen runs never run dry.

## Key bindings

While the config bar is editable (before you start typing):

- `←` / `→` — change mode (time / words / zen)
- `↑` / `↓` — change the target value
- any printable key — starts the run

While typing:

- printable keys / `Space` — typed against the target text
- `Backspace` — undo one character
- `Esc` — in **time** or **words** mode: abort, no stats saved. in **zen**
  mode: finish the run, compute stats, save to history.
- `Ctrl-C` — quit shaw entirely

On the result screen:

- `Enter` / `Esc` — start a new run with the same settings

## Statistics

After every run shaw shows:

- net WPM (correct chars / 5 / minutes)
- accuracy (% of keystrokes that were correct)
- raw WPM, consistency (100 − coefficient of variation of per-second WPM)
- elapsed seconds
- an ASCII bar chart of cumulative WPM across the run
- the top mistyped characters with counts

Every completed run (including zen) is appended to
`$XDG_CONFIG_HOME/shaw/history.json` (defaults to `~/.config/shaw/history.json`).
Run `shaw --history` to see a chart of net WPM across all past runs.

If the history file ever becomes corrupt, the next run preserves it as
`history.json.corrupt` and starts a fresh file — your numbers are never
silently destroyed.

## Configuration files

- `~/.config/shaw/config.json` — `default_dir` for the corpus folder
- `~/.config/shaw/history.json` — run history

Both honour `XDG_CONFIG_HOME` if set.

## CLI reference

```
shaw [folder] [flags]

positional arg:
  folder              corpus directory (overrides SHAW_DIR / saved default)

flags:
  --time N            timed mode: 15, 30, 60, or 120
  --words N           word mode: 10, 25, 50, or 100
  --zen               zen mode: type until Esc
  --history           print progress chart across past runs and exit
  --set-dir PATH      save PATH as shaw's default corpus directory, then exit
```

Mode flags are mutually exclusive. When none are given, shaw opens with a
30-second timed run preselected; change it in the config bar before typing.

## Development

```bash
make test    # go test ./...
make vet     # go vet ./...
make fmt     # gofmt -w .
```

The codebase is split into five focused packages under `internal/`:

- `corpus` — `.txt` discovery and endless normalised word stream
- `run` — pure typing-session state machine (no I/O)
- `stats` — WPM / accuracy / consistency math and ASCII chart
- `history` — XDG-aware JSON persistence, corruption-tolerant
- `tui` — bubbletea Model: config bar, scrolling viewport, result screen
- `config` — `default_dir` persistence

Plus a thin `main.go` that wires them together.
