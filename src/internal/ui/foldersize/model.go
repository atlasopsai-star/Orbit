package foldersize

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atlasopsai-star/Orbit/src/pkg/orbitfs"
)

type streamStartedMsg struct {
	Request uint64
	Stream  <-chan orbitfs.SizeProgress
}
type progressMsg struct {
	Request  uint64
	Progress orbitfs.SizeProgress
}
type cacheEntry struct {
	Result orbitfs.SizeResult
	At     time.Time
}

type Model struct {
	path     string
	stream   <-chan orbitfs.SizeProgress
	cancel   context.CancelFunc
	request  uint64
	current  orbitfs.SizeResult
	cachedAt time.Time
	cache    map[string]cacheEntry
	open     bool
	loading  bool
	err      error
	width    int
	height   int
}

func New() Model { return Model{cache: make(map[string]cacheEntry), width: 64, height: 14} }

func (m *Model) Open(path string, refresh bool) tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}
	m.path = path
	m.request++
	request := m.request
	m.open = true
	m.err = nil
	if !refresh {
		if cached, ok := m.cache[path]; ok {
			m.current = cached.Result
			m.cachedAt = cached.At
			m.loading = false
			return nil
		}
	}
	m.current = orbitfs.SizeResult{}
	m.cachedAt = time.Time{}
	m.loading = true
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	stream := orbitfs.DirectorySizeStream(ctx, path)
	return func() tea.Msg { return streamStartedMsg{Request: request, Stream: stream} }
}

func (m *Model) Close() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.open = false
	m.stream = nil
	m.loading = false
}
func (m *Model) IsOpen() bool { return m.open }
func (m *Model) SetDimensions(width, height int) {
	if width > 0 {
		m.width = width
	}
	if height > 0 {
		m.height = height
	}
}

func (m *Model) HandleKey(key string) tea.Cmd {
	if !m.open {
		return nil
	}
	switch key {
	case "esc":
		m.Close()
	case "r":
		return m.Open(m.path, true)
	}
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if !m.open {
		return nil
	}
	switch msg := msg.(type) {
	case streamStartedMsg:
		if msg.Request != m.request {
			return nil
		}
		m.stream = msg.Stream
		return waitProgress(msg.Request, msg.Stream)
	case progressMsg:
		if msg.Request != m.request {
			return nil
		}
		m.current = orbitfs.SizeResult{Bytes: msg.Progress.Bytes, Files: msg.Progress.Files, Directories: msg.Progress.Directories}
		m.err = msg.Progress.Err
		if m.err != nil {
			m.loading = false
			m.cancel = nil
			m.stream = nil
			return nil
		}
		if msg.Progress.Done {
			m.loading = false
			m.cancel = nil
			m.stream = nil
			if m.err == nil {
				m.cache[m.path] = cacheEntry{Result: m.current, At: time.Now()}
				m.cachedAt = m.cache[m.path].At
			}
			return nil
		}
		return waitProgress(msg.Request, m.stream)
	}
	return nil
}

func waitProgress(request uint64, stream <-chan orbitfs.SizeProgress) tea.Cmd {
	return func() tea.Msg {
		progress, ok := <-stream
		if !ok {
			// A normal scan emits its own Done event before closing. Treat an
			// unexpected close as an error so cancellation can never cache a
			// zero-valued result.
			return progressMsg{Request: request, Progress: orbitfs.SizeProgress{Err: errors.New("folder size scan stopped")}}
		}
		return progressMsg{Request: request, Progress: progress}
	}
}

func (m Model) View() string {
	if !m.open {
		return ""
	}
	lines := []string{"FOLDER SIZE", m.path, ""}
	if m.loading {
		lines = append(lines, fmt.Sprintf("Scanning…  %d files  %d folders", m.current.Files, m.current.Directories), fmt.Sprintf("%s discovered", formatBytes(m.current.Bytes)), "", "Esc Cancel")
	} else if m.err != nil {
		lines = append(lines, "Scan failed: "+m.err.Error(), "", "Esc Close")
	} else {
		lines = append(lines, formatBytes(m.current.Bytes), fmt.Sprintf("%d files  %d folders", m.current.Files, m.current.Directories))
		if !m.cachedAt.IsZero() {
			lines = append(lines, fmt.Sprintf("Calculated %s", age(m.cachedAt)))
		}
		lines = append(lines, "", "r Refresh  Esc Close")
	}
	if m.width < 64 || m.height < 16 {
		lines = []string{"FOLDER SIZE", formatPath(m.path)}
		if m.loading {
			lines = append(lines, fmt.Sprintf("Scanning… %d files", m.current.Files), "Esc cancel")
		} else if m.err != nil {
			lines = append(lines, "Scan failed", "Esc close")
		} else {
			lines = append(lines, formatBytes(m.current.Bytes), "r refresh  Esc close")
		}
	}
	return lipgloss.NewStyle().Width(max(1, m.width-6)).Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#F6C453")).Render(strings.Join(lines, "\n"))
}

func formatPath(path string) string {
	runes := []rune(path)
	if len(runes) <= 32 {
		return path
	}
	return "…" + string(runes[len(runes)-31:])
}

func formatBytes(size int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(size)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", size, units[unit])
	}
	return fmt.Sprintf("%.2f %s", value, units[unit])
}
func age(at time.Time) string {
	elapsed := time.Since(at)
	if elapsed < time.Second {
		return "just now"
	}
	if elapsed < time.Minute {
		return fmt.Sprintf("%ds ago", int(elapsed.Seconds()))
	}
	return fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
