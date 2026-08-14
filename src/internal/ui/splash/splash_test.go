package splash

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestLogoGeometry(t *testing.T) {
	if len(logo) != 6 {
		t.Fatalf("logo should be 6 rows, got %d", len(logo))
	}
	width := ansi.StringWidth(logo[0])
	for i, line := range logo {
		if w := ansi.StringWidth(line); w != width {
			t.Fatalf("logo rows must be uniform width: row %d is %d, want %d", i, w, width)
		}
	}
	if width < 30 {
		t.Fatalf("logo should be wide block art, got %d columns", width)
	}
}

func TestRenderBoundedAtAllSizes(t *testing.T) {
	sizes := [][2]int{{160, 50}, {120, 36}, {100, 30}, {80, 24}, {60, 20}, {40, 12}, {200, 60}}
	for _, size := range sizes {
		width, height := size[0], size[1]
		out := Render(width, height, 0.5, 0.25, "v0.1.0-rc1", "#0B1020", "#D7E3F4")
		lines := strings.Split(out, "\n")
		if len(lines) != height {
			t.Fatalf("%dx%d: expected %d lines, got %d", width, height, height, len(lines))
		}
		for i, line := range lines {
			if w := ansi.StringWidth(line); w != width {
				t.Fatalf("%dx%d line %d: visible width %d, want %d", width, height, i, w, width)
			}
		}
	}
}

func TestRenderContainsGoldArt(t *testing.T) {
	out := Render(120, 36, 1.0, 0.5, "v0.1.0-rc1", "#0B1020", "#D7E3F4")
	for _, probe := range []string{"\x1b[38;5;", "█", "100%", "v0.1.0-rc1", "TERMINAL FILE MANAGER"} {
		if !strings.Contains(out, probe) && !strings.Contains(ansi.Strip(out), probe) {
			t.Fatalf("render missing %q", probe)
		}
	}
}

func TestCharIntensityRange(t *testing.T) {
	for x := -5; x <= 130; x++ {
		for row := 0; row < 6; row++ {
			for _, phase := range []float64{0, 0.3, 0.6, 1.0} {
				value := charIntensity(x, row, 34, phase)
				if value < 0 || value > 1 {
					t.Fatalf("intensity out of range: %f", value)
				}
			}
		}
	}
}

func TestProgressAdvances(t *testing.T) {
	low := Render(100, 30, 0.0, 0.0, "v", "#0B1020", "#D7E3F4")
	full := Render(100, 30, 1.0, 0.0, "v", "#0B1020", "#D7E3F4")
	lowFilled := strings.Count(low, "\x1b[38;5;220m█")
	fullFilled := strings.Count(full, "\x1b[38;5;220m█")
	if fullFilled <= lowFilled {
		t.Fatalf("progress bar should fill: low=%d full=%d", lowFilled, fullFilled)
	}
}
