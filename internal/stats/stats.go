// Package stats computes post-run metrics and renders ASCII charts.
package stats

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/justin06lee/shaw/internal/run"
)

// missCap is the maximum number of mistyped characters reported.
const missCap = 5

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

	var typed, correct int
	misses := map[rune]int{}
	for _, k := range log {
		if k.Backspace {
			continue
		}
		typed++
		if k.Correct {
			correct++
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
		res.NetWPM = float64(correct) / 5 / mins
	}
	res.Accuracy = float64(correct) / float64(typed)
	res.Samples = perSecondWPM(log, r.Duration())
	res.Consistency = consistency(perSecondInstantWPM(log, r.Duration()))
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

// perSecondInstantWPM returns the net WPM achieved within each individual
// elapsed second (not cumulative). Used for the consistency metric.
func perSecondInstantWPM(log []run.Keystroke, dur time.Duration) []float64 {
	secs := int(math.Ceil(dur.Seconds()))
	if secs < 1 {
		return nil
	}
	counts := make([]int, secs)
	for _, k := range log {
		if k.Backspace || !k.Correct {
			continue
		}
		b := int(k.At.Seconds())
		if b >= secs {
			b = secs - 1
		}
		if b < 0 {
			b = 0
		}
		counts[b]++
	}
	out := make([]float64, secs)
	for i, c := range counts {
		out[i] = float64(c) / 5 * 60 // correct chars in this second -> WPM
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

// rankMisses sorts mistyped characters by count, descending, capped at missCap.
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
	if len(out) > missCap {
		out = out[:missCap]
	}
	return out
}

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
