package orbitfs

import (
	"context"
	"io/fs"
	"path/filepath"
)

type SizeResult struct {
	Bytes       int64
	Files       int64
	Directories int64
}

func DirectorySize(ctx context.Context, root string) (SizeResult, error) {
	var result SizeResult
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if entry.IsDir() {
			if path != root {
				result.Directories++
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		result.Files++
		result.Bytes += info.Size()
		return nil
	})
	return result, err
}
