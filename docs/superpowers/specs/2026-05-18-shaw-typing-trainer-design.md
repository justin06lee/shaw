# shaw — Terminal Typing Trainer

**Date:** 2026-05-18
**Status:** Approved design

## Summary

`shaw` is a command-line typing trainer. Point it at a folder of text files and
it runs a monkeytype-style typing test in the terminal: a random file is read
top-to-bottom, displayed three lines at a time with a scrolling viewport, and
the user types it. After each run it shows typing speed, accuracy, an in-run
WPM chart, and an error breakdown, then persists the result so progress can be
charted across sessions.

## Goals

- Fast, monkeytype-like typing practice in the terminal.
- Source text from the user's own `.txt` files.
- Multiple run modes: timed, word-count, and zen.
- Rich post-run statistics plus cross-run progress tracking.

## Non-Goals

- No built-in word lists or downloadable content packs — text comes only from
  the user's folder.
- No multiplayer, no network features.
- No theming/config files beyond the persisted history (YAGNI).

## CLI

```
shaw [folder]              # start; folder recursively scanned for .txt files
shaw [folder] --time 30    # preselect timed mode: 15 | 30 | 60 | 120 seconds
shaw [folder] --words 50   # preselect word mode: 10 | 25 | 50 | 100 words
shaw [folder] --zen        # preselect zen mode (type until Esc)
shaw --history             # print cross-run progress chart and exit
```

- `folder` defaults to the current working directory when omitted.
- Mode flags are mutually exclusive; supplying more than one is an error.
- When no mode flag is given, the app opens with a default of timed 30s, which
  the user can change in the top config bar before typing.
- `--history` ignores the folder argument and does not start a run.

## Screen Layout

A single screen, modeled on monkeytype. No separate start screen.

```
                          shaw

        [ time  words  zen ]   [ 15  30  60  120 ]

   the quick brown fox jumps over the lazy dog and then
   keeps going while the viewport scrolls three lines at
   a time as the typed words move out of view above

                     30   ·   esc to restart
```

- **Top config bar:** two segmented controls — mode (`time` / `words` / `zen`)
  and the value for that mode (`15 30 60 120`, or `10 25 50 100`, or nothing
  for zen). The active selection is highlighted.
- **Text area:** exactly three lines visible. Word-wrapped to terminal width.
  As the user finishes a line, the viewport scrolls so the active line stays in
  the middle.
- **Footer:** live indicator (remaining seconds, or words done / target, or
  elapsed time in zen) and key hints.

### Config bar interaction

- **Idle** (before first keystroke of a run, or after a run finishes): the
  config bar is focused and editable. Arrow keys / `h` `l` move within a
  control; `Tab` switches between the mode control and the value control;
  changing the mode swaps the value options and resets to that mode's default.
  Changing any setting regenerates the text and resets the run.
- **Active** (first keystroke typed until the run ends): the config bar is
  visually dimmed and ignores input. All keystrokes go to typing.
- A run ends on goal reached, or `Esc`. `Esc` during an active run aborts it
  (no stats, no history) and returns to idle. `Esc` while idle restarts with
  fresh text.

## Run Modes

| Mode  | Goal                              | Ends when                          |
|-------|-----------------------------------|------------------------------------|
| time  | 15 / 30 / 60 / 120 seconds        | timer expires                      |
| words | 10 / 25 / 50 / 100 correct words  | target word count completed        |
| zen   | none                              | user presses `Esc`                 |

The timer starts on the first keystroke.

## Text Sourcing

- The target folder is walked recursively; every `.txt` file is collected.
- A random file is chosen and read from its top.
- File content is normalized into a continuous word stream: all newlines and
  runs of whitespace collapse to single spaces. File line breaks are **not**
  preserved.
- When a file's words are exhausted before the run's goal is met, another
  random file is chosen and appended to the stream seamlessly.
- The stream is effectively endless, so timed and zen runs never run dry.

### Errors

- Empty folder or no `.txt` files found → friendly message, exit non-zero.
- A file that is not valid UTF-8 → skipped; if all files are skipped, treat as
  "no usable files".
- Terminal narrower than a minimum width (e.g. 40 columns) → message asking the
  user to widen the terminal.
- `Ctrl-C` → immediate quit, no history written.

## Components

Each is its own Go package with a focused responsibility and is unit-testable
in isolation.

### `corpus`

Discovers and serves text.

- `Scan(dir string) ([]string, error)` — recursive `.txt` discovery.
- `TextStream` — given the file list, yields an endless sequence of words.
  Picks a random file, reads top-to-bottom, normalizes whitespace, tokenizes on
  spaces; on exhaustion picks another random file. Skips non-UTF-8 files.
- No terminal or UI dependencies.

### `run`

A pure state machine for one typing session. No I/O.

- Holds: mode + target, the word stream consumed so far, per-character typed
  state (untyped / correct / incorrect), cursor index, keystroke log, run
  start/end timestamps, and per-second WPM samples.
- Accepts keystroke events (character, backspace) and advances state.
- Monkeytype error handling: an incorrect character is recorded red and the
  cursor still advances; backspace moves back and allows correction.
- Exposes whether the goal has been reached.

### `tui`

The bubbletea `Model`. Owns rendering and input routing.

- Renders the config bar, three-line word-wrapped viewport, and footer.
- Routes input: to the config bar when idle, to `run` when active.
- Word colors: grey untyped, white correct, red incorrect.
- Pulls words from `corpus.TextStream` as the viewport needs them.
- On run end, switches to the result view.

### `stats`

Computes results from a finished `run`.

- **WPM** (monkeytype-standard): raw = (all typed chars / 5) / minutes;
  net = (correct chars / 5) / minutes.
- **Accuracy** = correct keystrokes / total keystrokes.
- **Consistency** = coefficient of variation of per-second WPM samples,
  reported as a percentage (higher = steadier).
- **Error breakdown** = characters and words missed most often.
- Renders an ASCII line chart of per-second WPM (block/braille glyphs).

### `history`

Persists and reads run results.

- Storage: `~/.config/shaw/history.json` (XDG-aware: respects
  `$XDG_CONFIG_HOME`).
- Each record: timestamp, mode, target, net WPM, raw WPM, accuracy,
  consistency.
- `Append(record)` after each completed run (aborted runs are not saved).
- `Load()` for the `--history` view, which renders an ASCII chart of net WPM
  over time across past runs.

## Data Flow

```
corpus.Scan ─▶ corpus.TextStream ─▶ tui ─┬─▶ run (state machine)
                                         │
              keystrokes ───────────────▶┘
run (finished) ─▶ stats ─▶ result view
run (finished) ─▶ history.Append
--history ─▶ history.Load ─▶ ASCII chart
```

## Result Screen

Shown after a completed (non-aborted) run:

- Net WPM and accuracy, prominently.
- In-run WPM line chart (per-second samples).
- Raw WPM and consistency.
- Error breakdown (most-missed characters / words).
- A line confirming the result was saved to history.
- Key hints: `Esc`/`Enter` to start a new run (returns to idle config bar).

## Testing

- `corpus`: discovery on a temp dir tree; whitespace normalization;
  stream rollover to a new file on exhaustion; non-UTF-8 skip.
- `run`: keystroke sequences produce correct char states; backspace; goal
  detection for each mode; WPM-sample bucketing.
- `stats`: known keystroke logs produce expected WPM / accuracy / consistency;
  error breakdown ranking.
- `history`: append then load round-trip; XDG path resolution; corrupt-file
  tolerance.
- `tui`: bubbletea `Model` update tested via simulated messages — config bar
  editable only when idle; idle vs active input routing; viewport scroll.

## Open Questions

None — design approved.
