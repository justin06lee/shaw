package stats

import (
	"strings"
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

func TestConsistencySteadyIsHigh(t *testing.T) {
	if got := consistency([]float64{10, 10, 10}); got != 100 {
		t.Errorf("steady samples: got %v, want 100", got)
	}
}

func TestConsistencyVariableIsLow(t *testing.T) {
	// mean 10, stddev 10 => CV 1 => 100*(1-1) = 0.
	if got := consistency([]float64{0, 20}); got != 0 {
		t.Errorf("variable samples: got %v, want 0", got)
	}
}

func TestConsistencyTooFewSamples(t *testing.T) {
	if got := consistency([]float64{42}); got != 0 {
		t.Errorf("single sample: got %v, want 0", got)
	}
	if got := consistency(nil); got != 0 {
		t.Errorf("nil samples: got %v, want 0", got)
	}
}

func TestPerSecondWPMCumulative(t *testing.T) {
	// Two correct keystrokes: one at 0.5s, one at 1.5s, run lasts 2s.
	log := []run.Keystroke{
		{At: 500 * time.Millisecond, Correct: true},
		{At: 1500 * time.Millisecond, Correct: true},
	}
	got := perSecondWPM(log, 2*time.Second)
	if len(got) != 2 {
		t.Fatalf("got %d samples, want 2", len(got))
	}
	// Second 1: 1 correct char => (1/5)/(1/60) = 12 WPM.
	if got[0] != 12 {
		t.Errorf("sample 0: got %v, want 12", got[0])
	}
	// Second 2: 2 correct chars cumulative => (2/5)/(2/60) = 12 WPM.
	if got[1] != 12 {
		t.Errorf("sample 1: got %v, want 12", got[1])
	}
}

func TestPerSecondInstantWPM(t *testing.T) {
	// One correct keystroke in second 1, two in second 2.
	log := []run.Keystroke{
		{At: 200 * time.Millisecond, Correct: true},
		{At: 1200 * time.Millisecond, Correct: true},
		{At: 1800 * time.Millisecond, Correct: true},
	}
	got := perSecondInstantWPM(log, 2*time.Second)
	if len(got) != 2 {
		t.Fatalf("got %d samples, want 2", len(got))
	}
	// Second 1: 1 char => 1/5*60 = 12 WPM.
	if got[0] != 12 {
		t.Errorf("instant sample 0: got %v, want 12", got[0])
	}
	// Second 2: 2 chars => 2/5*60 = 24 WPM.
	if got[1] != 24 {
		t.Errorf("instant sample 1: got %v, want 24", got[1])
	}
}

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
