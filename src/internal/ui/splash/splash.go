package splash

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// logoLines is the ORBIT wordmark as block-letter art, 6 rows of 44 columns
// (five 8-column letters separated by single spaces).
var logoLines = []string{
	" ██████╗ ██████╗ ██████╗ ██╗     ██████╗ ",
	"██╔═══██╗██╔══██╗██╔══██╗██║     ╚══██╔╝",
	"██║   ██║██████╔╝██████╔╝██║       ██║  ",
	"██║   ██║██╔══██╗██╔══██╗██║       ██║  ",
	"╚██████╔╝██║  ██║██████╔╝██║       ██║  ",
	" ╚═════╝ ╚═╝  ╚═╝╚═════╝ ╚═╝       ╚═╝  ",
}

// logo is the wordmark normalized to uniform row widths so the block art
// stays aligned after centering.
var logo = normalizeLogo()

func normalizeLogo() []string {
	width := 0
	for _, line := range logoLines {
		if n := utf8.RuneCountInString(line); n > width {
			width = n
		}
	}
	out := make([]string, len(logoLines))
	for i, line := range logoLines {
		out[i] = line
		if n := utf8.RuneCountInString(line); n < width {
			out[i] += strings.Repeat(" ", width-n)
		}
	}
	return out
}

// goldRamp is a 256-color ramp from dark bronze to hot gold.
var goldRamp = [...]int{52, 94, 100, 136, 172, 178, 208, 214, 220, 226, 228, 230}

const (
	whiteHot    = 231
	dimGold     = 94
	borderGold  = 136
	progressOn  = 220
	progressOff = 240
)

// Render draws the full-screen shimmering ORBIT intro. progress (0..1) drives
// the loader bar; phase (in sweep units) drives the shimmer band traveling
// across the logo. The result is exactly width x height visible cells.
func Render(width, height int, progress, phase float64, version, bgHex, fgHex string) string {
	if width < 24 {
		width = 24
	}
	if height < 12 {
		height = 12
	}
	bg := hexANSI256(bgHex)
	fg := hexANSI256(fgHex)
	inner := width - 2

	rows := make([]string, 0, height)
	rows = append(rows, borderLine("╭", "╮", inner, bg))

	// Content rows: tagline, logo, subtitle, loader, version.
	blank := blankRow(inner, bg)
	content := make([]string, 0, 12)
	content = append(content, paintLine(center("TERMINAL FILE MANAGER", inner), inner, bg, dim(dimGold)))
	content = append(content, blank)
	for _, line := range logo {
		content = append(content, paintLine(center(line, inner), inner, bg, func(x int, r rune) string {
			return fgColor(charIntensity(x, 0, inner, phase))
		}))
	}
	content = append(content, blank)
	content = append(content, paintLine(center("A fast, beautiful, keyboard-first file manager for Mac developers", inner), inner, bg, dim(fg)))
	content = append(content, blank)
	content = append(content, renderLoader(inner, progress, bg))
	content = append(content, paintLine(center("INITIALIZING WORKSPACE", inner), inner, bg, dim(fg)))
	content = append(content, blank)
	content = append(content, paintLine(center(version, inner), inner, bg, dim(dimGold)))

	// Center the content block vertically inside the border.
	budget := height - 2
	if len(content) > budget {
		content = content[:budget]
	}
	padTop := (budget - len(content)) / 2
	for range padTop {
		rows = append(rows, bgLine(inner, bg))
	}
	rows = append(rows, content...)
	for len(rows) < height-1 {
		rows = append(rows, bgLine(inner, bg))
	}
	rows = append(rows, borderLine("╰", "╯", inner, bg))

	// Clip to the requested height defensively.
	if len(rows) > height {
		rows = rows[:height]
	}
	return strings.Join(rows, "\n")
}

func borderLine(left, right string, inner, bg int) string {
	return fmt.Sprintf("\x1b[48;5;%dm\x1b[38;5;%dm%s%s%s\x1b[0m", bg, borderGold, left, strings.Repeat("─", inner), right)
}

func bgLine(inner, bg int) string {
	return fmt.Sprintf("\x1b[48;5;%dm\x1b[38;5;%dm│%s│\x1b[0m", bg, borderGold, strings.Repeat(" ", inner))
}

func blankRow(inner, bg int) string {
	return paintLine("", inner, bg, dim(236))
}

// dim returns a paint function that renders every character in one 256 color.
func dim(color int) func(int, rune) string {
	return func(int, rune) string { return fmt.Sprintf("\x1b[38;5;%dm", color) }
}

// fgColor maps a shimmer intensity (0..1) onto the gold ramp.
func fgColor(intensity float64) string {
	if intensity > 0.92 {
		return fmt.Sprintf("\x1b[38;5;%dm", whiteHot)
	}
	index := int(math.Round(math.Min(1, math.Max(0, intensity)) * float64(len(goldRamp)-1)))
	return fmt.Sprintf("\x1b[38;5;%dm", goldRamp[index])
}

// charIntensity is the gold brightness of the cell at column x, row row for a
// sweep phase. It combines a static center glow with two traveling highlight
// bands for the shimmer effect.
func charIntensity(x, row, width int, phase float64) float64 {
	center := float64(width-1) / 2
	base := 0.34 + 0.36*math.Exp(-math.Pow(float64(x)-center, 2)/(float64(width)*0.32))
	sweep := (phase - 0.1) * float64(width) * 1.6
	bandA := math.Exp(-math.Pow(float64(x)-sweep, 2) / (float64(width) * 0.5))
	bandB := math.Exp(-math.Pow(float64(x)+float64(row)*0.7-sweep, 2)/(float64(width)*0.9)) * 0.8
	intensity := base + 0.72*math.Max(bandA, bandB)
	if intensity > 1 {
		return 1
	}
	return intensity
}

// renderLoader draws the progress bar row: brackets, gold fill blocks, dim
// empty blocks, and a percentage readout.
func renderLoader(inner int, progress float64, bg int) string {
	barWidth := min(44, max(8, inner-30))
	barWidth = min(barWidth, max(1, inner-4))
	filled := int(math.Round(progress * float64(barWidth)))
	bar := "[" + strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled) + "]"
	percent := fmt.Sprintf(" %3.0f%% ", progress*100)
	plain := bar + percent
	runes := []rune(center(plain, inner))
	if len(runes) > inner {
		runes = runes[:inner]
	}
	var b strings.Builder
	b.WriteString("\x1b[48;5;" + itoa(bg) + "m\x1b[38;5;236m│")
	for _, r := range runes {
		switch r {
		case '█':
			b.WriteString("\x1b[38;5;" + itoa(progressOn) + "m█\x1b[0m\x1b[48;5;" + itoa(bg) + "m")
		case '░':
			b.WriteString("\x1b[38;5;" + itoa(progressOff) + "m░\x1b[0m\x1b[48;5;" + itoa(bg) + "m")
		default:
			b.WriteString("\x1b[38;5;94m" + string(r) + "\x1b[0m\x1b[48;5;" + itoa(bg) + "m")
		}
	}
	for i := len(runes); i < inner; i++ {
		b.WriteString(" ")
	}
	b.WriteString("│\x1b[0m")
	return b.String()
}

// paintLine paints one content row: background fill plus per-character paint.
func paintLine(plain string, inner, bg int, paint func(int, rune) string) string {
	runes := []rune(plain)
	if len(runes) > inner {
		runes = runes[:inner]
	}
	var b strings.Builder
	b.WriteString("\x1b[48;5;" + itoa(bg) + "m\x1b[38;5;236m│")
	for i, r := range runes {
		if r == ' ' {
			b.WriteString(" ")
			continue
		}
		b.WriteString(paint(i, r))
		b.WriteString(string(r))
		b.WriteString("\x1b[0m\x1b[48;5;" + itoa(bg) + "m")
	}
	for i := len(runes); i < inner; i++ {
		b.WriteString(" ")
	}
	b.WriteString("│\x1b[0m")
	return b.String()
}

// center centers plain text within the given inner width.
func center(text string, inner int) string {
	runes := []rune(text)
	if len(runes) >= inner {
		return text
	}
	left := (inner - len(runes)) / 2
	return strings.Repeat(" ", left) + text + strings.Repeat(" ", inner-left-len(runes))
}

func hexANSI256(hex string) int {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 235
	}
	r, errR := strconv.ParseInt(hex[0:2], 16, 32)
	g, errG := strconv.ParseInt(hex[2:4], 16, 32)
	b, errB := strconv.ParseInt(hex[4:6], 16, 32)
	if errR != nil || errG != nil || errB != nil {
		return 235
	}
	ri := int(math.Round(float64(r) / 255 * 5))
	gi := int(math.Round(float64(g) / 255 * 5))
	bi := int(math.Round(float64(b) / 255 * 5))
	return 16 + 36*ri + 6*gi + bi
}

func itoa(n int) string { return strconv.Itoa(n) }
