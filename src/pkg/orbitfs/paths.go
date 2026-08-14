package orbitfs

import (
	"path/filepath"
	"strconv"
	"strings"
)

func CopyRelativePath(path, base string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absBase, absPath)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

func DuplicateName(dir, name string, exists func(string) bool) string {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	candidate := filepath.Join(dir, stem+" copy"+ext)
	for index := 2; exists(candidate); index++ {
		candidate = filepath.Join(dir, stem+" copy "+strconv.Itoa(index)+ext)
	}
	return candidate
}

func DuplicatePath(source string, exists func(string) bool) string {
	return DuplicateName(filepath.Dir(source), filepath.Base(source), exists)
}
