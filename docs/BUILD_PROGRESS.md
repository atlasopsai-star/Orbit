# Orbit build progress

## Completed

- Orbit foundation and Orbit Dark are committed on `orbit-foundation` and pushed to `origin`.
- Added a context-aware action registry and Orbit Actions palette on configurable `Ctrl-K`.
- Added structured macOS actions for Finder reveal, Terminal.app, installed editor apps, path copying, and the existing Trash backend.
- Added collision-safe duplicate copies and asynchronous folder-size scans with cancellation and stale-result protection.
- Added Orbit search surfaces for current-directory filtering, recursive filename search, and bounded content search.

## Verified

- `go test ./...` passes.
- `go vet ./...` passes.
- `go build -o ./bin/orbit .` passes.
- Focused action, filesystem, palette, and search tests pass.
- PTY dogfood passes at 120x40: startup, first-run dismissal, `Ctrl-K` palette, keyboard filtering, filename search modal, content search modal, Esc, navigation, and clean quit.
- All external process calls pass paths as structured arguments; no shell path concatenation was added.

## Current

- The working milestone is ready for a focused commit on `orbit-foundation`; the checkout remains on the external SSD at `/Volumes/AtlasDrive/Atlas/projects/Orbit`.

## Remaining

- Status-bar/action-hint polish and wider narrow-terminal visual review.
- Configurable terminal preference support beyond Terminal.app.
- Folder-size progress display, explicit cancel/refresh UX, and temporary caching.
- `.gitignore`-aware search, streaming result presentation, and richer result selection/open behavior.
- Broader preview, Git-awareness, storage-visibility, performance, and website/legacy-upstream migration work.

## Blocked

- Orbit automatic updates remain disabled until Orbit has a real release stream.
- No public release, Homebrew package, or `main` push was performed.
