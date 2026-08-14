package preview

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"encoding/csv"
	"errors"
	"fmt"
	"image"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/yorukot/ansichroma"

	"github.com/atlasopsai-star/Orbit/src/pkg/utils"

	"github.com/atlasopsai-star/Orbit/src/internal/common"
	"github.com/atlasopsai-star/Orbit/src/internal/ui"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/rendering"
)

func renderDirectoryPreview(r *rendering.Renderer, itemPath string, previewHeight int) string {
	files, err := os.ReadDir(itemPath)
	if err != nil {
		slog.Error("Error render directory preview", "error", err)
		r.AddLines(common.FilePreviewDirectoryUnreadableText)
		return r.Render()
	}

	info, _ := os.Stat(itemPath)
	if info != nil {
		r.AddLines(fmt.Sprintf("Directory  %d items  Modified %s", len(files), info.ModTime().Format("2006-01-02 15:04")))
	}
	if len(files) == 0 {
		r.AddLines(common.FilePreviewEmptyText)
		return r.Render()
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir() && !files[j].IsDir() {
			return true
		}
		if !files[i].IsDir() && files[j].IsDir() {
			return false
		}
		return files[i].Name() < files[j].Name()
	})

	for i := 0; i < previewHeight && i < len(files); i++ {
		file := files[i]
		isLink := false
		if info, err := file.Info(); err == nil {
			isLink = info.Mode()&os.ModeSymlink != 0
		}
		style := common.GetElementIcon(file.Name(), file.IsDir(), isLink, common.Config.Nerdfont)
		res := lipgloss.NewStyle().Foreground(lipgloss.Color(style.Color)).Background(common.FilePanelBGColor).
			Render(style.Icon+" ") + common.FilePanelStyle.Render(file.Name())
		r.AddLines(res)
	}
	return r.Render()
}

const (
	maxArchivePreviewEntries = 10_000
	maxArchivePreviewSize    = 64 * 1024 * 1024
)

func renderArchivePreview(r *rendering.Renderer, itemPath string, previewHeight int) (string, bool) {
	info, err := os.Stat(itemPath)
	if err != nil {
		return r.AddLines(fmt.Sprintf("Archive preview unavailable: %v", err)).Render(), true
	}
	if info.Size() > maxArchivePreviewSize {
		return r.AddLines(fmt.Sprintf("Archive preview capped — %s archive", common.FormatFileSize(info.Size()))).Render(), true
	}

	name := strings.ToLower(filepath.Base(itemPath))
	switch {
	case strings.HasSuffix(name, ".zip"):
		return renderZipPreview(r, itemPath, previewHeight), true
	case strings.HasSuffix(name, ".tar"), strings.HasSuffix(name, ".tar.gz"), strings.HasSuffix(name, ".tgz"):
		return renderTarPreview(r, itemPath, previewHeight), true
	default:
		return "", false
	}
}

// zipEntryCount reports the number of entries in a ZIP archive by reading only
// the end-of-central-directory records from the file tail. It never parses the
// central directory, so archives with a huge entry list (or a crafted central
// directory) cannot make the preview allocate one header object per entry.
func zipEntryCount(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	size := info.Size()
	if size < 22 {
		return 0, zip.ErrFormat
	}
	tailSize := int64(22 + 65535)
	if tailSize > size {
		tailSize = size
	}
	tail := make([]byte, tailSize)
	if _, err := file.ReadAt(tail, size-tailSize); err != nil {
		return 0, err
	}
	// Scan backwards for the EOCD signature; the record is valid only when its
	// comment length exactly fills the remaining tail bytes.
	for index := len(tail) - 22; index >= 0; index-- {
		if binary.LittleEndian.Uint32(tail[index:]) != 0x06054b50 {
			continue
		}
		if int(binary.LittleEndian.Uint16(tail[index+20:])) != len(tail)-index-22 {
			continue
		}
		count := int(binary.LittleEndian.Uint16(tail[index+10:]))
		if count != 0xffff {
			return count, nil
		}
		// zip64: a locator record sits directly before the EOCD and points at
		// the 56-byte zip64 EOCD record holding the real entry count.
		if index >= 20 && binary.LittleEndian.Uint32(tail[index-20:]) == 0x07064b50 {
			offset := int64(binary.LittleEndian.Uint64(tail[index-12:]))
			if offset < 0 || offset+56 > size {
				return 0, zip.ErrFormat
			}
			record := make([]byte, 56)
			if _, err := file.ReadAt(record, offset); err != nil {
				return 0, err
			}
			if binary.LittleEndian.Uint32(record) != 0x06064b50 {
				return 0, zip.ErrFormat
			}
			total := binary.LittleEndian.Uint64(record[32:])
			if total > uint64(^uint(0)>>1) {
				return 0, zip.ErrFormat
			}
			return int(total), nil
		}
		return 0, zip.ErrFormat
	}
	return 0, zip.ErrFormat
}

func renderZipPreview(r *rendering.Renderer, itemPath string, previewHeight int) string {
	info, _ := os.Stat(itemPath)
	size := "unknown size"
	if info != nil {
		size = common.FormatFileSize(info.Size())
	}

	// Bounded entry listing: when the archive declares more entries than we are
	// willing to materialize as header objects, show the real count from the
	// EOCD and skip opening the central directory entirely.
	entryCount, countErr := zipEntryCount(itemPath)
	if countErr == nil && entryCount > maxArchivePreviewEntries {
		r.AddLines(fmt.Sprintf("ZIP  %s  %d entries", size, entryCount))
		r.AddLines(fmt.Sprintf("listing capped at %d entries", maxArchivePreviewEntries))
		return r.Render()
	}

	archive, err := zip.OpenReader(itemPath)
	if err != nil {
		return r.AddLines(fmt.Sprintf("Archive preview unavailable: %v", err)).Render()
	}
	defer archive.Close()

	entrySuffix := ""
	if entryCount > maxArchivePreviewEntries {
		entryCount = maxArchivePreviewEntries
		entrySuffix = "+"
	}
	r.AddLines(fmt.Sprintf("ZIP  %s  %d%s entries", size, entryCount, entrySuffix))

	limit := previewHeight - 2
	if limit < 1 {
		limit = 1
	}
	for index, file := range archive.File {
		if index >= limit {
			break
		}
		r.AddLines(file.Name)
	}
	if len(archive.File) > limit {
		remaining := len(archive.File) - limit
		if remaining > maxArchivePreviewEntries {
			remaining = maxArchivePreviewEntries
		}
		r.AddLines(fmt.Sprintf("… %d+ more entries", remaining))
	}
	return r.Render()
}

func renderTarPreview(r *rendering.Renderer, itemPath string, previewHeight int) string {
	file, err := os.Open(itemPath)
	if err != nil {
		return r.AddLines(fmt.Sprintf("Archive preview unavailable: %v", err)).Render()
	}
	defer file.Close()

	var input io.Reader = file
	if strings.HasSuffix(strings.ToLower(itemPath), ".gz") || strings.HasSuffix(strings.ToLower(itemPath), ".tgz") {
		reader, gzipErr := gzip.NewReader(file)
		if gzipErr != nil {
			return r.AddLines(fmt.Sprintf("Archive preview unavailable: %v", gzipErr)).Render()
		}
		defer reader.Close()
		input = reader
	}

	limit := previewHeight - 2
	if limit < 1 {
		limit = 1
	}
	entries := 0
	names := make([]string, 0, limit)
	truncated := false
	reader := tar.NewReader(input)
	for {
		header, nextErr := reader.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return r.AddLines(fmt.Sprintf("Archive preview unavailable: %v", nextErr)).Render()
		}
		entries++
		if len(names) < limit {
			names = append(names, header.Name)
		}
		if entries >= maxArchivePreviewEntries {
			truncated = true
			break
		}
	}

	info, _ := os.Stat(itemPath)
	size := "unknown size"
	if info != nil {
		size = common.FormatFileSize(info.Size())
	}
	entryCount := strconv.Itoa(entries)
	if truncated {
		entryCount += "+"
	}
	r.AddLines(fmt.Sprintf("TAR  %s  %s entries", size, entryCount))
	for _, name := range names {
		r.AddLines(name)
	}
	if truncated || entries > limit {
		remaining := entries - limit
		if remaining < 0 {
			remaining = 0
		}
		r.AddLines(fmt.Sprintf("… %d+ more entries", remaining))
	}
	return r.Render()
}

// renderImagePreview returns (render, rawTransmit). rawTransmit is non-empty
// only for Kitty protocol and must be sent via tea.Raw().
func (m *Model) renderImagePreview(r *rendering.Renderer, itemPath string, previewWidth,
	previewHeight int, sideAreaWidth int, kittyClear string,
) (string, string) {
	if !m.open {
		return r.AddLines(common.FilePreviewPanelClosedText).Render(), kittyClear
	}

	if !common.Config.ShowImagePreview {
		return r.AddLines(common.FilePreviewImagePreviewDisabledText).Render(), kittyClear
	}

	imageRender, rawTransmit, err := m.imagePreviewer.ImagePreview(itemPath, previewWidth, previewHeight,
		common.Theme.FilePanelBG, sideAreaWidth)
	if errors.Is(err, image.ErrFormat) {
		return r.AddLines(common.FilePreviewUnsupportedImageFormatsText).Render(), kittyClear
	}

	if err != nil {
		slog.Error("Error converting image to ANSI", "error", err)
		return r.AddLines(common.FilePreviewImageConversionErrorText).Render(), kittyClear
	}

	// For Kitty placeholders or ANSI output, use vertical alignment
	return r.AddStyleModifier(func(s lipgloss.Style) lipgloss.Style {
		return s.AlignHorizontal(lipgloss.Center).AlignVertical(lipgloss.Center)
	}).AddLines(imageRender).Render(), rawTransmit
}

func renderPreviewError(err error) string {
	return common.FilePreviewError + fmt.Sprintf("\n%s", err)
}

func (m *Model) renderTextPreview(r *rendering.Renderer, itemPath string,
	previewWidth, previewHeight int,
) string {
	format := lexers.Match(filepath.Base(itemPath))
	if format == nil {
		isText, err := common.IsTextFile(itemPath)
		if err != nil {
			slog.Error("Error while checking text file", "error", err)
			return r.AddLines(renderPreviewError(err)).Render()
		} else if !isText {
			return r.AddLines(common.FilePreviewUnsupportedFormatText).Render()
		}
	}

	fileContent, err := utils.ReadFileContent(itemPath, previewWidth, previewHeight)
	if err != nil {
		slog.Error("Error open file", "error", err)
		return r.AddLines(renderPreviewError(err)).Render()
	}

	if fileContent == "" {
		return r.AddLines(common.FilePreviewEmptyText).Render()
	}

	if fileInfo, statErr := os.Stat(itemPath); statErr == nil && fileInfo.Size() > maxPreviewFileSize {
		r.AddLines(fmt.Sprintf("Preview truncated — %s file", common.FormatFileSize(fileInfo.Size())))
	}

	if isDelimitedPreview(itemPath) {
		return renderDelimitedPreview(r, itemPath, previewHeight)
	}

	if format != nil {
		background := ""
		if !common.Config.TransparentBackground {
			background = common.Theme.FilePanelBG
		}
		if common.Config.CodePreviewer == "bat" {
			if m.batCmd == "" {
				return r.AddLines(common.FilePreviewBatNotInstalledText).Render()
			}
			fileContent, err = getBatSyntaxHighlightedContent(itemPath, previewHeight, background, m.batCmd)
		} else {
			fileContent, err = ansichroma.HightlightString(fileContent, format.Config().Name,
				common.Theme.CodeSyntaxHighlightTheme, background)
		}
		if err != nil {
			slog.Error("Error render code highlight", "error", err)
			return r.AddLines(renderPreviewError(err)).Render()
		}
	}

	r.AddLines(fileContent)
	return r.Render()
}

const maxPreviewFileSize int64 = 8 * 1024 * 1024

func isDelimitedPreview(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".csv" || ext == ".tsv"
}

func renderDelimitedPreview(r *rendering.Renderer, itemPath string, previewHeight int) string {
	file, err := os.Open(itemPath)
	if err != nil {
		return r.AddLines(renderPreviewError(err)).Render()
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(io.LimitReader(file, 256*1024)))
	reader.Comma = ','
	if strings.EqualFold(filepath.Ext(itemPath), ".tsv") {
		reader.Comma = '\t'
	}
	reader.FieldsPerRecord = -1
	rows := max(1, previewHeight-2)
	for index := 0; index < rows; index++ {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			r.AddLines(fmt.Sprintf("CSV preview stopped at row %d: %v", index+1, readErr))
			break
		}
		values := make([]string, 0, min(len(record), 8))
		for _, value := range record[:min(len(record), 8)] {
			runes := []rune(value)
			if len(runes) > 32 {
				value = string(runes[:29]) + "..."
			}
			values = append(values, value)
		}
		r.AddLines(fmt.Sprintf("%3s  %s", strconv.Itoa(index+1), strings.Join(values, " │ ")))
	}
	return r.Render()
}

// Only use this when height and width are synced with filemodel's expectations
func (m *Model) RenderText(text string) string {
	return m.RenderTextWithDimension(text, m.contentHeight, m.contentWidth)
}

func (m *Model) RenderTextWithDimension(text string, height int, width int) string {
	// For zero size, don't need to render anything. Its kinda hack, but
	// its to prevent error logs
	if width == 0 && height == 0 {
		return ""
	}
	return ui.FilePreviewPanelRenderer(height, width).
		AddLines(text).
		Render()
}

// RenderWithPath returns (render, rawTransmit). rawTransmit is non-empty
// for Kitty images (transmit data) or when clearing Kitty images (delete-all).
// It must be sent via tea.Raw().
func (m *Model) RenderWithPath(
	itemPath string,
	previewWidth int,
	previewHeight int,
	fullModelWidth int,
) (string, string) {
	r := ui.FilePreviewPanelRenderer(previewHeight, previewWidth)
	// Raw command to clear any previous Kitty images when showing non-image content
	kittyClear := m.imagePreviewer.GetKittyClearRaw()

	// Adjust dimensions if border is enabled
	contentWidth := previewWidth
	contentHeight := previewHeight
	if common.Config.EnableFilePreviewBorder {
		contentWidth = previewWidth - common.BorderPadding
		contentHeight = previewHeight - common.BorderPadding
	}

	fileInfo, infoErr := os.Stat(itemPath)
	if infoErr != nil {
		slog.Error("Error get file info", "error", infoErr)
		return r.AddLines(common.FilePreviewNoFileInfoText).Render(), kittyClear
	}
	slog.Debug("Attempting to render preview", "itemPath", itemPath,
		"mode", fileInfo.Mode().String(), "isRegular", fileInfo.Mode().IsRegular())

	// For non regular files which are not directories Dont try to read them
	// See Issue #876
	if !fileInfo.Mode().IsRegular() && (fileInfo.Mode()&fs.ModeDir) == 0 {
		return r.AddLines(common.FilePreviewUnsupportedFileMode).Render(), kittyClear
	}

	ext := filepath.Ext(itemPath)
	if slices.Contains(common.UnsupportedPreviewFormats, ext) {
		return r.AddLines(common.FilePreviewUnsupportedFormatText).Render(), kittyClear
	}

	if fileInfo.IsDir() {
		return renderDirectoryPreview(r, itemPath, contentHeight), kittyClear
	}

	if archive, ok := renderArchivePreview(r, itemPath, contentHeight); ok {
		return archive, kittyClear
	}

	if m.thumbnailGenerator != nil && m.thumbnailGenerator.SupportsExt(ext) {
		thumbnailPath, err := m.thumbnailGenerator.GetThumbnailOrGenerate(itemPath)
		if err != nil {
			slog.Error("Error generating thumbnail", "error", err)
			return r.AddLines(common.FilePreviewThumbnailGenerationErrorText).Render(), kittyClear
		}
		return m.renderImagePreview(
			r, thumbnailPath, contentWidth, contentHeight,
			fullModelWidth-previewWidth, kittyClear)
	}

	if isImageFile(itemPath) {
		return m.renderImagePreview(
			r, itemPath, contentWidth, contentHeight,
			fullModelWidth-previewWidth, kittyClear)
	}

	return m.renderTextPreview(r, itemPath, contentWidth, contentHeight), kittyClear
}
