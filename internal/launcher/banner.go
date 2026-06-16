package launcher

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
)

// bannerFetch returns the raw `--banner` output for the binary at binPath.
// It is a package var so tests can stub it instead of execing real binaries.
var bannerFetch = execBanner

func execBanner(binPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, binPath, "--banner").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// displayName is the friendly name shown in the UI: the install id without the
// .shaw suffix (e.g. "snake.shaw" -> "snake").
func displayName(id string) string { return strings.TrimSuffix(id, ".shaw") }

// lineWidth is the display width of a single line of art (multibyte-safe).
func lineWidth(s string) int { return runewidth.StringWidth(s) }

// fitBanner returns exactly h lines, each exactly w display columns wide, with
// the art clipped to fit and centered horizontally and vertically.
func fitBanner(art string, w, h int) []string {
	raw := strings.Split(strings.TrimRight(art, "\n"), "\n")
	var lines []string
	for _, ln := range raw {
		lines = append(lines, runewidth.Truncate(ln, w, ""))
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	out := make([]string, h)
	for i := range out {
		out[i] = strings.Repeat(" ", w)
	}
	top := (h - len(lines)) / 2
	for i, ln := range lines {
		pad := (w - runewidth.StringWidth(ln)) / 2
		if pad < 0 {
			pad = 0
		}
		out[top+i] = runewidth.FillRight(strings.Repeat(" ", pad)+ln, w)
	}
	return out
}

// bannerBlock renders fitted art as an artW x artH block string (joined lines).
func bannerBlock(art string) string {
	return strings.Join(fitBanner(art, artW, artH), "\n")
}

// defaultBannerBlock renders the styled fallback banner (friendly name on an
// accent block) as an artW x artH block string.
func defaultBannerBlock(name string) string {
	return defaultBannerStyle.Render(displayName(name))
}
