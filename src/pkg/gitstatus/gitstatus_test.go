package gitstatus

import (
	"reflect"
	"testing"
)

func TestParsePorcelain(t *testing.T) {
	root := "/Volumes/Atlas Drive/Orbit"
	output := " M src/main.go\x00?? notes/space name.txt\x00D  old.txt\x00UU conflict.go\x00R  old.go\x00new/renamed.go\x00"
	got := ParsePorcelain(output, root)
	want := map[string]string{
		root + "/src/main.go":          "M",
		root + "/notes/space name.txt": "?",
		root + "/old.txt":              "D",
		root + "/conflict.go":          "U",
		root + "/new/renamed.go":       "M",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%#v want=%#v", got, want)
	}
}
