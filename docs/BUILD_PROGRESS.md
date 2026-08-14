# Orbit build progress

## Completed

- Orbit foundation and Orbit Dark are committed on `orbit-foundation` and pushed to `origin`.
- Added a context-aware Orbit Actions registry and keyboard-first `Ctrl-K` palette.
- Added structured Finder, Terminal/editor launch, path-copy, Trash, duplicate, compress, extract, and folder-size actions.
- Search now streams bounded batches, cancels stale requests, caps renderable results at 200, emits multiple content matches with line numbers, skips heavy directories, and applies hierarchical `.gitignore` rules with negation support.
- Search Enter navigates the focused file panel to the containing directory and focuses the selected filename.
- Folder-size scanning now exposes truthful progress, Esc cancellation, refresh, and a session cache with stale-request protection.
- Added preferred macOS terminal configuration (`terminal`, `iterm2`, `ghostty`, `warp`) and preferred editor configuration.
- Hardened palette/search/folder-size/diff rendering for narrow dimensions and added adaptive file-panel action hints.
- Added asynchronous lightweight Git repository detection, branch display, visible file status markers, and a read-only Git diff modal.
- Improved directory preview metadata, bounded large-file messaging, and CSV/TSV table-style previews while retaining the existing preview architecture.

## Verified

- `go test ./...` passes.
- `go vet ./...` passes.
- `go build -o ./bin/orbit .` passes.
- Focused action, Git, filesystem, folder-size, palette, search, diff, file-panel, and preview tests pass.
- PTY dogfood passes at 120x40: startup, first-use dismissal, `Ctrl-K`, recursive search, visible `model.go` result, Enter navigation/focus, folder-size modal, Finder filtering, and clean quit.
- Narrow PTY launches at 80x24 and 60x20 remain alive and exit cleanly; the smallest terminal shows Orbit's existing honest size warning.
- External process calls pass paths as structured arguments; no shell path concatenation was added.

## Current

- Phase 4 implementation is ready for final review and focused commits on the external-SSD checkout `/Volumes/AtlasDrive/Atlas/projects/Orbit`.

## Remaining

- Run real disposable-macOS integration checks for each installed terminal/editor, Finder, Trash, and preferred-app configuration.
- Add Git-aware refresh invalidation after every file operation and optional staged/unstaged diff selection.
- Improve search `.gitignore` edge-case compatibility, result streaming UX polish, and content-substring highlighting.
- Add folder-size progress caching invalidation for more file operations and optional copy-info action.
- Expand archive preview, Markdown/structured preview-specific polish, and large-directory benchmark coverage.
- Profile 10,000/100,000-entry directories and tune rendering/search based on measurements.
- Website/legacy-upstream migration and release packaging remain later work.

## Blocked

- Automatic updates remain disabled until Orbit has a real release stream.
- No public release, Homebrew package, or `main` push was performed.
