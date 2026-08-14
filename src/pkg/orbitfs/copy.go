package orbitfs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyNoOverwrite copies a path without ever replacing an existing destination.
func CopyNoOverwrite(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(source)
		if err != nil {
			return err
		}
		return os.Symlink(target, destination)
	}
	if info.IsDir() {
		return copyDirectoryNoOverwrite(source, destination, info.Mode())
	}
	return copyFileNoOverwrite(source, destination, info.Mode())
}

func copyDirectoryNoOverwrite(source, destination string, mode os.FileMode) error {
	if err := os.Mkdir(destination, mode.Perm()); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		_ = os.Remove(destination)
		return err
	}
	for _, entry := range entries {
		if err := CopyNoOverwrite(filepath.Join(source, entry.Name()), filepath.Join(destination, entry.Name())); err != nil {
			_ = os.Remove(destination)
			return err
		}
	}
	return nil
}

func copyFileNoOverwrite(source, destination string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode.Perm())
	if err != nil {
		return err
	}
	if _, err = io.Copy(output, input); err != nil {
		_ = output.Close()
		_ = os.Remove(destination)
		return fmt.Errorf("copy %s: %w", source, err)
	}
	return output.Close()
}
