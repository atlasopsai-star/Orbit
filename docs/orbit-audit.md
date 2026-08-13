# Orbit audit

Date: 2026-08-13
Baseline: `a97381e` (`orbit-baseline-2026-08-13`)

## What already works well

- Mature Go + Bubble Tea v2 + Lip Gloss terminal file-manager architecture.
- Multi-pane navigation, previews, themes, hotkeys, plugins, file operations, and process UI are already implemented upstream.
- Baseline `go build ./...` succeeds and the complete `go test ./...` suite passes.
- The fork has a clean upstream snapshot and now tracks Superfile as `upstream`.

## Risks and UX opportunities

- The fork still contains many upstream-facing names in packaging, assets, docs, and internal identifiers; these need classification rather than blind replacement.
- Expensive directory metadata, previews, and recursive search require continued profiling on external volumes and large trees.
- macOS-specific actions (Finder, Trash, Terminal, editors, Quick Look) should be added behind tested, cancellable operations.
- Configuration migration must never overwrite an existing Superfile configuration; Orbit currently uses a separate namespace and does not auto-migrate.
- The command palette, favorites/recent locations, conflict handling, safe undo, and Git diff preview are high-value follow-up work.

## Milestone 1 changes in this branch

- Orbit module imports and Go module path.
- Orbit XDG config/cache/data/state namespace and log filename.
- Orbit embedded configuration directory and update target/default.
- Orbit CLI identity, window title, build output, and core docs.
- Explicit Superfile ancestry retained in legal attribution.
- Stable compatibility names such as `SuperFile*` Go variables and `open_spf_prompt` TOML keys remain internal/config API surfaces; they are documented rather than renamed.
- Orbit Dark is now the default embedded theme and is also the runtime fallback when a selected theme cannot be loaded.
