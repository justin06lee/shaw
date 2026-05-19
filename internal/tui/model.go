package tui

import (
	"time"

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
	target    int // current target value (seconds or words)
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
	m.target = target
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
func (m Model) State() State         { return m.state }
func (m Model) Mode() run.Mode       { return modeOrder[m.modeIdx] }
func (m Model) Target() int          { return m.target }
func (m Model) Run() *run.Run        { return m.run }
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
	case tickMsg:
		if m.state == StateActive && m.run.GoalReached() {
			m.finish()
			return m, nil
		}
		if m.state == StateActive {
			return m, tick()
		}
		return m, nil
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
		next, cmd := m.handleActiveKey(msg)
		return next, tea.Batch(cmd, tick())
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
	m.target = targetOptions[m.Mode()][m.targetIdx]
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

// tickMsg drives the per-second footer/timer while a run is active.
type tickMsg time.Time

// tick schedules the next timer message one second out.
func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}
