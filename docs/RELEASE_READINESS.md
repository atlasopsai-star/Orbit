# Orbit release-readiness audit

## Local release candidate

Verified on the Apple Silicon development machine:

- `go build -o ./bin/orbit .` succeeds.
- `bin/orbit --version` reports `orbit version v1.6.0`.
- A copied binary runs from a temporary directory without the source checkout.
- Help, configuration bootstrap, theme loading, and isolated XDG paths work outside the repository.

Orbit does not publish a release from this branch. No updater or public distribution channel is enabled.

## Automation isolation

The following inherited workflows are intentionally manual and job-disabled:

- `.github/workflows/mirror.yml` — previously targeted `yorukot/superfile` on Codeberg.
- `.github/workflows/winget.yml` — previously used the `yorukot.superfile` WinGet identity.
- `.github/workflows/update-gomod2nix.yml` — inherited dependency automation is not yet reviewed for Orbit ownership.

They must not be re-enabled until Orbit has reviewed destinations, credentials, package identities, and push permissions. No GitHub release, Homebrew publication, WinGet submission, upstream mirror, or automatic update stream is part of Phase 5.

## Branding audit

Runtime product surfaces use Orbit branding and the `orbit` binary. Remaining Superfile references are classified as one of:

- required attribution/license and fork-history documentation;
- inherited website/install scaffolding that is not a Phase 5 publication surface;
- internal compatibility names and cleanup debt.

The website and installer scaffolding must receive a separate Orbit-owned distribution decision before publication. Do not treat those inherited assets as production-ready Orbit installers.

## Verification notes

- Focused race checks pass for `src/internal/ui/preview`, `src/internal/ui/foldersize`, and `src/pkg/orbitfs`.
- `go test -race ./...` currently reports pre-existing races in broader internal file-panel/processbar tests. This is tracked as a release blocker for a public beta; Phase 5 does not mask or suppress those reports.
- PTY launches exited cleanly at 160x50, 120x40, 100x30, 80x24, and 60x20. Existing dogfood evidence remains the source of truth for semantic palette/search/help checkpoints because the automated ANSI checkpoint detector is not reliable on every render path.
