package actions

import (
	"sort"
	"strings"
)

type Context struct {
	SelectedPath     string
	CurrentDirectory string
	IsDirectory      bool
	IsMac            bool
	HasTrash         bool
	AvailableApps    map[string]bool
	GitStatus        string
}

type Action struct {
	ID          string
	Label       string
	Description string
	Category    string
	Keywords    []string
	Destructive bool
	Shortcut    string
	Platform    string
	Available   func(Context) bool
}

func all(ctx Context) []Action {
	app := func(name string) bool { return ctx.AvailableApps != nil && ctx.AvailableApps[name] }
	return []Action{
		{ID: "open", Label: "Open", Description: "Open the selected item", Category: "Open", Keywords: []string{"launch"}, Available: func(Context) bool { return ctx.SelectedPath != "" }},
		{ID: "reveal-finder", Label: "Reveal in Finder", Description: "Show the selected item in Finder", Category: "Mac", Keywords: []string{"finder", "show"}, Available: func(Context) bool { return ctx.IsMac && ctx.SelectedPath != "" }},
		{ID: "open-terminal", Label: "Open Terminal Here", Description: "Open a terminal in this location", Category: "Mac", Keywords: []string{"shell", "terminal", "console"}, Available: func(Context) bool { return ctx.IsMac && ctx.SelectedPath != "" }},
		{ID: "open-cursor", Label: "Open in Cursor", Description: "Open the selected item in Cursor", Category: "Developer", Keywords: []string{"editor", "code"}, Available: func(Context) bool { return app("Cursor") && ctx.SelectedPath != "" }},
		{ID: "open-vscode", Label: "Open in VS Code", Description: "Open the selected item in Visual Studio Code", Category: "Developer", Keywords: []string{"editor", "code", "visual studio"}, Available: func(Context) bool { return app("Visual Studio Code") && ctx.SelectedPath != "" }},
		{ID: "open-zed", Label: "Open in Zed", Description: "Open the selected item in Zed", Category: "Developer", Keywords: []string{"editor", "code"}, Available: func(Context) bool { return app("Zed") && ctx.SelectedPath != "" }},
		{ID: "open-xcode", Label: "Open in Xcode", Description: "Open the selected item in Xcode", Category: "Developer", Keywords: []string{"editor", "project"}, Available: func(Context) bool { return app("Xcode") && ctx.SelectedPath != "" }},
		{ID: "copy-absolute-path", Label: "Copy Absolute Path", Description: "Copy the selected item's full path", Category: "File", Keywords: []string{"clipboard", "path"}, Available: func(Context) bool { return ctx.SelectedPath != "" }},
		{ID: "copy-relative-path", Label: "Copy Relative Path", Description: "Copy the path relative to the current directory", Category: "File", Keywords: []string{"clipboard", "path"}, Available: func(Context) bool { return ctx.SelectedPath != "" && ctx.CurrentDirectory != "" }},
		{ID: "move-to-trash", Label: "Move to Trash", Description: "Move the selected item to the system Trash", Category: "File", Keywords: []string{"delete", "remove", "recycle"}, Destructive: true, Available: func(ctx Context) bool { return ctx.SelectedPath != "" && ctx.HasTrash }},
		{ID: "duplicate", Label: "Duplicate", Description: "Create a safe copy without overwriting", Category: "File", Keywords: []string{"copy", "clone"}, Available: func(Context) bool { return ctx.SelectedPath != "" }},
		{ID: "folder-size", Label: "Calculate Folder Size", Description: "Scan the selected folder asynchronously", Category: "Utility", Keywords: []string{"size", "disk", "storage"}, Available: func(Context) bool { return ctx.IsDirectory }},
		{ID: "compress", Label: "Compress", Description: "Create an archive from the selected item", Category: "Utility", Keywords: []string{"archive", "zip"}, Available: func(Context) bool { return ctx.SelectedPath != "" }},
		{ID: "extract", Label: "Extract", Description: "Extract the selected archive", Category: "Utility", Keywords: []string{"archive", "unzip"}, Available: func(ctx Context) bool {
			return ctx.SelectedPath != "" && !ctx.IsDirectory && isArchive(ctx.SelectedPath)
		}},
		{ID: "git-diff", Label: "View Git Diff", Description: "Show the selected file's unstaged diff", Category: "Git", Keywords: []string{"git", "changes", "modified"}, Available: func(ctx Context) bool {
			return ctx.SelectedPath != "" && !ctx.IsDirectory && ctx.GitStatus != "" && ctx.GitStatus != "?"
		}},
		{ID: "search-files", Label: "Search Current Directory", Description: "Filter the current directory by filename", Category: "Search", Keywords: []string{"find"}, Available: func(Context) bool { return ctx.CurrentDirectory != "" }},
		{ID: "search-recursive", Label: "Search Descendants", Description: "Search descendant filenames", Category: "Search", Keywords: []string{"find", "recursive"}, Available: func(Context) bool { return ctx.CurrentDirectory != "" }},
		{ID: "search-content", Label: "Search File Contents", Description: "Search text inside descendant files", Category: "Search", Keywords: []string{"grep", "content", "text"}, Available: func(Context) bool { return ctx.CurrentDirectory != "" }},
		{ID: "lookup", Label: "File & Folder Lookup", Description: "Find files and folders anywhere on this Mac", Category: "Search", Keywords: []string{"find", "lookup", "global", "everywhere", "spotlight"}, Available: func(Context) bool { return true }},
	}
}

func Default(ctx Context) []Action { return New(all(ctx)).Available(ctx) }

func (r Registry) Available(ctx Context) []Action {
	out := make([]Action, 0, len(r.items))
	for _, item := range r.items {
		if item.Available == nil || item.Available(ctx) {
			out = append(out, item)
		}
	}
	return out
}

type Registry struct{ items []Action }

func New(items []Action) Registry  { return Registry{items: append([]Action(nil), items...)} }
func (r Registry) Items() []Action { return append([]Action(nil), r.items...) }

func Filter(items []Action, query string) []Action {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return append([]Action(nil), items...)
	}
	type ranked struct {
		action Action
		score  int
		order  int
	}
	rankedItems := make([]ranked, 0, len(items))
	for i, item := range items {
		haystack := strings.ToLower(strings.Join(append([]string{item.Label, item.Description, item.Category}, item.Keywords...), " "))
		if score, ok := fuzzyScore(haystack, query); ok {
			rankedItems = append(rankedItems, ranked{item, score, i})
		}
	}
	sort.SliceStable(rankedItems, func(i, j int) bool {
		if rankedItems[i].score != rankedItems[j].score {
			return rankedItems[i].score > rankedItems[j].score
		}
		return rankedItems[i].order < rankedItems[j].order
	})
	out := make([]Action, 0, len(rankedItems))
	for _, item := range rankedItems {
		out = append(out, item.action)
	}
	return out
}

func fuzzyScore(haystack, query string) (int, bool) {
	best, matched := 0, false
	for _, token := range strings.Fields(haystack) {
		score, ok := fuzzyTokenScore(token, query)
		if ok && (!matched || score > best) {
			best, matched = score, true
		}
	}
	return best, matched
}

func fuzzyTokenScore(token, query string) (int, bool) {
	if strings.Contains(token, query) {
		return 1000 - len(token), true
	}
	runes, needles := []rune(token), []rune(query)
	cursor, score, previous := 0, 0, -2
	for _, needle := range needles {
		found := -1
		for index := cursor; index < len(runes); index++ {
			if runes[index] == needle {
				found = index
				break
			}
		}
		if found < 0 {
			return 0, false
		}
		if found == previous+1 {
			score += 6
		} else {
			score++
		}
		if found == 0 {
			score += 3
		}
		previous, cursor = found, found+1
	}
	return score - (len(runes)-len(needles))/20, true
}

func isArchive(path string) bool {
	lower := strings.ToLower(path)
	for _, ext := range []string{".zip", ".tar", ".gz", ".tgz", ".bz2", ".xz", ".7z", ".rar"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}
