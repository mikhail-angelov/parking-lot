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
go test -race ./...
go build ./...
golangci-lint run
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

## Telegram Mini App

The game runs inside Telegram as a WebApp. The frontend already integrates
the Telegram WebApp SDK (`public/index.html` → `telegram-web-app.js`, theme and
viewport handling in `public/app.js`), so the same static build works both in
a browser and inside Telegram.

**No bot code is required.** The menu button is a bot setting, so everything
is configured once in BotFather — there is nothing to host.

### Setup (one time, in Telegram)

1. Open **@BotFather** → `/newbot` → choose a name and username (e.g. `ParkingPuzzleBot`).
2. BotFather → your bot → **Bot Settings → Menu Button** → type **Web App**, text `🎮 Играть`, URL `https://mikhail-angelov.github.io/parking-lot/`.
3. (Recommended) BotFather → your bot → **Bot Settings → App** (or `/newapp`) with the same URL. This registers a proper Mini App: the game appears in the bot's apps list and opens full-screen.

Done: open the bot in Telegram → tap 🎮 → the game opens full-screen.
Difficulty packs load from the same GitHub Pages URL; progress is saved in
the WebApp's local storage.

A small bot (`cmd/bot`, dependency-free long polling) was considered for
`/start` greetings and menu-button setup, then dropped as unnecessary —
git history keeps it if a welcome screen, leaderboards or per-level chat
sharing ever become wanted.
