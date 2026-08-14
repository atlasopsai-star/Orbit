package internal

import (
	"reflect"
	"testing"

	"github.com/atlasopsai-star/Orbit/src/internal/ui/processbar"
)

func TestOperationMessagesPreserveAffectedPaths(t *testing.T) {
	paths := []string{"/tmp/source.txt", "/tmp/destination"}
	cases := []struct {
		name  string
		paths []string
	}{
		{name: "paste", paths: NewPasteOperationMsg(processbar.Successful, 1, paths...).affectedPaths},
		{name: "create", paths: NewCreateOperationMsg(processbar.Successful, 1, paths...).affectedPaths},
		{name: "delete", paths: NewDeleteOperationMsg(processbar.Successful, 1, paths...).affectedPaths},
		{name: "extract", paths: NewExtractOperationMsg(processbar.Successful, 1, paths...).affectedPaths},
		{name: "compress", paths: NewCompressOperationMsg(processbar.Successful, 1, paths...).affectedPaths},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !reflect.DeepEqual(tc.paths, paths) {
				t.Fatalf("affected paths = %#v, want %#v", tc.paths, paths)
			}
		})
	}
}
