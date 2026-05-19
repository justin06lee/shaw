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

// Elapsed is the time since the first keystroke, or 0 before typing starts.
func (r *Run) Elapsed() time.Duration { return r.elapsed() }

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
