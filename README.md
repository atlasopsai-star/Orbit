# Orbit

<p align="center">
  <strong>Orbit</strong><br>
  <em>A fast, beautiful, keyboard-first terminal file manager for Mac developers.</em>
</p>

Orbit is a polished evolution of [Superfile](https://github.com/yorukot/superfile): the familiar multi-pane terminal file-manager workflow, with an Orbit identity and a focused roadmap for safer file operations, macOS integration, fast search, useful previews, Git awareness, and responsive performance.

## Status

Orbit is in active foundation development. Existing file-manager functionality is preserved while the product is migrated and improved in small, tested milestones.

## Build from source

Requirements: Go 1.26+ and a terminal with Unicode support.

```bash
git clone https://github.com/atlasopsai-star/Orbit.git
cd Orbit
go build -o ./bin/orbit .
./bin/orbit .
```

Development commands are also available through the Makefile:

```bash
make test
make build
```

## Configuration

Orbit uses its own XDG namespace and will not overwrite an existing Superfile configuration:

- Linux: `~/.config/orbit/`
- macOS: the corresponding application-support `orbit` namespace from the XDG implementation
- State, cache, and data paths use the corresponding `orbit` namespace.

The default files include `config.toml`, `hotkeys.toml`, and theme files. Orbit ships with `Orbit Dark`, a deep-navy theme with gold focus states and cyan navigation accents. Update checks are disabled by default until Orbit publishes releases.

## Core controls

Orbit retains the upstream keyboard-first workflow. Press `?` inside Orbit for the complete help overlay and configured shortcuts.

## Roadmap

The next milestones prioritize visual polish, macOS actions (Finder, Trash, Terminal, Cursor, VS Code, Zed, and Xcode), quick actions and command search, deterministic filename/content search, safer conflicts and progress, richer previews, lightweight Git status, and large-directory performance.

## Acknowledgments and license

Orbit is based on Superfile by Yorukot and contributors. The upstream MIT license and third-party notices are preserved in [`LICENSE`](LICENSE) and [`NOTICE.md`](NOTICE.md) and [`ATTRIBUTION.md`](ATTRIBUTION.md). Orbit changes are maintained by Atlas Ops AI.

Orbit is distributed under the MIT License.
