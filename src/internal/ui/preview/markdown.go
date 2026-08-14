package preview

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/x/ansi"
	"github.com/yorukot/ansichroma"

	"github.com/atlasopsai-star/Orbit/src/internal/common"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/rendering"
)

// isMarkdownFile reports whether the path should use the rendered Markdown
// preview: Markdown extensions, or a file literally named README (which is
// conventionally Markdown even when extensionless).
func isMarkdownFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".mdown", ".mkd", ".mdx":
		return true
	}
	return strings.EqualFold(filepath.Base(path), "readme")
}

const (
	maxMarkdownPreviewBytes = 512 * 1024
	// maxMarkdownPreviewLines caps how many raw lines are parsed; the rendered
	// output is height-bounded anyway, so anything beyond this would only burn
	// allocations on a pathological input (e.g. a huge file of one-char lines).
	maxMarkdownPreviewLines = 4096
)

// renderMarkdownPreview parses itemPath as Markdown and renders it as styled
// ANSI lines into the panel renderer. The parser is line-based and bounded:
// only a capped amount of input is read and only previewHeight output lines
// are produced, so large docs cannot blow up memory or the layout.
func renderMarkdownPreview(r *rendering.Renderer, itemPath string, previewWidth, previewHeight int) string {
	lines, truncated, err := readMarkdownLines(itemPath)
	if err != nil {
		return r.AddLines(renderPreviewError(err)).Render()
	}
	if len(lines) == 0 {
		return r.AddLines(common.FilePreviewEmptyText).Render()
	}

	styler := newMarkdownStyler(previewWidth)
	out := styler.renderBlocks(parseMarkdownBlocks(lines), previewHeight)
	if truncated {
		out = append(out, "Preview truncated — large Markdown file")
	}
	if len(out) > previewHeight {
		out = out[:previewHeight]
	}
	r.AddLines(out...)
	return r.Render()
}

// readMarkdownLines reads raw Markdown lines (untouched, so block parsing sees
// real structure) up to a byte cap. The second result reports whether the
// read was truncated.
func readMarkdownLines(path string) ([]string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, false, err
	}
	truncated := info.Size() > maxMarkdownPreviewBytes

	scanner := bufio.NewScanner(bufio.NewReader(io.LimitReader(file, maxMarkdownPreviewBytes)))
	scanner.Buffer(make([]byte, 0, 64*1024), 512*1024)
	lines := make([]string, 0, 128)
	for scanner.Scan() {
		lines = append(lines, strings.TrimSuffix(scanner.Text(), "\r"))
		if len(lines) >= maxMarkdownPreviewLines {
			truncated = true
			break
		}
	}
	return lines, truncated, scanner.Err()
}

// --- Block parsing ----------------------------------------------------------

type mdBlockKind int

const (
	mdBlockParagraph mdBlockKind = iota
	mdBlockHeading
	mdBlockList
	mdBlockCode
	mdBlockQuote
	mdBlockRule
	mdBlockTable
)

type mdListItem struct {
	text    string
	depth   int
	ordered bool
	number  int
}

type mdBlock struct {
	kind  mdBlockKind
	level int
	text  []string
	items []mdListItem
	rows  [][]string
	lang  string
} // parseMarkdownBlocks is a small line-based Markdown block parser covering the
// constructs that appear in READMEs and docs: ATX and setext headings, fenced
// and indented code, lists (one nesting level), blockquotes, rules, GFM tables
// and paragraphs. It deliberately trades full CommonMark fidelity for a
// bounded, predictable preview.
func parseMarkdownBlocks(lines []string) []mdBlock {
	blocks := make([]mdBlock, 0, 16)
	index := 0
	n := len(lines)

	// Skip YAML front matter when present.
	if n >= 2 && strings.TrimSpace(lines[0]) == "---" {
		for index = 1; index < n; index++ {
			if strings.TrimSpace(lines[index]) == "---" {
				index++
				break
			}
		}
	}

outer:
	for index < n {
		line := lines[index]
		trimmed := strings.TrimSpace(line)

		switch {
		case trimmed == "":
			index++
		case isFenceOpen(trimmed):
			lang := strings.TrimSpace(strings.TrimLeft(trimmed, "`~"))
			block, next := collectFence(lines, index, trimmed[:3])
			block.lang = lang
			blocks = append(blocks, block)
			index = next
		case isATXHeading(trimmed):
			level, text := parseATXHeading(trimmed)
			blocks = append(blocks, mdBlock{kind: mdBlockHeading, level: level, text: []string{text}})
			index++
		case strings.HasPrefix(trimmed, ">"):
			block, next := collectQuote(lines, index)
			blocks = append(blocks, block)
			index = next
		case isListLine(trimmed):
			block, next := collectList(lines, index)
			blocks = append(blocks, block)
			index = next
		case isTableStart(lines, index):
			block, next := collectTable(lines, index)
			blocks = append(blocks, block)
			index = next
		case isHorizontalRule(trimmed):
			blocks = append(blocks, mdBlock{kind: mdBlockRule})
			index++
		default:
			// Paragraph, possibly a setext heading.
			text := []string{trimmed}
			index++
			for index < n {
				cont := strings.TrimSpace(lines[index])
				if cont == "" {
					break
				}
				if isSetextUnderline(cont, "=") {
					blocks = append(blocks, mdBlock{kind: mdBlockHeading, level: 1, text: text})
					index++
					continue outer
				}
				if isSetextUnderline(cont, "-") {
					blocks = append(blocks, mdBlock{kind: mdBlockHeading, level: 2, text: text})
					index++
					continue outer
				}
				if isATXHeading(cont) || isFenceOpen(cont) || isListLine(cont) ||
					strings.HasPrefix(cont, ">") || isHorizontalRule(cont) || isTableStart(lines, index) {
					break
				}
				text = append(text, cont)
				index++
			}
			blocks = append(blocks, mdBlock{kind: mdBlockParagraph, text: text})
		}
	}
	return blocks
}

func isFenceOpen(trimmed string) bool {
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}

func collectFence(lines []string, start int, fence string) (mdBlock, int) {
	text := make([]string, 0)
	index := start + 1
	for index < len(lines) {
		if strings.HasPrefix(strings.TrimSpace(lines[index]), fence) {
			return mdBlock{kind: mdBlockCode, text: text}, index + 1
		}
		text = append(text, lines[index])
		index++
	}
	return mdBlock{kind: mdBlockCode, text: text}, index
}

func isATXHeading(trimmed string) bool {
	if !strings.HasPrefix(trimmed, "#") {
		return false
	}
	hashes := 0
	for hashes < len(trimmed) && trimmed[hashes] == '#' {
		hashes++
	}
	return hashes <= 6 && (hashes == len(trimmed) || trimmed[hashes] == ' ' || trimmed[hashes] == '\t')
}

func parseATXHeading(trimmed string) (int, string) {
	hashes := 0
	for hashes < len(trimmed) && trimmed[hashes] == '#' {
		hashes++
	}
	return hashes, strings.TrimSpace(strings.TrimLeft(trimmed[hashes:], " \t"))
}

func isSetextUnderline(trimmed, ch string) bool {
	if trimmed == "" {
		return false
	}
	count := 0
	for _, r := range trimmed {
		if r == ' ' || r == '\t' {
			continue
		}
		if string(r) != ch {
			return false
		}
		count++
	}
	return count >= 1
}

func isHorizontalRule(trimmed string) bool {
	if trimmed == "" {
		return false
	}
	marker := trimmed[0]
	if marker != '-' && marker != '*' && marker != '_' {
		return false
	}
	count := 0
	for _, r := range trimmed {
		if r == ' ' || r == '\t' {
			continue
		}
		if byte(r) != marker {
			return false
		}
		count++
	}
	return count >= 3
}

func collectQuote(lines []string, start int) (mdBlock, int) {
	text := make([]string, 0)
	index := start
	for index < len(lines) {
		trimmed := strings.TrimSpace(lines[index])
		if !strings.HasPrefix(trimmed, ">") {
			break
		}
		text = append(text, strings.TrimSpace(strings.TrimPrefix(trimmed, ">")))
		index++
	}
	return mdBlock{kind: mdBlockQuote, text: text}, index
}

func isListLine(trimmed string) bool {
	_, ok := parseListLine(trimmed)
	return ok
}

// parseListLine recognizes "- item", "* item", "+ item" and "1. item" style
// lines with up to one level of nesting by indentation.
func parseListLine(trimmed string) (mdListItem, bool) {
	raw := trimmed
	trimmed = strings.TrimLeft(trimmed, " \t")
	indent := len(raw) - len(trimmed)
	if indent >= 4 || trimmed == "" {
		return mdListItem{}, false
	}
	depth := 0
	if indent >= 2 {
		depth = 1
	}
	switch {
	case trimmed[0] == '-' || trimmed[0] == '+' || trimmed[0] == '*':
		if len(trimmed) == 1 || trimmed[1] == ' ' || trimmed[1] == '\t' {
			return mdListItem{text: strings.TrimSpace(trimmed[1:]), depth: depth}, true
		}
	case trimmed[0] >= '0' && trimmed[0] <= '9':
		i := 0
		for i < len(trimmed) && trimmed[i] >= '0' && trimmed[i] <= '9' {
			i++
		}
		if i > 0 && i < len(trimmed) && (trimmed[i] == '.' || trimmed[i] == ')') {
			number, _ := strconv.Atoi(trimmed[:i])
			return mdListItem{text: strings.TrimSpace(trimmed[i+1:]), depth: depth, ordered: true, number: number}, true
		}
	}
	return mdListItem{}, false
}

func collectList(lines []string, start int) (mdBlock, int) {
	items := make([]mdListItem, 0)
	index := start
	for index < len(lines) {
		item, ok := parseListLine(lines[index])
		if !ok {
			break
		}
		items = append(items, item)
		index++
	}
	return mdBlock{kind: mdBlockList, items: items}, index
}

func isTableStart(lines []string, index int) bool {
	if index+1 >= len(lines) || !strings.Contains(lines[index], "|") {
		return false
	}
	return isTableSeparator(strings.TrimSpace(lines[index+1]))
}

func isTableSeparator(trimmed string) bool {
	if trimmed == "" || !strings.Contains(trimmed, "-") {
		return false
	}
	trimmed = strings.Trim(trimmed, " \t|")
	for _, part := range strings.Split(trimmed, "|") {
		p := strings.TrimSpace(part)
		if p == "" {
			return false
		}
		hasDash := false
		for _, r := range p {
			if r == '-' {
				hasDash = true
			} else if r != ':' && r != ' ' && r != '\t' {
				return false
			}
		}
		if !hasDash {
			return false
		}
	}
	return true
}

func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	cells := strings.Split(line, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

func collectTable(lines []string, start int) (mdBlock, int) {
	rows := [][]string{splitTableRow(lines[start])}
	index := start + 2 // skip the header and the separator row
	for index < len(lines) {
		line := strings.TrimSpace(lines[index])
		if line == "" || !strings.Contains(line, "|") {
			break
		}
		rows = append(rows, splitTableRow(line))
		index++
	}
	return mdBlock{kind: mdBlockTable, rows: rows}, index
}

// --- Rendering --------------------------------------------------------------

type mdRun struct {
	text  string
	style lipgloss.Style
}

// markdownStyler holds the render width and produces styled ANSI lines. All
// colors come from the active Orbit theme so the preview matches the panel.
type markdownStyler struct {
	width int
}

func newMarkdownStyler(width int) *markdownStyler {
	return &markdownStyler{width: max(1, width)}
}

func (md *markdownStyler) renderBlocks(blocks []mdBlock, height int) []string {
	out := make([]string, 0, 64)
	for _, block := range blocks {
		if len(out) >= height {
			break
		}
		remaining := height - len(out)
		switch block.kind {
		case mdBlockHeading:
			out = append(out, md.renderHeading(block)...)
		case mdBlockParagraph:
			out = append(out, md.layout(parseInline(strings.Join(block.text, " ")))...)
		case mdBlockList:
			out = append(out, md.renderList(block, remaining)...)
		case mdBlockCode:
			out = append(out, md.renderCode(block, remaining)...)
		case mdBlockQuote:
			out = append(out, md.renderQuote(block, remaining)...)
		case mdBlockRule:
			out = append(out, md.renderRule())
		case mdBlockTable:
			out = append(out, md.renderTable(block, remaining)...)
		}
	}
	if len(out) > height {
		out = out[:height]
	}
	return out
}

func (md *markdownStyler) renderHeading(b mdBlock) []string {
	marker := strings.Repeat("#", b.level)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.Hint))
	var style lipgloss.Style
	switch b.level {
	case 1:
		style = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(common.Theme.Cursor))
	case 2:
		style = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(common.Theme.Hint))
	default:
		style = lipgloss.NewStyle().Bold(true)
	}
	return md.layout([]mdRun{{text: marker + " ", style: dim}, {text: strings.Join(b.text, " "), style: style}})
}

func (md *markdownStyler) renderList(b mdBlock, remaining int) []string {
	out := make([]string, 0, len(b.items))
	markerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.Cursor))
	for _, item := range b.items {
		if len(out) >= remaining {
			break
		}
		marker := "•"
		if item.ordered {
			marker = strconv.Itoa(item.number) + "."
		}
		prefix := strings.Repeat("  ", item.depth) + marker + " "
		lines := md.layoutIndent(parseInline(item.text), len(prefix))
		if len(lines) == 0 {
			// Empty item (e.g. "- " alone) must still show its marker.
			out = append(out, markerStyle.Render(prefix))
			continue
		}
		for i, line := range lines {
			if i == 0 {
				out = append(out, markerStyle.Render(prefix)+line)
			} else {
				out = append(out, strings.Repeat(" ", len(prefix))+line)
			}
			if len(out) >= remaining {
				break
			}
		}
	}
	return out
}

func (md *markdownStyler) renderCode(b mdBlock, remaining int) []string {
	background := ""
	if !common.Config.TransparentBackground {
		background = common.Theme.FilePanelBG
	}
	if b.lang != "" {
		if lexer := lexers.Get(b.lang); lexer != nil {
			highlighted, err := ansichroma.HightlightString(strings.Join(b.text, "\n"),
				lexer.Config().Name, common.Theme.CodeSyntaxHighlightTheme, background)
			if err == nil {
				out := make([]string, 0, len(b.text))
				for line := range strings.SplitSeq(highlighted, "\n") {
					if len(out) >= remaining {
						break
					}
					out = append(out, line)
				}
				return out
			}
		}
	}
	out := make([]string, 0, len(b.text))
	for _, line := range b.text {
		if len(out) >= remaining {
			break
		}
		out = append(out, line)
	}
	return out
}

func (md *markdownStyler) renderQuote(b mdBlock, remaining int) []string {
	out := make([]string, 0)
	barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.FilePanelBorder))
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.Hint))
	for _, line := range b.text {
		if len(out) >= remaining {
			break
		}
		lines := md.layoutIndent([]mdRun{{text: line, style: textStyle}}, 2)
		if len(lines) == 0 {
			// Bare ">" line: still show the gutter bar.
			lines = []string{""}
		}
		for i, l := range lines {
			if i == 0 {
				out = append(out, barStyle.Render("│ ")+l)
			} else {
				out = append(out, "  "+l)
			}
			if len(out) >= remaining {
				break
			}
		}
	}
	return out
}

func (md *markdownStyler) renderRule() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.FilePanelBorder))
	return style.Render(strings.Repeat("─", md.width))
}

func (md *markdownStyler) renderTable(b mdBlock, remaining int) []string {
	if len(b.rows) == 0 {
		return nil
	}
	cols := len(b.rows[0])
	for _, row := range b.rows {
		cols = max(cols, len(row))
	}
	widths := make([]int, cols)
	for _, row := range b.rows {
		for i, cell := range row {
			widths[i] = max(widths[i], ansi.StringWidth(cell))
		}
	}
	// Shrink the widest columns until the whole table fits the panel width.
	budget := md.width - (cols-1)*3 - 2
	for total := sumWidths(widths); total > budget && budget >= cols; total = sumWidths(widths) {
		widest := 0
		for i := 1; i < cols; i++ {
			if widths[i] > widths[widest] {
				widest = i
			}
		}
		if widths[widest] <= 0 {
			break
		}
		widths[widest]--
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(common.Theme.Cursor))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.Hint))
	pad := func(cell string, width int) string {
		if extra := width - ansi.StringWidth(cell); extra > 0 {
			return cell + strings.Repeat(" ", extra)
		}
		return cell
	}

	out := make([]string, 0, len(b.rows))
	for rowIdx, row := range b.rows {
		if len(out) >= remaining {
			break
		}
		cells := make([]string, cols)
		for i := range cols {
			var cell string
			if i < len(row) {
				cell = pad(row[i], widths[i])
			} else {
				cell = strings.Repeat(" ", widths[i])
			}
			if rowIdx == 0 {
				cell = headerStyle.Render(cell)
			}
			cells[i] = cell
		}
		out = append(out, sepStyle.Render("│")+strings.Join(cells, sepStyle.Render("│"))+sepStyle.Render("│"))
		if rowIdx == 0 && len(out) < remaining {
			seps := make([]string, cols)
			for i := range cols {
				seps[i] = strings.Repeat("─", widths[i])
			}
			out = append(out, sepStyle.Render("├")+strings.Join(seps, sepStyle.Render("┼"))+sepStyle.Render("┤"))
		}
	}
	return out
}

func sumWidths(widths []int) int {
	total := 0
	for _, w := range widths {
		total += w
	}
	return total
}

// --- Inline rendering -------------------------------------------------------

var mdInlineSpecials = []string{"**", "~~", "`", "[", "<", "*", "_"}

// parseInline splits a line into styled runs for bold, italic, inline code,
// strikethrough, links and autolinks. Nesting is intentionally flat.
func parseInline(text string) []mdRun {
	runs := make([]mdRun, 0, 4)
	var plain strings.Builder
	flush := func() {
		if plain.Len() > 0 {
			runs = append(runs, mdRun{text: plain.String(), style: lipgloss.NewStyle()})
			plain.Reset()
		}
	}

	i := 0
	for i < len(text) {
		// Backslash escapes the next Markdown punctuation character.
		if text[i] == '\\' && i+1 < len(text) && strings.ContainsRune("*_`[<~", rune(text[i+1])) {
			plain.WriteByte(text[i+1])
			i += 2
			continue
		}

		found := -1
		for _, spec := range mdInlineSpecials {
			if strings.HasPrefix(text[i:], spec) {
				found = len(spec)
				break
			}
		}
		if found == -1 {
			plain.WriteByte(text[i])
			i++
			continue
		}
		spec := text[i : i+found]

		if spec == "[" {
			end := strings.Index(text[i:], "](")
			if end >= 0 {
				closeParen := strings.Index(text[i+end+2:], ")")
				if closeParen >= 0 {
					label := text[i+1 : i+end]
					url := text[i+end+2 : i+end+2+closeParen]
					isImage := plain.Len() > 0 && strings.HasSuffix(plain.String(), "!")
					if isImage {
						p := plain.String()
						plain.Reset()
						plain.WriteString(p[:len(p)-1])
					}
					flush()
					dim := lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.Hint))
					if isImage {
						runs = append(runs, mdRun{text: label + " (image)", style: dim})
					} else if label != "" {
						runs = append(runs, mdRun{text: label, style: lipgloss.NewStyle()})
					}
					if url != "" {
						runs = append(runs, mdRun{text: " (" + url + ")", style: dim})
					}
					i += end + 2 + closeParen + 1
					continue
				}
			}
			plain.WriteString("[")
			i++
			continue
		}

		if spec == "<" {
			end := strings.IndexByte(text[i:], '>')
			if end > 0 {
				inner := text[i+1 : i+end]
				if strings.HasPrefix(inner, "http://") || strings.HasPrefix(inner, "https://") ||
					strings.HasPrefix(inner, "mailto:") || strings.HasPrefix(inner, "ftp://") {
					flush()
					dim := lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.Hint))
					runs = append(runs, mdRun{text: inner, style: dim})
					i += end + 1
					continue
				}
			}
			plain.WriteString("<")
			i++
			continue
		}

		if matched, consumed, style, ok := matchInline(text[i:], spec); ok {
			flush()
			runs = append(runs, mdRun{text: matched, style: style})
			i += consumed
			continue
		}
		plain.WriteString(spec)
		i += found
	}
	flush()
	return runs
}

func matchInline(s, spec string) (string, int, lipgloss.Style, bool) {
	var style lipgloss.Style
	switch spec {
	case "**":
		style = lipgloss.NewStyle().Bold(true)
	case "*", "_":
		style = lipgloss.NewStyle().Italic(true)
	case "`":
		style = lipgloss.NewStyle().Foreground(lipgloss.Color(common.Theme.Cursor))
	case "~~":
		style = lipgloss.NewStyle().Strikethrough(true)
	default:
		return "", 0, lipgloss.NewStyle(), false
	}
	start := len(spec)
	if len(s) <= start {
		return "", 0, lipgloss.NewStyle(), false
	}
	end := strings.Index(s[start:], spec)
	if end <= 0 {
		return "", 0, lipgloss.NewStyle(), false
	}
	return s[start : start+end], start + end + len(spec), style, true
}

// layout wraps runs into lines no wider than the panel. Wrapping happens at
// word boundaries across styled runs, so bold or inline-code spans survive
// line breaks.
func (md *markdownStyler) layout(runs []mdRun) []string {
	return md.layoutIndent(runs, 0)
}

func (md *markdownStyler) layoutIndent(runs []mdRun, indent int) []string {
	width := md.width - indent
	if width < 1 {
		width = 1
	}
	lines := make([]string, 0)
	var line strings.Builder
	lineWidth := 0
	flush := func() {
		if line.Len() > 0 {
			lines = append(lines, line.String())
			line.Reset()
			lineWidth = 0
		}
	}
	for _, run := range runs {
		for _, word := range strings.Fields(run.text) {
			wordWidth := ansi.StringWidth(word)
			if lineWidth > 0 && lineWidth+1+wordWidth > width {
				flush()
			}
			if lineWidth > 0 {
				line.WriteByte(' ')
				lineWidth++
			}
			line.WriteString(run.style.Render(word))
			lineWidth += wordWidth
		}
	}
	flush()
	return lines
}
