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
