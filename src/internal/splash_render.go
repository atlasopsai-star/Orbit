package internal

import (
	"math"
	"time"

	tea "charm.land/bubbletea/v2"

	variable "github.com/atlasopsai-star/Orbit/src/config"
	"github.com/atlasopsai-star/Orbit/src/internal/common"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/splash"
)

// splashMinDuration is how long the shimmering ORBIT intro plays before the
// TUI is revealed. Any keypress skips it early.
const splashMinDuration = 2400 * time.Millisecond

// splashSweepPeriod controls how often a highlight band sweeps the logo.
const splashSweepPeriod = 1400 * time.Millisecond

type splashTickMsg struct{}

func (m *model) splashTickCmd() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg { return splashTickMsg{} })
}

func (m *model) splashRender() string {
	width, height := m.fullWidth, m.fullHeight
	if width <= 0 {
		width = 120
	}
	if height <= 0 {
		height = 36
	}
	elapsed := time.Since(m.splashStart)
	progress := math.Min(1, float64(elapsed)/float64(splashMinDuration))
	phase := float64(elapsed) / float64(splashSweepPeriod)
	return splash.Render(width, height, progress, phase,
		variable.CurrentVersion+variable.PreReleaseSuffix,
		common.Theme.FullScreenBG, common.Theme.FullScreenFG)
}
