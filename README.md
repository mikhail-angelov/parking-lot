# Parking Puzzle

[![CI and GitHub Pages](https://github.com/mikhail-angelov/parking-lot/actions/workflows/ci-pages.yml/badge.svg)](https://github.com/mikhail-angelov/parking-lot/actions/workflows/ci-pages.yml)

A browser-based Rush Hour-style puzzle: move vehicles along their axes and clear the exit on the right for the red car.

**[Play online](https://mikhail-angelov.github.io/parking-lot/)**

The repository includes:

- a responsive browser game with no build step;
- 600 pre-generated levels with progressive difficulty;
- a Go engine for validating moves and finding optimal solutions with BFS;
- difficulty and quality analysis;
- a deterministic level-pack generator;
- a CLI for generating, solving, analyzing, and rendering levels as text.

## Controls

Drag a vehicle with a mouse or finger along its axis. You can also select a vehicle and move it with the arrow keys. Progress is saved locally in the browser.

## Run locally

The game loads levels over HTTP, so serve it with a local web server:

```sh
python3 -m http.server 8000 --directory public
```

Then open <http://localhost:8000>.

## Test

```sh
go test ./...
node --test public/game_test.js
```

## CLI

Common commands:

```sh
go run ./cmd/parking render path/to/level.json
go run ./cmd/parking solve path/to/level.json
go run ./cmd/parking analyze path/to/level.json
go run ./cmd/parking generate --difficulty medium --seed 1
```

To regenerate the level packs:

```sh
scripts/generate-levels.sh --packs 6 --pack-size 100 --seed 1 --workers 8
```

See the [project specification](docs/parking-puzzle-SPEC.md) for the full game model and architecture.

## Deployment

The `CI and GitHub Pages` workflow runs the tests on every push and pull request. After a successful push to `master`, it automatically deploys the contents of `public/` to GitHub Pages.

Before the first deployment, select **Settings → Pages → Build and deployment → Source: GitHub Actions** in the repository settings.
