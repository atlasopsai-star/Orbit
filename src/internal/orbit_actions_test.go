package internal

import (
	"reflect"
	"testing"
)

func TestOrbitTerminalCommandUsesConfiguredAppAndStructuredPath(t *testing.T) {
	path := "/Volumes/Atlas Drive/資料/Project"
	for _, test := range []struct {
		name string
		want string
	}{
		{name: "", want: "Terminal"},
		{name: "terminal", want: "Terminal"},
		{name: "iterm2", want: "iTerm"},
		{name: "ghostty", want: "Ghostty"},
		{name: "warp", want: "Warp"},
		{name: "unknown", want: "Terminal"},
	} {
		t.Run(test.name, func(t *testing.T) {
			command, args := orbitTerminalCommand(path, test.name)
			if command != "open" || !reflect.DeepEqual(args, []string{"-a", test.want, path}) {
				t.Fatalf("command=%q args=%#v", command, args)
			}
		})
	}
}
