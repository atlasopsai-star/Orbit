package internal

import (
	"context"
	"sync"

	zoxidelib "github.com/lazysegtree/go-zoxide"

	"github.com/atlasopsai-star/Orbit/src/internal/ui/helpmenu"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/spferror"

	"github.com/atlasopsai-star/Orbit/src/internal/ui/clipboard"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/sortmodel"

	"github.com/atlasopsai-star/Orbit/src/internal/ui/filemodel"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/foldersize"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/gitdiff"
	"github.com/atlasopsai-star/Orbit/src/pkg/gitstatus"

	"github.com/atlasopsai-star/Orbit/src/internal/ui/metadata"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/notify"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/palette"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/processbar"
	searchui "github.com/atlasopsai-star/Orbit/src/internal/ui/search"
	"github.com/atlasopsai-star/Orbit/src/internal/ui/sidebar"

	"charm.land/bubbles/v2/textinput"

	"github.com/atlasopsai-star/Orbit/src/internal/ui/prompt"
	zoxideui "github.com/atlasopsai-star/Orbit/src/internal/ui/zoxide"
)

// Type representing the type of focused panel
type focusPanelType int

type modelQuitStateType int

// Constants for panel with no focus
const (
	nonePanelFocus focusPanelType = iota
	processBarFocus
	sidebarFocus
	metadataFocus
)

const (
	notQuitting modelQuitStateType = iota
	quitInitiated
	quitConfirmationInitiated
	quitConfirmationReceived
	quitDone
)

// Main model
// TODO : We could consider using *model as tea.Model, instead of model.
// for reducing re-allocations. The struct is 20K bytes. But this could lead to
// issues like race conditions and whatnot, which are hidden since we are creating
// new model in each tea update.
type model struct {
	// Main Panels
	fileModel       filemodel.Model
	sidebarModel    sidebar.Model
	processBarModel processbar.Model
	clipboard       clipboard.Model
	clipboardWriter func(string) error
	focusPanel      focusPanelType

	// Modals
	notifyModel     notify.Model
	actionPalette   palette.Model
	searchModal     searchui.Model
	folderSizeModal foldersize.Model
	gitDiffModal    gitdiff.Model
	typingModal     typingModal
	helpMenu        helpmenu.Model
	promptModal     prompt.Model
	zoxideModal     zoxideui.Model
	sortModal       sortmodel.Model
	spfError        spferror.Model
	mutexErrorModal sync.Mutex

	// Zoxide client for directory tracking
	zClient *zoxidelib.Client

	fileMetaData metadata.Model

	// no use directly for increment, use nextIoReqCnt
	ioReqCnt int32

	modelQuitState       modelQuitStateType
	firstTextInput       bool
	toggleFooter         bool
	firstLoadingComplete bool
	firstUse             bool

	// This entirely disables metadata fetching. Used in test model
	disableMetadata bool

	// Height in number of lines of actual viewport of
	// main panel and sidebar excluding border
	mainPanelHeight int

	// Height in number of lines of actual viewport of
	// footer panels - process/metadata/clipboard - excluding border
	footerHeight int
	fullWidth    int
	fullHeight   int

	// whether usable trash directory exists or not
	hasTrash bool

	// Async Git snapshot state. The checked flag distinguishes a scanned
	// non-repository from a location that has not been requested yet.
	gitStatus   gitstatus.Snapshot
	gitLocation string
	gitRequest  uint64
	gitCancel   context.CancelFunc
	gitChecked  bool
	gitLoading  bool
}

type typingModal struct {
	location  string
	open      bool
	textInput textinput.Model
}

type editorFinishedMsg struct{ err error }
