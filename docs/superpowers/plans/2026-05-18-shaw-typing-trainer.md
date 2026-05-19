# shaw Typing Trainer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `shaw`, a monkeytype-style terminal typing trainer that draws text from a folder of `.txt` files and reports typing speed, accuracy, and progress.

**Architecture:** Five focused Go packages under `internal/` — `corpus` (text discovery + endless word stream), `run` (pure typing-session state machine), `stats` (post-run metrics + ASCII charts), `history` (JSON persistence), `tui` (bubbletea Model: config bar, typing viewport, result screen). A thin `main.go` parses flags and wires them together.

**Tech Stack:** Go 1.26, [bubbletea](https://github.com/charmbracelet/bubbletea) (TUI runtime), [lipgloss](https://github.com/charmbracelet/lipgloss) (styling). Standard `testing` package for tests.

---

## File Structure

```
go.mod
main.go                       CLI flag parsing, wiring, --history view
internal/
  corpus/corpus.go            Scan(), TextStream
  corpus/corpus_test.go
  run/run.go                  Run state machine, Keystroke, enums
  run/run_test.go
  stats/stats.go              Compute(), RenderChart()
  stats/stats_test.go
  history/history.go          Path(), Append(), Load(), Record
  history/history_test.go
  tui/wrap.go                 WrapLines(), Viewport() — pure layout helpers
  tui/wrap_test.go
  tui/model.go                bubbletea Model: state machine + input routing
  tui/model_test.go
  tui/view.go                 render functions (config bar, typing area, result)
docs/superpowers/specs/2026-05-18-shaw-typing-trainer-design.md   (exists)
```

**Package dependency direction:** `corpus`, `run`, `history` are leaf packages.
`stats` imports `run`. `tui` imports `corpus`, `run`, `stats`, `history`.
`main` imports `corpus`, `tui`, `stats`, `history`.

---

## Task 1: Project scaffold

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `internal/version_test.go`

- [ ] **Step 1: Initialize the module**

Run:
```bash
cd /Users/huiyunlee/Workspace/github.com/justin06lee/shaw
go mod init github.com/justin06lee/shaw
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
```
Expected: `go.mod` and `go.sum` created listing both dependencies.

- [ ] **Step 2: Write a placeholder main**

Create `main.go`:
```go
package main

import "fmt"

func main() {
	fmt.Println("shaw")
}
```

- [ ] **Step 3: Write a smoke test**

Create `internal/version_test.go`:
```go
package internal

import "testing"

// TestBuilds is a placeholder ensuring the module compiles and `go test` runs.
func TestBuilds(t *testing.T) {
	if 1+1 != 2 {
		t.Fatal("arithmetic broken")
	}
}
```

- [ ] **Step 4: Verify build and test**

Run: `go build ./... && go test ./...`
Expected: build succeeds, `ok github.com/justin06lee/shaw/internal`.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum main.go internal/version_test.go
git commit -m "chore: scaffold shaw Go module"
```

---

## Task 2: corpus.Scan — recursive .txt discovery

**Files:**
- Create: `internal/corpus/corpus.go`
- Test: `internal/corpus/corpus_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/corpus/corpus_test.go`:
```go
package corpus

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanFindsTxtRecursively(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "alpha")
	writeFile(t, filepath.Join(dir, "sub", "b.txt"), "beta")
	writeFile(t, filepath.Join(dir, "sub", "c.md"), "ignored")

	got, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)
	want := []string{
		filepath.Join(dir, "a.txt"),
		filepath.Join(dir, "sub", "b.txt"),
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestScanEmptyDir(t *testing.T) {
	got, err := Scan(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no files, got %v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/corpus/`
Expected: FAIL — `undefined: Scan`.

- [ ] **Step 3: Implement Scan**

Create `internal/corpus/corpus.go`:
```go
// Package corpus discovers .txt files and serves their words as an endless stream.
package corpus

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// Scan walks dir recursively and returns the paths of all .txt files.
func Scan(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.EqualFold(filepath.Ext(path), ".txt") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/corpus/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/corpus/
git commit -m "feat: recursive .txt discovery in corpus package"
```

---

## Task 3: corpus.TextStream — endless normalized word stream

**Files:**
- Modify: `internal/corpus/corpus.go`
- Test: `internal/corpus/corpus_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/corpus/corpus_test.go`:
```go
import "math/rand"   // add to the existing import block

func TestTextStreamNormalizesWhitespace(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "one\ntwo\t three   four\n\nfive")
	files, _ := Scan(dir)

	s := NewTextStream(files, rand.New(rand.NewSource(1)))
	var got []string
	for i := 0; i < 5; i++ {
		w, ok := s.Next()
		if !ok {
			t.Fatalf("stream ended early at %d", i)
		}
		got = append(got, w)
	}
	want := []string{"one", "two", "three", "four", "five"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestTextStreamRollsOverToAnotherFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "x")
	writeFile(t, filepath.Join(dir, "b.txt"), "y")
	files, _ := Scan(dir)

	s := NewTextStream(files, rand.New(rand.NewSource(1)))
	// Two single-word files: 10 reads must all succeed (stream is endless).
	for i := 0; i < 10; i++ {
		if _, ok := s.Next(); !ok {
			t.Fatalf("stream ended at read %d, expected endless", i)
		}
	}
}

func TestTextStreamSkipsNonUTF8(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.txt"), []byte{0xff, 0xfe, 0xfd}, 0o644); err != nil {
		t.Fatal(err)
	}
	files, _ := Scan(dir)
	s := NewTextStream(files, rand.New(rand.NewSource(1)))
	if _, ok := s.Next(); ok {
		t.Fatal("expected no words from a non-UTF8-only corpus")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/corpus/`
Expected: FAIL — `undefined: NewTextStream`.

- [ ] **Step 3: Implement TextStream**

Append to `internal/corpus/corpus.go`:
```go
import (
	"math/rand"   // add these to the existing import block
	"os"
	"unicode/utf8"
)

// TextStream yields an endless sequence of words drawn from a set of files.
// When a file is exhausted it picks another random file. Files that are not
// valid UTF-8 are skipped permanently.
type TextStream struct {
	files []string
	rng   *rand.Rand
	buf   []string // unread words from the current file
	dead  map[int]bool
}

// NewTextStream creates a stream over files using rng for file selection.
func NewTextStream(files []string, rng *rand.Rand) *TextStream {
	return &TextStream{files: files, rng: rng, dead: map[int]bool{}}
}

// Next returns the next word. It returns ("", false) only when no file in the
// corpus yields usable words (empty corpus or every file is non-UTF8/empty).
func (s *TextStream) Next() (string, bool) {
	for len(s.buf) == 0 {
		if !s.loadRandomFile() {
			return "", false
		}
	}
	w := s.buf[0]
	s.buf = s.buf[1:]
	return w, true
}

// loadRandomFile fills buf from a random non-dead file. Returns false when no
// file can produce words.
func (s *TextStream) loadRandomFile() bool {
	alive := make([]int, 0, len(s.files))
	for i := range s.files {
		if !s.dead[i] {
			alive = append(alive, i)
		}
	}
	if len(alive) == 0 {
		return false
	}
	idx := alive[s.rng.Intn(len(alive))]
	data, err := os.ReadFile(s.files[idx])
	if err != nil || !utf8.Valid(data) {
		s.dead[idx] = true
		return s.loadRandomFile()
	}
	words := strings.Fields(string(data))
	if len(words) == 0 {
		s.dead[idx] = true
		return s.loadRandomFile()
	}
	s.buf = words
	return true
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/corpus/`
Expected: PASS (all corpus tests).

- [ ] **Step 5: Commit**

```bash
git add internal/corpus/
git commit -m "feat: endless normalized word stream in corpus package"
```

---

## Task 4: run package — core typing state machine

**Files:**
- Create: `internal/run/run.go`
- Test: `internal/run/run_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/run/run_test.go`:
```go
package run

import "testing"

func TestTypeMarksCharStates(t *testing.T) {
	r := New(ModeZen, 0)
	r.AppendWords([]string{"go"})
	r.Type('g')
	r.Type('x') // wrong
	states := r.States()
	if states[0] != Correct {
		t.Errorf("char 0: got %v, want Correct", states[0])
	}
	if states[1] != Incorrect {
		t.Errorf("char 1: got %v, want Incorrect", states[1])
	}
	if r.Cursor() != 2 {
		t.Errorf("cursor: got %d, want 2", r.Cursor())
	}
}

func TestBackspaceResetsChar(t *testing.T) {
	r := New(ModeZen, 0)
	r.AppendWords([]string{"go"})
	r.Type('g')
	r.Backspace()
	if r.Cursor() != 0 {
		t.Errorf("cursor: got %d, want 0", r.Cursor())
	}
	if r.States()[0] != Untyped {
		t.Errorf("char 0: got %v, want Untyped", r.States()[0])
	}
}

func TestBackspaceAtStartIsNoop(t *testing.T) {
	r := New(ModeZen, 0)
	r.AppendWords([]string{"go"})
	r.Backspace()
	if r.Cursor() != 0 {
		t.Errorf("cursor: got %d, want 0", r.Cursor())
	}
}

func TestTypePastEndIsNoop(t *testing.T) {
	r := New(ModeZen, 0)
	r.AppendWords([]string{"a"})
	r.Type('a')
	r.Type('b') // past end
	if r.Cursor() != 1 {
		t.Errorf("cursor: got %d, want 1", r.Cursor())
	}
}

func TestAppendWordsJoinsWithSpaces(t *testing.T) {
	r := New(ModeZen, 0)
	r.AppendWords([]string{"one", "two"})
	if string(r.Text()) != "one two" {
		t.Errorf("text: got %q, want %q", string(r.Text()), "one two")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/run/`
Expected: FAIL — `undefined: New`.

- [ ] **Step 3: Implement the core state machine**

Create `internal/run/run.go`:
```go
// Package run is a pure state machine for a single typing session.
// It performs no I/O. Time is read through an injectable Now function.
package run

import "time"

// Mode is the kind of typing run.
type Mode int

const (
	ModeTime  Mode = iota // run ends when Target seconds elapse
	ModeWords             // run ends when Target words are typed
	ModeZen               // run ends only on external request (Esc)
)

// CharState is the typed status of one character of the target text.
type CharState int

const (
	Untyped CharState = iota
	Correct
	Incorrect
)

// Keystroke is one recorded input event, timestamped from run start.
type Keystroke struct {
	At        time.Duration
	Typed     rune
	Expected  rune
	Correct   bool
	Backspace bool
}

// Run holds the state of one typing session.
type Run struct {
	mode    Mode
	target  int
	text    []rune
	states  []CharState
	cursor  int
	log     []Keystroke
	started time.Time
	Now     func() time.Time // injectable clock; defaults to time.Now
}

// New creates a Run. target is seconds for ModeTime, word count for ModeWords,
// and ignored for ModeZen.
func New(mode Mode, target int) *Run {
	return &Run{mode: mode, target: target, Now: time.Now}
}

// AppendWords adds words to the target text, joined by single spaces.
func (r *Run) AppendWords(words []string) {
	for _, w := range words {
		if len(r.text) > 0 {
			r.text = append(r.text, ' ')
			r.states = append(r.states, Untyped)
		}
		for _, ch := range w {
			r.text = append(r.text, ch)
			r.states = append(r.states, Untyped)
		}
	}
}

// Type records a typed rune at the cursor and advances it.
func (r *Run) Type(typed rune) {
	if r.cursor >= len(r.text) {
		return
	}
	if r.started.IsZero() {
		r.started = r.Now()
	}
	expected := r.text[r.cursor]
	correct := typed == expected
	if correct {
		r.states[r.cursor] = Correct
	} else {
		r.states[r.cursor] = Incorrect
	}
	r.log = append(r.log, Keystroke{
		At: r.elapsed(), Typed: typed, Expected: expected, Correct: correct,
	})
	r.cursor++
}

// Backspace moves the cursor back one and clears that character's state.
func (r *Run) Backspace() {
	if r.cursor == 0 {
		return
	}
	r.cursor--
	r.states[r.cursor] = Untyped
	r.log = append(r.log, Keystroke{At: r.elapsed(), Backspace: true})
}

// elapsed is the time since the first keystroke (0 before it).
func (r *Run) elapsed() time.Duration {
	if r.started.IsZero() {
		return 0
	}
	return r.Now().Sub(r.started)
}

// Accessors.
func (r *Run) Text() []rune        { return r.text }
func (r *Run) States() []CharState { return r.states }
func (r *Run) Cursor() int         { return r.cursor }
func (r *Run) Log() []Keystroke    { return r.log }
func (r *Run) Mode() Mode          { return r.mode }
func (r *Run) Target() int         { return r.target }
func (r *Run) Started() bool       { return !r.started.IsZero() }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/run/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/run/
git commit -m "feat: core typing state machine in run package"
```

---

## Task 5: run package — goal detection and duration

**Files:**
- Modify: `internal/run/run.go`
- Test: `internal/run/run_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/run/run_test.go`:
```go
import "time"  // add to the existing import block

// fakeClock returns successive fixed times for the injectable Now function.
func fakeClock(times ...time.Time) func() time.Time {
	i := 0
	return func() time.Time {
		t := times[i]
		if i < len(times)-1 {
			i++
		}
		return t
	}
}

func TestGoalWordsReachedWhenTextFullyTyped(t *testing.T) {
	r := New(ModeWords, 1)
	r.AppendWords([]string{"hi"})
	if r.GoalReached() {
		t.Fatal("goal reached before typing")
	}
	r.Type('h')
	r.Type('i')
	if !r.GoalReached() {
		t.Fatal("goal not reached after typing all chars")
	}
}

func TestGoalTimeReachedWhenTargetSecondsElapse(t *testing.T) {
	base := time.Unix(0, 0)
	r := New(ModeTime, 30)
	r.Now = fakeClock(base, base.Add(30*time.Second))
	r.AppendWords([]string{"abc"})
	r.Type('a') // started at base
	if !r.GoalReached() {
		t.Fatal("goal not reached at +30s")
	}
}

func TestGoalTimeNotReachedEarly(t *testing.T) {
	base := time.Unix(0, 0)
	r := New(ModeTime, 30)
	r.Now = fakeClock(base, base.Add(5*time.Second))
	r.AppendWords([]string{"abc"})
	r.Type('a')
	if r.GoalReached() {
		t.Fatal("goal reached too early at +5s")
	}
}

func TestGoalZenNeverReached(t *testing.T) {
	r := New(ModeZen, 0)
	r.AppendWords([]string{"a"})
	r.Type('a')
	if r.GoalReached() {
		t.Fatal("zen mode should never auto-finish")
	}
}

func TestDurationIsLastKeystrokeTime(t *testing.T) {
	base := time.Unix(0, 0)
	r := New(ModeZen, 0)
	r.Now = fakeClock(base, base.Add(2*time.Second))
	r.AppendWords([]string{"ab"})
	r.Type('a') // sets started=base
	r.Type('b') // logged at +2s
	if r.Duration() != 2*time.Second {
		t.Fatalf("duration: got %v, want 2s", r.Duration())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/run/`
Expected: FAIL — `undefined: (*Run).GoalReached`.

- [ ] **Step 3: Implement GoalReached and Duration**

Append to `internal/run/run.go`:
```go
// GoalReached reports whether the run's completion condition is met.
func (r *Run) GoalReached() bool {
	switch r.mode {
	case ModeWords:
		return len(r.text) > 0 && r.cursor >= len(r.text)
	case ModeTime:
		return r.Started() &&
			r.elapsed() >= time.Duration(r.target)*time.Second
	default: // ModeZen
		return false
	}
}

// Duration is the time of the last keystroke relative to run start.
func (r *Run) Duration() time.Duration {
	if len(r.log) == 0 {
		return 0
	}
	return r.log[len(r.log)-1].At
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/run/`
Expected: PASS (all run tests).

- [ ] **Step 5: Commit**

```bash
git add internal/run/
git commit -m "feat: goal detection and duration in run package"
```

---

## Task 6: stats.Compute — WPM, accuracy, consistency, errors

**Files:**
- Create: `internal/stats/stats.go`
- Test: `internal/stats/stats_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/stats/stats_test.go`:
```go
package stats

import (
	"testing"
	"time"

	"github.com/justin06lee/shaw/internal/run"
)

// buildRun types want against an all-correct target, with the final keystroke
// landing at total duration. Each char is typed correct unless in wrong set.
func buildRun(t *testing.T, word string, dur time.Duration) *run.Run {
	t.Helper()
	base := time.Unix(0, 0)
	r := run.New(run.ModeZen, 0)
	// Clock: base for the first Type (sets start), then dur for every later read.
	first := true
	r.Now = func() time.Time {
		if first {
			first = false
			return base
		}
		return base.Add(dur)
	}
	r.AppendWords([]string{word})
	for _, ch := range word {
		r.Type(ch)
	}
	return r
}

func TestComputeNetWPM(t *testing.T) {
	// 10 correct chars over 60s => 10/5 / 1min = 2 WPM.
	res := Compute(buildRun(t, "abcdefghij", 60*time.Second))
	if res.NetWPM != 2.0 {
		t.Errorf("NetWPM: got %v, want 2.0", res.NetWPM)
	}
	if res.Accuracy != 1.0 {
		t.Errorf("Accuracy: got %v, want 1.0", res.Accuracy)
	}
}

func TestComputeAccuracyWithErrors(t *testing.T) {
	base := time.Unix(0, 0)
	r := run.New(run.ModeZen, 0)
	r.Now = func() time.Time { return base }
	r.AppendWords([]string{"cat"})
	r.Type('c')
	r.Type('x') // wrong (expected 'a')
	r.Type('t')
	res := Compute(r)
	if res.Accuracy != 2.0/3.0 {
		t.Errorf("Accuracy: got %v, want %v", res.Accuracy, 2.0/3.0)
	}
}

func TestComputeMissedChars(t *testing.T) {
	base := time.Unix(0, 0)
	r := run.New(run.ModeZen, 0)
	r.Now = func() time.Time { return base }
	r.AppendWords([]string{"aaa"})
	r.Type('a')
	r.Type('x') // miss on 'a'
	r.Type('y') // miss on 'a'
	res := Compute(r)
	if len(res.MissedChars) == 0 || res.MissedChars[0].Char != 'a' {
		t.Fatalf("expected 'a' as top missed char, got %v", res.MissedChars)
	}
	if res.MissedChars[0].Count != 2 {
		t.Errorf("miss count: got %d, want 2", res.MissedChars[0].Count)
	}
}

func TestComputeEmptyRunIsSafe(t *testing.T) {
	res := Compute(run.New(run.ModeZen, 0))
	if res.NetWPM != 0 || res.Accuracy != 0 {
		t.Errorf("empty run should be all zero, got %+v", res)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/stats/`
Expected: FAIL — `undefined: Compute`.

- [ ] **Step 3: Implement Compute**

Create `internal/stats/stats.go`:
```go
// Package stats computes post-run metrics and renders ASCII charts.
package stats

import (
	"math"
	"sort"
	"time"

	"github.com/justin06lee/shaw/internal/run"
)

// CharCount pairs an expected character with how often it was mistyped.
type CharCount struct {
	Char  rune
	Count int
}

// Result is the full set of metrics for one finished run.
type Result struct {
	NetWPM      float64
	RawWPM      float64
	Accuracy    float64     // correct keystrokes / total keystrokes, 0..1
	Consistency float64     // 0..100, higher = steadier
	Samples     []float64   // cumulative net WPM sampled per second
	MissedChars []CharCount // most-mistyped characters, descending
	Mode        run.Mode
	Target      int
}

// Compute derives all metrics from a finished run.
func Compute(r *run.Run) Result {
	log := r.Log()
	res := Result{Mode: r.Mode(), Target: r.Target()}

	var typed, correct, correctChars int
	misses := map[rune]int{}
	for _, k := range log {
		if k.Backspace {
			continue
		}
		typed++
		if k.Correct {
			correct++
			correctChars++
		} else {
			misses[k.Expected]++
		}
	}
	if typed == 0 {
		return res
	}

	mins := r.Duration().Minutes()
	if mins > 0 {
		res.RawWPM = float64(typed) / 5 / mins
		res.NetWPM = float64(correctChars) / 5 / mins
	}
	res.Accuracy = float64(correct) / float64(typed)
	res.Samples = perSecondWPM(log, r.Duration())
	res.Consistency = consistency(res.Samples)
	res.MissedChars = rankMisses(misses)
	return res
}

// perSecondWPM returns cumulative net WPM at the end of each elapsed second.
func perSecondWPM(log []run.Keystroke, dur time.Duration) []float64 {
	secs := int(math.Ceil(dur.Seconds()))
	if secs < 1 {
		return nil
	}
	out := make([]float64, secs)
	for i := 1; i <= secs; i++ {
		cutoff := time.Duration(i) * time.Second
		correct := 0
		for _, k := range log {
			if k.Backspace || !k.Correct || k.At > cutoff {
				continue
			}
			correct++
		}
		out[i-1] = float64(correct) / 5 / (float64(i) / 60)
	}
	return out
}

// consistency is 100*(1 - coefficient of variation) of the samples, clamped.
func consistency(samples []float64) float64 {
	if len(samples) < 2 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		sum += s
	}
	mean := sum / float64(len(samples))
	if mean == 0 {
		return 0
	}
	var variance float64
	for _, s := range samples {
		variance += (s - mean) * (s - mean)
	}
	variance /= float64(len(samples))
	cv := math.Sqrt(variance) / mean
	c := 100 * (1 - cv)
	if c < 0 {
		return 0
	}
	return c
}

// rankMisses sorts mistyped characters by count, descending, capped at 5.
func rankMisses(misses map[rune]int) []CharCount {
	out := make([]CharCount, 0, len(misses))
	for ch, n := range misses {
		out = append(out, CharCount{Char: ch, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Char < out[j].Char
	})
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/stats/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/stats/
git commit -m "feat: run metrics computation in stats package"
```

---

## Task 7: stats.RenderChart — ASCII WPM chart

**Files:**
- Modify: `internal/stats/stats.go`
- Test: `internal/stats/stats_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/stats/stats_test.go`:
```go
import "strings"  // add to the existing import block

func TestRenderChartDimensions(t *testing.T) {
	out := RenderChart([]float64{1, 5, 3, 8, 2}, 20, 5)
	lines := strings.Split(out, "\n")
	if len(lines) != 5 {
		t.Fatalf("got %d lines, want 5", len(lines))
	}
	for i, ln := range lines {
		if len([]rune(ln)) != 20 {
			t.Errorf("line %d width: got %d, want 20", i, len([]rune(ln)))
		}
	}
}

func TestRenderChartEmptyIsBlank(t *testing.T) {
	out := RenderChart(nil, 10, 3)
	for _, ln := range strings.Split(out, "\n") {
		if strings.TrimSpace(ln) != "" {
			t.Errorf("expected blank chart, got %q", ln)
		}
	}
}

func TestRenderChartPlotsPeak(t *testing.T) {
	// The tallest bar should reach the top row.
	out := RenderChart([]float64{0, 10, 0}, 3, 4)
	top := strings.Split(out, "\n")[0]
	if !strings.Contains(top, "█") {
		t.Errorf("peak not plotted in top row: %q", top)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/stats/`
Expected: FAIL — `undefined: RenderChart`.

- [ ] **Step 3: Implement RenderChart**

Append to `internal/stats/stats.go`:
```go
import "strings"  // add to the existing import block

// RenderChart draws samples as a vertical bar chart of the given width and
// height (in terminal cells). Samples are resampled to fit width columns.
func RenderChart(samples []float64, width, height int) string {
	rows := make([][]rune, height)
	for i := range rows {
		rows[i] = []rune(strings.Repeat(" ", width))
	}
	if len(samples) > 0 && width > 0 && height > 0 {
		max := 0.0
		for _, s := range samples {
			if s > max {
				max = s
			}
		}
		if max > 0 {
			for col := 0; col < width; col++ {
				idx := col * len(samples) / width
				barHeight := int(math.Round(samples[idx] / max * float64(height)))
				for row := 0; row < barHeight && row < height; row++ {
					rows[height-1-row][col] = '█'
				}
			}
		}
	}
	out := make([]string, height)
	for i, r := range rows {
		out[i] = string(r)
	}
	return strings.Join(out, "\n")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/stats/`
Expected: PASS (all stats tests).

- [ ] **Step 5: Commit**

```bash
git add internal/stats/
git commit -m "feat: ASCII WPM bar chart in stats package"
```

---

## Task 8: history package — JSON persistence

**Files:**
- Create: `internal/history/history.go`
- Test: `internal/history/history_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/history/history_test.go`:
```go
package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPathUsesXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdgtest")
	got, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/tmp/xdgtest", "shaw", "history.json")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAppendThenLoadRoundTrips(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	rec := Record{
		Time: time.Unix(1000, 0).UTC(), Mode: "time", Target: 30,
		NetWPM: 55.5, RawWPM: 60, Accuracy: 0.95, Consistency: 80,
	}
	if err := Append(rec); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].NetWPM != 55.5 || got[0].Mode != "time" {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestAppendAccumulates(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_ = Append(Record{Mode: "words", Target: 25})
	_ = Append(Record{Mode: "zen", Target: 0})
	got, _ := Load()
	if len(got) != 2 {
		t.Fatalf("got %d records, want 2", len(got))
	}
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestLoadCorruptFileIsEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	p := filepath.Join(dir, "shaw", "history.json")
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte("{not json"), 0o644)
	got, err := Load()
	if err != nil {
		t.Fatalf("corrupt file should not error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty on corrupt file, got %v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/history/`
Expected: FAIL — `undefined: Path`.

- [ ] **Step 3: Implement history**

Create `internal/history/history.go`:
```go
// Package history persists typing-run results as a JSON file.
package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Record is one persisted run result.
type Record struct {
	Time        time.Time `json:"time"`
	Mode        string    `json:"mode"`
	Target      int       `json:"target"`
	NetWPM      float64   `json:"net_wpm"`
	RawWPM      float64   `json:"raw_wpm"`
	Accuracy    float64   `json:"accuracy"`
	Consistency float64   `json:"consistency"`
}

// Path returns the history file location, honoring XDG_CONFIG_HOME.
func Path() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "shaw", "history.json"), nil
}

// Load reads all records. A missing or corrupt file yields an empty slice and
// no error, so a damaged history never blocks a run.
func Load() ([]Record, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var recs []Record
	if err := json.Unmarshal(data, &recs); err != nil {
		return nil, nil // tolerate corruption
	}
	return recs, nil
}

// Append adds rec to the history file, creating it and its directory if needed.
func Append(rec Record) error {
	p, err := Path()
	if err != nil {
		return err
	}
	recs, err := Load()
	if err != nil {
		return err
	}
	recs = append(recs, rec)
	data, err := json.MarshalIndent(recs, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/history/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/history/
git commit -m "feat: JSON run-history persistence"
```

---

## Task 9: tui layout helpers — WrapLines and Viewport

**Files:**
- Create: `internal/tui/wrap.go`
- Test: `internal/tui/wrap_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tui/wrap_test.go`:
```go
package tui

import "testing"

func TestWrapLinesBreaksOnWordBoundary(t *testing.T) {
	// "aaa bbb ccc" width 7 => "aaa bbb" (7) then "ccc".
	text := []rune("aaa bbb ccc")
	lines := WrapLines(text, 7)
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %+v", len(lines), lines)
	}
	if string(text[lines[0].Start:lines[0].End]) != "aaa bbb" {
		t.Errorf("line 0: got %q", string(text[lines[0].Start:lines[0].End]))
	}
	if string(text[lines[1].Start:lines[1].End]) != "ccc" {
		t.Errorf("line 1: got %q", string(text[lines[1].Start:lines[1].End]))
	}
}

func TestWrapLinesSingleLineFits(t *testing.T) {
	lines := WrapLines([]rune("short"), 40)
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1", len(lines))
	}
}

func TestLineOfCursor(t *testing.T) {
	lines := WrapLines([]rune("aaa bbb ccc"), 7) // line0 [0,7), line1 [8,11)
	if got := LineOfCursor(lines, 2); got != 0 {
		t.Errorf("cursor 2: got line %d, want 0", got)
	}
	if got := LineOfCursor(lines, 9); got != 1 {
		t.Errorf("cursor 9: got line %d, want 1", got)
	}
	if got := LineOfCursor(lines, 11); got != 1 {
		t.Errorf("cursor at end: got line %d, want 1", got)
	}
}

func TestViewportReturnsThreeLinesCentered(t *testing.T) {
	lines := []Line{{0, 1}, {1, 2}, {2, 3}, {3, 4}, {4, 5}}
	start, count := Viewport(lines, 2) // cursor on line 2 => window 1..3
	if start != 1 || count != 3 {
		t.Errorf("got start=%d count=%d, want 1,3", start, count)
	}
}

func TestViewportClampsAtTop(t *testing.T) {
	lines := []Line{{0, 1}, {1, 2}, {2, 3}}
	start, count := Viewport(lines, 0)
	if start != 0 || count != 3 {
		t.Errorf("got start=%d count=%d, want 0,3", start, count)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/`
Expected: FAIL — `undefined: WrapLines`.

- [ ] **Step 3: Implement wrap helpers**

Create `internal/tui/wrap.go`:
```go
// Package tui renders the shaw terminal interface with bubbletea.
package tui

// Line is a half-open rune-index range [Start,End) into the target text.
type Line struct {
	Start, End int
}

// WrapLines breaks text into lines no wider than width, splitting on spaces.
// A word longer than width occupies its own (over-wide) line.
func WrapLines(text []rune, width int) []Line {
	if width < 1 {
		width = 1
	}
	var lines []Line
	lineStart := 0
	lastSpace := -1
	for i := 0; i < len(text); i++ {
		if text[i] == ' ' {
			lastSpace = i
		}
		if i-lineStart >= width {
			if lastSpace > lineStart {
				lines = append(lines, Line{lineStart, lastSpace})
				lineStart = lastSpace + 1
			} else {
				lines = append(lines, Line{lineStart, i})
				lineStart = i
			}
			lastSpace = -1
		}
	}
	if lineStart < len(text) || len(lines) == 0 {
		lines = append(lines, Line{lineStart, len(text)})
	}
	return lines
}

// LineOfCursor returns the index of the line containing cursor (an index into
// the text). A cursor at the very end maps to the last line.
func LineOfCursor(lines []Line, cursor int) int {
	for i, ln := range lines {
		if cursor >= ln.Start && cursor < ln.End {
			return i
		}
	}
	if len(lines) == 0 {
		return 0
	}
	return len(lines) - 1
}

// Viewport returns the start line index and count (<=3) for a 3-line window
// that keeps the cursor's line centered, clamped to the available lines.
func Viewport(lines []Line, cursorLine int) (start, count int) {
	const window = 3
	if len(lines) <= window {
		return 0, len(lines)
	}
	start = cursorLine - 1
	if start < 0 {
		start = 0
	}
	if start+window > len(lines) {
		start = len(lines) - window
	}
	return start, window
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/
git commit -m "feat: text-wrapping and viewport helpers for tui"
```

---

## Task 10: tui.Model — config bar, state machine, input routing

**Files:**
- Create: `internal/tui/model.go`
- Test: `internal/tui/model_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tui/model_test.go`:
```go
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/justin06lee/shaw/internal/run"
)

// fixedSource is a WordSource that returns the same word forever, so model
// tests need no filesystem.
type fixedSource struct{ word string }

func (f fixedSource) Next() (string, bool) { return f.word, true }

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func newModel() Model {
	return New(fixedSource{word: "alpha"}, run.ModeTime, 30, 80, 24)
}

func TestModelStartsIdle(t *testing.T) {
	m := newModel()
	if m.State() != StateIdle {
		t.Errorf("got %v, want StateIdle", m.State())
	}
}

func TestConfigBarChangesModeWhenIdle(t *testing.T) {
	m := newModel()
	// Right arrow on the mode control moves time -> words.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(Model)
	if m.Mode() != run.ModeWords {
		t.Errorf("got mode %v, want ModeWords", m.Mode())
	}
}

func TestTypingTransitionsToActive(t *testing.T) {
	m := newModel()
	updated, _ := m.Update(keyMsg("a")) // first char of "alpha"
	m = updated.(Model)
	if m.State() != StateActive {
		t.Errorf("got %v, want StateActive", m.State())
	}
}

func TestConfigBarIgnoredWhileActive(t *testing.T) {
	m := newModel()
	updated, _ := m.Update(keyMsg("a")) // -> active
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight}) // arrow ignored
	m = updated.(Model)
	if m.Mode() != run.ModeTime {
		t.Errorf("mode changed during active run: got %v", m.Mode())
	}
}

func TestEscDuringActiveReturnsToIdle(t *testing.T) {
	m := newModel()
	updated, _ := m.Update(keyMsg("a"))
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.State() != StateIdle {
		t.Errorf("got %v, want StateIdle after Esc", m.State())
	}
}

func TestWordsRunReachingGoalShowsResult(t *testing.T) {
	// 1-word target "alpha": typing all 5 chars correctly hits the goal.
	m := New(fixedSource{word: "alpha"}, run.ModeWords, 1, 80, 24)
	for _, ch := range "alpha" {
		updated, _ := m.Update(keyMsg(string(ch)))
		m = updated.(Model)
	}
	if m.State() != StateResult {
		t.Errorf("got %v, want StateResult", m.State())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/`
Expected: FAIL — `undefined: New`.

- [ ] **Step 3: Implement the Model**

Create `internal/tui/model.go`:
```go
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/justin06lee/shaw/internal/run"
	"github.com/justin06lee/shaw/internal/stats"
)

// WordSource yields words for a run. corpus.TextStream satisfies it.
type WordSource interface {
	Next() (string, bool)
}

// State is the top-level screen the model is showing.
type State int

const (
	StateIdle   State = iota // config bar editable, waiting for first keystroke
	StateActive              // run in progress, typing captured
	StateResult              // finished run, showing metrics
)

// mode/target option tables for the config bar.
var modeOrder = []run.Mode{run.ModeTime, run.ModeWords, run.ModeZen}
var targetOptions = map[run.Mode][]int{
	run.ModeTime:  {15, 30, 60, 120},
	run.ModeWords: {10, 25, 50, 100},
	run.ModeZen:   {0},
}

// topUpThreshold: when fewer than this many chars remain past the cursor in a
// time/zen run, more words are pulled from the source.
const topUpThreshold = 120

// Model is the bubbletea model for shaw.
type Model struct {
	src    WordSource
	state  State
	width  int
	height int

	modeIdx   int // index into modeOrder
	targetIdx int // index into targetOptions[current mode]
	barFocus  int // 0 = mode control, 1 = target control

	run    *run.Run
	result stats.Result
}

// New builds a Model in the idle state with the given initial mode/target.
func New(src WordSource, mode run.Mode, target, width, height int) Model {
	m := Model{src: src, state: StateIdle, width: width, height: height}
	for i, md := range modeOrder {
		if md == mode {
			m.modeIdx = i
		}
	}
	opts := targetOptions[mode]
	for i, tv := range opts {
		if tv == target {
			m.targetIdx = i
		}
	}
	m.resetRun()
	return m
}

// Init satisfies tea.Model. The timer tick is started lazily on first keystroke.
func (m Model) Init() tea.Cmd { return nil }

// Accessors used by tests and the view.
func (m Model) State() State    { return m.state }
func (m Model) Mode() run.Mode  { return modeOrder[m.modeIdx] }
func (m Model) Target() int     { return targetOptions[m.Mode()][m.targetIdx] }
func (m Model) Run() *run.Run   { return m.run }
func (m Model) Result() stats.Result { return m.result }

// resetRun builds a fresh run for the current mode/target and pre-fills text.
func (m *Model) resetRun() {
	m.run = run.New(m.Mode(), m.Target())
	m.state = StateIdle
	if m.Mode() == run.ModeWords {
		m.pullWords(m.Target())
	} else {
		m.pullWords(60) // initial buffer for time/zen
	}
}

// pullWords appends n words from the source to the run.
func (m *Model) pullWords(n int) {
	words := make([]string, 0, n)
	for i := 0; i < n; i++ {
		w, ok := m.src.Next()
		if !ok {
			break
		}
		words = append(words, w)
	}
	m.run.AppendWords(words)
}

// Update routes input based on the current state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}
	switch m.state {
	case StateIdle:
		return m.handleIdleKey(msg)
	case StateActive:
		return m.handleActiveKey(msg)
	default: // StateResult
		if msg.Type == tea.KeyEsc || msg.Type == tea.KeyEnter {
			m.resetRun()
		}
		return m, nil
	}
}

// handleIdleKey edits the config bar or starts a run on the first typed rune.
func (m Model) handleIdleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		m.barFocus = 1 - m.barFocus
		return m, nil
	case tea.KeyLeft:
		m.moveBar(-1)
		return m, nil
	case tea.KeyRight:
		m.moveBar(1)
		return m, nil
	case tea.KeyEsc:
		m.resetRun() // fresh text
		return m, nil
	case tea.KeyRunes, tea.KeySpace:
		m.state = StateActive
		return m.handleActiveKey(msg)
	}
	return m, nil
}

// moveBar shifts the focused config control by delta and regenerates the run.
func (m *Model) moveBar(delta int) {
	if m.barFocus == 0 {
		m.modeIdx = wrap(m.modeIdx+delta, len(modeOrder))
		m.targetIdx = 0
	} else {
		m.targetIdx = wrap(m.targetIdx+delta, len(targetOptions[m.Mode()]))
	}
	m.resetRun()
}

func wrap(i, n int) int {
	return ((i % n) + n) % n
}

// handleActiveKey feeds typing into the run and checks for completion.
func (m Model) handleActiveKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.resetRun() // abort, no stats saved
		return m, nil
	case tea.KeyBackspace:
		m.run.Backspace()
		return m, nil
	case tea.KeySpace:
		m.run.Type(' ')
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			m.run.Type(r)
		}
	default:
		return m, nil
	}
	// Top up the word buffer for time/zen runs as the cursor nears the end.
	if m.Mode() != run.ModeWords &&
		len(m.run.Text())-m.run.Cursor() < topUpThreshold {
		m.pullWords(40)
	}
	if m.run.GoalReached() {
		m.finish()
	}
	return m, nil
}

// finish computes stats and moves to the result screen.
func (m *Model) finish() {
	m.result = stats.Compute(m.run)
	m.state = StateResult
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/`
Expected: PASS (all tui tests).

- [ ] **Step 5: Commit**

```bash
git add internal/tui/
git commit -m "feat: tui Model state machine and config bar"
```

---

## Task 11: tui timer tick + view rendering

**Files:**
- Modify: `internal/tui/model.go`
- Create: `internal/tui/view.go`
- Test: `internal/tui/model_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/tui/model_test.go`:
```go
import "time"  // add to the existing import block

func TestTickFinishesExpiredTimeRun(t *testing.T) {
	m := New(fixedSource{word: "alpha"}, run.ModeTime, 30, 80, 24)
	updated, _ := m.Update(keyMsg("a")) // start the run
	m = updated.(Model)
	// Force the run's clock so the goal is met, then deliver a tick.
	m.Run().Now = func() time.Time { return time.Now().Add(time.Hour) }
	updated, _ = m.Update(tickMsg(time.Now()))
	m = updated.(Model)
	if m.State() != StateResult {
		t.Errorf("got %v, want StateResult after expiry tick", m.State())
	}
}

func TestViewRendersWithoutPanic(t *testing.T) {
	m := newModel()
	if m.View() == "" {
		t.Error("idle view is empty")
	}
	updated, _ := m.Update(keyMsg("a"))
	m = updated.(Model)
	if m.View() == "" {
		t.Error("active view is empty")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/`
Expected: FAIL — `undefined: tickMsg` and `m.View undefined`.

- [ ] **Step 3: Add the timer tick and View**

Append to `internal/tui/model.go`:
```go
import "time"  // add to the existing import block

// tickMsg drives the per-second footer/timer while a run is active.
type tickMsg time.Time

// tick schedules the next timer message one second out.
func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}
```

Modify `Update` in `internal/tui/model.go` — add a `tickMsg` case to the
existing `switch msg := msg.(type)` block, before the `default`:
```go
	case tickMsg:
		if m.state == StateActive && m.run.GoalReached() {
			m.finish()
			return m, nil
		}
		if m.state == StateActive {
			return m, tick()
		}
		return m, nil
```

Modify `handleIdleKey` — in the `case tea.KeyRunes, tea.KeySpace:` branch,
return a tick command so the timer starts when typing begins:
```go
	case tea.KeyRunes, tea.KeySpace:
		m.state = StateActive
		next, _ := m.handleActiveKey(msg)
		return next, tick()
```

Create `internal/tui/view.go`:
```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/justin06lee/shaw/internal/run"
	"github.com/justin06lee/shaw/internal/stats"
)

const minWidth = 40

var (
	styleDim      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	styleCorrect  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleWrong    = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	styleActive   = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	styleInactive = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

// View renders the whole screen for the current state.
func (m Model) View() string {
	if m.width < minWidth {
		return "terminal too narrow — widen to at least 40 columns"
	}
	switch m.state {
	case StateResult:
		return m.resultView()
	default:
		return m.typingView()
	}
}

// typingView renders the config bar, three-line text area, and footer.
func (m Model) typingView() string {
	bar := m.configBar()
	text := m.textArea()
	footer := m.footer()
	return strings.Join([]string{"", bar, "", text, "", footer}, "\n")
}

// configBar renders the mode and target segmented controls.
func (m Model) configBar() string {
	dim := m.state == StateActive
	modes := []string{"time", "words", "zen"}
	var parts []string
	for i, name := range modes {
		parts = append(parts, segment(name, i == m.modeIdx, dim, m.barFocus == 0))
	}
	modeCtl := strings.Join(parts, " ")

	var tparts []string
	for i, tv := range targetOptions[m.Mode()] {
		if m.Mode() == run.ModeZen {
			continue
		}
		tparts = append(tparts, segment(fmt.Sprintf("%d", tv),
			i == m.targetIdx, dim, m.barFocus == 1))
	}
	targetCtl := strings.Join(tparts, " ")
	return "  " + modeCtl + "    " + targetCtl
}

// segment styles one config-bar option.
func segment(label string, selected, dim, focused bool) string {
	st := styleInactive
	if selected {
		st = styleActive
	}
	if dim {
		st = styleDim
	}
	if focused && selected && !dim {
		label = "[" + label + "]"
	}
	return st.Render(label)
}

// textArea renders the 3-line scrolling viewport of the target text.
func (m Model) textArea() string {
	text := m.run.Text()
	width := m.width - 4
	lines := WrapLines(text, width)
	cursorLine := LineOfCursor(lines, m.run.Cursor())
	start, count := Viewport(lines, cursorLine)
	states := m.run.States()
	cursor := m.run.Cursor()

	var out []string
	for i := start; i < start+count; i++ {
		ln := lines[i]
		var b strings.Builder
		for j := ln.Start; j < ln.End; j++ {
			ch := string(text[j])
			switch {
			case j == cursor:
				b.WriteString(styleActive.Render(ch))
			case states[j] == run.Correct:
				b.WriteString(styleCorrect.Render(ch))
			case states[j] == run.Incorrect:
				b.WriteString(styleWrong.Render(ch))
			default:
				b.WriteString(styleDim.Render(ch))
			}
		}
		out = append(out, "  "+b.String())
	}
	return strings.Join(out, "\n")
}

// footer renders the live progress indicator and key hints.
func (m Model) footer() string {
	var status string
	switch m.Mode() {
	case run.ModeTime:
		status = fmt.Sprintf("%ds", m.Target())
	case run.ModeWords:
		status = fmt.Sprintf("%d words", m.Target())
	default:
		status = "zen"
	}
	return styleDim.Render("  " + status + "   ·   esc to restart")
}

// resultView renders metrics, the WPM chart, and the error breakdown.
func (m Model) resultView() string {
	r := m.result
	var b strings.Builder
	b.WriteString(styleActive.Render(
		fmt.Sprintf("\n  %.0f wpm   %.0f%% acc\n", r.NetWPM, r.Accuracy*100)))
	b.WriteString(styleDim.Render(fmt.Sprintf(
		"  raw %.0f   consistency %.0f%%\n\n", r.RawWPM, r.Consistency)))
	chart := stats.RenderChart(r.Samples, m.width-4, 8)
	for _, ln := range strings.Split(chart, "\n") {
		b.WriteString("  " + styleCorrect.Render(ln) + "\n")
	}
	b.WriteString("\n")
	if len(r.MissedChars) > 0 {
		var parts []string
		for _, mc := range r.MissedChars {
			parts = append(parts, fmt.Sprintf("%q×%d", mc.Char, mc.Count))
		}
		b.WriteString(styleDim.Render("  missed: " + strings.Join(parts, "  ") + "\n"))
	}
	b.WriteString(styleDim.Render("  saved to history   ·   enter for a new run\n"))
	return b.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/`
Expected: PASS (all tui tests).

- [ ] **Step 5: Commit**

```bash
git add internal/tui/
git commit -m "feat: timer tick and full view rendering for tui"
```

---

## Task 12: history save on finish + main.go wiring

**Files:**
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/model_test.go`
- Rewrite: `main.go`

- [ ] **Step 1: Write the failing test for history-on-finish**

Append to `internal/tui/model_test.go`:
```go
import (
	"path/filepath"   // add to the existing import block
	"github.com/justin06lee/shaw/internal/history"
)

func TestFinishWritesHistory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New(fixedSource{word: "alpha"}, run.ModeWords, 1, 80, 24)
	for _, ch := range "alpha" {
		updated, _ := m.Update(keyMsg(string(ch)))
		m = updated.(Model)
	}
	if m.State() != StateResult {
		t.Fatalf("run did not finish: state %v", m.State())
	}
	recs, err := history.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d history records, want 1", len(recs))
	}
	_ = filepath.Separator // keep filepath import used if trimmed
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/`
Expected: FAIL — `TestFinishWritesHistory` finds 0 records.

- [ ] **Step 3: Save history in finish()**

Modify the `finish` method in `internal/tui/model.go` to append a history
record. Add the `history` import and replace the method body:
```go
import "github.com/justin06lee/shaw/internal/history"  // add to import block

// finish computes stats, persists the result, and shows the result screen.
func (m *Model) finish() {
	m.result = stats.Compute(m.run)
	m.state = StateResult
	modeName := map[run.Mode]string{
		run.ModeTime: "time", run.ModeWords: "words", run.ModeZen: "zen",
	}[m.Mode()]
	_ = history.Append(history.Record{
		Time:        time.Now(),
		Mode:        modeName,
		Target:      m.Target(),
		NetWPM:      m.result.NetWPM,
		RawWPM:      m.result.RawWPM,
		Accuracy:    m.result.Accuracy,
		Consistency: m.result.Consistency,
	})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/`
Expected: PASS.

- [ ] **Step 5: Write main.go**

Rewrite `main.go`:
```go
// Command shaw is a monkeytype-style terminal typing trainer.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/justin06lee/shaw/internal/corpus"
	"github.com/justin06lee/shaw/internal/history"
	"github.com/justin06lee/shaw/internal/run"
	"github.com/justin06lee/shaw/internal/stats"
	"github.com/justin06lee/shaw/internal/tui"
)

func main() {
	timeFlag := flag.Int("time", 0, "timed mode: 15, 30, 60, or 120 seconds")
	wordsFlag := flag.Int("words", 0, "word mode: 10, 25, 50, or 100 words")
	zenFlag := flag.Bool("zen", false, "zen mode: type until Esc")
	histFlag := flag.Bool("history", false, "print progress chart and exit")
	flag.Parse()

	if *histFlag {
		printHistory()
		return
	}

	mode, target := run.ModeTime, 30
	switch {
	case *zenFlag:
		mode, target = run.ModeZen, 0
	case *wordsFlag > 0:
		mode, target = run.ModeWords, *wordsFlag
	case *timeFlag > 0:
		mode, target = run.ModeTime, *timeFlag
	}

	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}
	files, err := corpus.Scan(dir)
	if err != nil {
		fail("cannot scan %s: %v", dir, err)
	}
	if len(files) == 0 {
		fail("no .txt files found in %s", dir)
	}

	stream := corpus.NewTextStream(files, rand.New(rand.NewSource(time.Now().UnixNano())))
	if _, ok := stream.Next(); !ok {
		fail("no usable text in %s (files empty or not UTF-8)", dir)
	}
	stream = corpus.NewTextStream(files, rand.New(rand.NewSource(time.Now().UnixNano())))

	m := tui.New(stream, mode, target, 80, 24)
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fail("%v", err)
	}
}

// printHistory renders an ASCII chart of net WPM across past runs.
func printHistory() {
	recs, err := history.Load()
	if err != nil {
		fail("cannot read history: %v", err)
	}
	if len(recs) == 0 {
		fmt.Println("no runs recorded yet")
		return
	}
	samples := make([]float64, len(recs))
	for i, r := range recs {
		samples[i] = r.NetWPM
	}
	fmt.Printf("net wpm across %d runs:\n\n", len(recs))
	fmt.Println(stats.RenderChart(samples, 60, 12))
	fmt.Printf("\nlatest: %.0f wpm\n", samples[len(samples)-1])
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "shaw: "+format+"\n", args...)
	os.Exit(1)
}
```

- [ ] **Step 6: Verify the whole build and test suite**

Run: `go build ./... && go test ./...`
Expected: build succeeds; every package reports `ok`.

- [ ] **Step 7: Manual smoke test**

Run:
```bash
mkdir -p /tmp/shawtest && printf 'the quick brown fox jumps over the lazy dog\n' > /tmp/shawtest/a.txt
go run . /tmp/shawtest --time 15
```
Expected: the typing screen opens; typing works; after 15s a result screen with
a WPM chart appears; `Ctrl-C` exits. Then `go run . --history` prints a chart.

- [ ] **Step 8: Commit**

```bash
git add internal/tui/ main.go
git commit -m "feat: persist run history and wire up shaw CLI"
```

---

## Self-Review Notes

**Spec coverage check:**
- CLI flags (`--time`/`--words`/`--zen`/`--history`, folder default cwd) → Task 12.
- Monkeytype-style top config bar, editable when idle, dimmed when active → Tasks 10 (state machine) + 11 (rendering).
- 3-line scrolling word-wrapped viewport → Tasks 9 + 11.
- Monkeytype error handling (red char, continue, backspace) → Task 4.
- Run modes time/words/zen with goal detection → Task 5; mode/target option tables → Task 10.
- Random `.txt` file, top-to-bottom, newlines→spaces, rollover on exhaust → Tasks 2 + 3.
- WPM (net/raw), accuracy, consistency, error breakdown, in-run chart → Tasks 6 + 7.
- Cross-run history persistence + `--history` chart → Tasks 8 + 12.
- Error cases (empty folder, non-UTF8, narrow terminal, Ctrl-C) → Tasks 3, 11, 12.

**Type consistency check:** `WordSource.Next()` matches `corpus.TextStream.Next()` signature `(string, bool)`. `run.Mode`/`run.CharState` enums used consistently across `run`, `stats`, `tui`. `stats.Result` fields referenced in `view.go` match Task 6 definitions. `history.Record` fields match between Tasks 8 and 12.

**No placeholders:** every step contains complete, compilable code.
