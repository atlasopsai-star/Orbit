package gitstatus

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type Snapshot struct {
	Root   string
	Branch string
	Files  map[string]string
}

func Detect(ctx context.Context, directory string) (Snapshot, error) {
	rootOutput, err := run(ctx, directory, "rev-parse", "--show-toplevel")
	if err != nil {
		return Snapshot{}, fmt.Errorf("not a git repository: %w", err)
	}
	root := filepath.Clean(strings.TrimSpace(rootOutput))
	branch, err := run(ctx, root, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		branch, _ = run(ctx, root, "rev-parse", "--short", "HEAD")
	}
	statusOutput, err := run(ctx, root, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return Snapshot{}, fmt.Errorf("read git status: %w", err)
	}
	return Snapshot{Root: root, Branch: strings.TrimSpace(branch), Files: ParsePorcelain(statusOutput, root)}, nil
}

func ParsePorcelain(output, root string) map[string]string {
	files := make(map[string]string)
	parts := strings.Split(output, "\x00")
	for index := 0; index < len(parts); index++ {
		record := parts[index]
		if len(record) < 4 {
			continue
		}
		code := strings.TrimSpace(record[:2])
		path := record[3:]
		if code == "" || path == "" {
			continue
		}
		if strings.HasPrefix(code, "R") || strings.HasPrefix(code, "C") {
			if index+1 < len(parts) {
				index++
				path = parts[index]
			}
		}
		files[filepath.Clean(filepath.Join(root, path))] = statusCode(code)
	}
	return files
}

func statusCode(code string) string {
	if strings.Contains(code, "U") {
		return "U"
	}
	if strings.Contains(code, "?") {
		return "?"
	}
	if strings.Contains(code, "D") {
		return "D"
	}
	if strings.Contains(code, "A") {
		return "A"
	}
	return "M"
}

func run(ctx context.Context, directory string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = directory
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
