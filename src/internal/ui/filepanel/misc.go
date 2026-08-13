package filepanel

import "github.com/atlasopsai-star/Orbit/src/internal/common"

func (p PanelMode) String() string {
	switch p {
	case SelectMode:
		return "selectMode"
	case BrowserMode:
		return "browserMode"
	default:
		return common.InvalidTypeString
	}
}
