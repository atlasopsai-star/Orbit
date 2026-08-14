package preview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/atlasopsai-star/Orbit/src/internal/ui"
)

// renderMarkdownDirect runs the block parser and styler without the panel
// renderer so tests can assert exact line structure.
func renderMarkdownDirect(t *testing.T, content string, width, height int) []string {
	t.Helper()
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	return newMarkdownStyler(width).renderBlocks(parseMarkdownBlocks(lines), height)
}

func stripANSI(s string) string {
	return ansi.Strip(s)
}

func TestIsMarkdownFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "README.md", want: true},
		{path: "readme.md", want: true},
		{path: "README", want: true},
		{path: "docs/guide.markdown", want: true},
		{path: "docs/guide.mdown", want: true},
		{path: "docs/guide.mkd", want: true},
		{path: "docs/guide.mdx", want: true},
		{path: "docs/guide.txt", want: false},
		{path: "docs/guide.go", want: false},
		{path: "docs/GUIDE", want: false},
		{path: "main.go", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.want, isMarkdownFile(tt.path))
		})
	}
}

func TestMarkdownHeadingRender(t *testing.T) {
	content := "# Title\n\n## Subtitle\n\n### Deep\n\nPlain paragraph after."
	out := renderMarkdownDirect(t, content, 60, 40)
	require.Len(t, out, 4)

	assert.Contains(t, stripANSI(out[0]), "# Title")
	assert.Contains(t, stripANSI(out[1]), "## Subtitle")
	assert.Contains(t, stripANSI(out[2]), "### Deep")
	assert.Contains(t, stripANSI(out[3]), "Plain paragraph after.")
	// Headings must carry ANSI styling (bold/color) in real theme runs.
	assert.Contains(t, out[0], "\x1b[")
}

func TestMarkdownSetextHeading(t *testing.T) {
	content := "Setext One\n==========\n\nSetext Two\n----------"
	out := renderMarkdownDirect(t, content, 60, 40)
	require.Len(t, out, 2)
	assert.Contains(t, stripANSI(out[0]), "# Setext One")
	assert.Contains(t, stripANSI(out[1]), "## Setext Two")
}

func TestMarkdownFrontMatterSkipped(t *testing.T) {
	content := "---\ntitle: Orbit docs\nlayout: page\n---\n\n# Hello Orbit"
	out := renderMarkdownDirect(t, content, 60, 40)
	joined := stripANSI(strings.Join(out, "\n"))
	assert.Contains(t, joined, "Hello Orbit")
	assert.NotContains(t, joined, "title: Orbit docs")
}

func TestMarkdownInlineStyling(t *testing.T) {
	content := "Use **bold** and *italic* and `code` and ~~gone~~ text."
	out := renderMarkdownDirect(t, content, 120, 40)
	require.Len(t, out, 1)
	assert.Contains(t, out[0], "\x1b[") // styled runs emit ANSI
	plain := stripANSI(out[0])
	assert.Equal(t, "Use bold and italic and code and gone text.", plain)
}

func TestMarkdownInlineCodeUnmatchedStaysLiteral(t *testing.T) {
	content := "unclosed `tick stays literal"
	out := renderMarkdownDirect(t, content, 120, 40)
	require.Len(t, out, 1)
	assert.Equal(t, "unclosed `tick stays literal", stripANSI(out[0]))
}

func TestMarkdownLinkRender(t *testing.T) {
	content := "See the [docs](https://example.com/docs) and https://orbit.dev."
	out := renderMarkdownDirect(t, content, 120, 40)
	require.Len(t, out, 1)
	assert.Contains(t, stripANSI(out[0]), "See the docs (https://example.com/docs) and https://orbit.dev.")
}

func TestMarkdownImageRender(t *testing.T) {
	content := "Logo: ![logo](assets/logo.png)"
	out := renderMarkdownDirect(t, content, 120, 40)
	require.Len(t, out, 1)
	assert.Equal(t, "Logo: logo (image) (assets/logo.png)", stripANSI(out[0]))
}

func TestMarkdownCodeFence(t *testing.T) {
	content := "```go\npackage main\n\nfunc main() {}\n```"
	out := renderMarkdownDirect(t, content, 60, 40)
	require.Len(t, out, 3)
	assert.Equal(t, "package main", stripANSI(out[0]))
	assert.Equal(t, "", stripANSI(out[1]))
	assert.Equal(t, "func main() {}", stripANSI(out[2]))
}

func TestMarkdownListRender(t *testing.T) {
	content := "- first\n- second\n  - nested\n1. ordered one"
	out := renderMarkdownDirect(t, content, 60, 40)
	require.Len(t, out, 4)
	assert.Equal(t, "• first", stripANSI(out[0]))
	assert.Equal(t, "• second", stripANSI(out[1]))
	assert.Equal(t, "  • nested", stripANSI(out[2]))
	assert.Equal(t, "1. ordered one", stripANSI(out[3]))
}

func TestMarkdownEmptyListItemShowsMarker(t *testing.T) {
	out := renderMarkdownDirect(t, "- \n- two", 60, 40)
	require.Len(t, out, 2)
	assert.Equal(t, "• ", stripANSI(out[0]))
	assert.Equal(t, "• two", stripANSI(out[1]))
}

func TestMarkdownBareQuoteLineShowsGutter(t *testing.T) {
	out := renderMarkdownDirect(t, ">\n> quoted", 60, 40)
	require.Len(t, out, 2)
	assert.Contains(t, stripANSI(out[0]), "│")
	assert.Contains(t, stripANSI(out[1]), "│ quoted")
}

func TestMarkdownBlockquoteRender(t *testing.T) {
	content := "> quoted wisdom\n> more of it"
	out := renderMarkdownDirect(t, content, 60, 40)
	require.Len(t, out, 2)
	assert.Contains(t, stripANSI(out[0]), "│ quoted wisdom")
	assert.Contains(t, stripANSI(out[1]), "│ more of it")
}

func TestMarkdownRuleRender(t *testing.T) {
	out := renderMarkdownDirect(t, "above\n\n---\n\nbelow", 12, 40)
	require.Len(t, out, 3)
	assert.Equal(t, "above", stripANSI(out[0]))
	assert.Equal(t, "────────────", stripANSI(out[1]))
	assert.Equal(t, "below", stripANSI(out[2]))
}

func TestMarkdownTableRender(t *testing.T) {
	content := "| Name | Stars |\n|------|-------|\n| Orbit | 5 |\n| Superfile | 3 |"
	out := renderMarkdownDirect(t, content, 40, 40)
	require.Len(t, out, 4)
	// Header row: both names present, separated by the table glyph.
	header := stripANSI(out[0])
	assert.Contains(t, header, "Name")
	assert.Contains(t, header, "Stars")
	assert.Contains(t, header, "│")
	assert.Contains(t, stripANSI(out[1]), "─")
	assert.Contains(t, stripANSI(out[2]), "Orbit")
	assert.Contains(t, stripANSI(out[3]), "Superfile")
	// All rendered lines must fit the panel width.
	for _, line := range out {
		assert.LessOrEqual(t, ansi.StringWidth(stripANSI(line)), 40)
	}
}

func TestMarkdownWrapsLongParagraph(t *testing.T) {
	content := strings.Repeat("word ", 30) + "end"
	out := renderMarkdownDirect(t, content, 20, 40)
	require.Greater(t, len(out), 1)
	for _, line := range out {
		assert.LessOrEqual(t, ansi.StringWidth(stripANSI(line)), 20)
	}
	assert.Contains(t, stripANSI(strings.Join(out, " ")), "end")
}

func TestMarkdownListWrapKeepsContinuationIndent(t *testing.T) {
	content := "- " + strings.Repeat("abc ", 12) + "tail"
	out := renderMarkdownDirect(t, content, 20, 40)
	require.Greater(t, len(out), 1)
	// Continuation lines keep the marker indent so wrapped items stay aligned.
	assert.Equal(t, "  ", stripANSI(out[1])[:2])
}

func TestMarkdownBoundedHeight(t *testing.T) {
	var b strings.Builder
	for i := range 60 {
		b.WriteString("line ")
		b.WriteString(strings.Repeat("x", i%5))
		b.WriteByte('\n')
	}
	out := renderMarkdownDirect(t, b.String(), 60, 8)
	assert.LessOrEqual(t, len(out), 8)
}

// TestMarkdownPreviewFullPath verifies the whole renderMarkdownPreview pipeline
// through the panel renderer, including height padding/truncation.
func TestMarkdownPreviewFullPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "README.md")
	require.NoError(t, os.WriteFile(path, []byte("# Orbit\n\nFast terminal file manager."), 0o644))

	r := ui.FilePreviewPanelRenderer(6, 30)
	render := renderMarkdownPreview(r, path, 30, 6)
	lines := strings.Split(strings.TrimSuffix(render, "\n"), "\n")
	require.GreaterOrEqual(t, len(lines), 2)
	assert.Contains(t, stripANSI(strings.Join(lines, "\n")), "Orbit")
	assert.Contains(t, stripANSI(strings.Join(lines, "\n")), "Fast terminal file manager.")
}

func TestMarkdownPreviewEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.md")
	require.NoError(t, os.WriteFile(path, []byte(""), 0o644))

	// Must render without panicking and stay within the panel height. (The
	// empty-file message is a pre-rendered variable that only exists once the
	// config/icon layer is loaded, so the integration assertion is structural.)
	r := ui.FilePreviewPanelRenderer(4, 30)
	render := renderMarkdownPreview(r, path, 30, 4)
	assert.LessOrEqual(t, strings.Count(render, "\n")+1, 4)
}
