package filepanel

import "testing"

func TestWithinRoot(t *testing.T) {
	root := "/tmp/repo"
	for _, test := range []struct {
		path string
		want bool
	}{
		{path: root, want: true},
		{path: root + "/src", want: true},
		{path: root + "/src/file.go", want: true},
		{path: "/tmp/repository-other", want: false},
		{path: "/tmp/repo-sibling", want: false},
	} {
		if got := withinRoot(root, test.path); got != test.want {
			t.Fatalf("withinRoot(%q, %q) = %v, want %v", root, test.path, got, test.want)
		}
	}
}
