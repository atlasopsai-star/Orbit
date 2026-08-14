package internal

import (
	zoxidelib "github.com/lazysegtree/go-zoxide"
	"sync"

	"github.com/atotto/clipboard"

	"github.com/atlasopsai-star/Orbit/src/internal/ui/helpmenu"

	"github.com/atlasopsai-star/Orbit/src/internal/ui/filemodel"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/foldersize"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/gitdiff"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/sortmodel"

	clipboardui "github.com/atlasopsai-star/Orbit/src/internal/ui/clipboard"
	lookupui "github.com/atlasopsai-star/Orbit/src/internal/ui/lookup"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/metadata"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/palette"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/processbar"
	searchui "github.com/atlasopsai-star/Orbit/src/internal/ui/search"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/sidebar"

	"github.com/atlasopsai-star/Orbit/src/internal/common"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/prompt"
	zoxideui "github.com/atlasopsai-star/Orbit/src/internal/ui/zoxide"
)

// Generate and return model containing default configurations for interface
// Maybe we can replace slice of strings with var args - Should we ?
// TODO: Move the configuration parameters to a ModelConfig struct.
// Something like `RendererConfig` struct for `Renderer` struct in ui/renderer package
// Or even better API like varargs lambda function opts
// which can be WithFooter(), WithXYZ()
// Lots of improvements are waiting on it
//   - Allow Sending thumbnailGeneratorNeeded as false to preview.New()
//     to prevent noise in test logs. Same with imagePreviewer
func defaultModelConfig(toggleDotFile, toggleFooter, firstUse bool,
	firstPanelPaths []string, zClient *zoxidelib.Client) *model {
	return &model{
		mu:              new(sync.Mutex),
		focusPanel:      nonePanelFocus,
		processBarModel: processbar.New(),
		clipboard:       clipboardui.New(),
		clipboardWriter: clipboard.WriteAll,
		sidebarModel:    sidebar.New(),
		fileMetaData:    metadata.New(),
		fileModel:       filemodel.New(firstPanelPaths, toggleDotFile),
		helpMenu:        helpmenu.New(),
		actionPalette:   palette.New(),
		searchModal:     searchui.New(),
		lookupModal:     lookupui.New(),
		folderSizeModal: foldersize.New(),
		gitDiffModal:    gitdiff.New(),
		promptModal:     prompt.DefaultModel(prompt.PromptMinHeight, prompt.PromptMinWidth),
		zoxideModal:     zoxideui.DefaultModel(zoxideui.ZoxideMinHeight, zoxideui.ZoxideMinWidth, zClient),
		sortModal:       sortmodel.New(),
		zClient:         zClient,
		modelQuitState:  notQuitting,
		toggleFooter:    toggleFooter,
		firstUse:        firstUse,
		hasTrash:        common.InitTrash(),
	}
}
