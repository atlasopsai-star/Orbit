package orbitfs

import (
	"context"
	"io/fs"
	"path/filepath"
	"time"
)

type SizeResult struct {
	Bytes       int64
	Files       int64
	Directories int64
}
type SizeProgress struct {
	Bytes       int64
	Files       int64
	Directories int64
	Done        bool
	Err         error
}

func DirectorySize(ctx context.Context, root string) (SizeResult, error) {
	var result SizeResult
	for progress := range DirectorySizeStream(ctx, root) {
		if progress.Err != nil {
			return result, progress.Err
		}
		result = SizeResult{Bytes: progress.Bytes, Files: progress.Files, Directories: progress.Directories}
	}
	select {
	case <-ctx.Done():
		return result, ctx.Err()
	default:
		return result, nil
	}
}

func DirectorySizeStream(ctx context.Context, root string) <-chan SizeProgress {
	out := make(chan SizeProgress, 1)
	go func() {
		defer close(out)
		result := SizeResult{}
		entriesSinceEmit := 0
		lastEmit := time.Now()
		emit := func(done bool, err error) bool {
			progress := SizeProgress{Bytes: result.Bytes, Files: result.Files, Directories: result.Directories, Done: done, Err: err}
			select {
			case out <- progress:
				return true
			case <-ctx.Done():
				return false
			}
		}
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
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
			} else if info, err := entry.Info(); err == nil {
				result.Files++
				result.Bytes += info.Size()
			}
			entriesSinceEmit++
			if entriesSinceEmit >= 128 || time.Since(lastEmit) >= 100*time.Millisecond {
				if !emit(false, nil) {
					return ctx.Err()
				}
				entriesSinceEmit = 0
				lastEmit = time.Now()
			}
			return nil
		})
		if err != nil {
			if err == context.Canceled || err == context.DeadlineExceeded {
				return
			}
			_ = emit(true, err)
			return
		}
		_ = emit(true, nil)
	}()
	return out
}
