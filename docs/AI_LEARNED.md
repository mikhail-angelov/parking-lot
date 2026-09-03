# Verified Codex workflows

## 2026-09-02 — Fast tiered level generation

### Goal

Regenerate the six 100-level packs without spending tens of minutes on each
Medium seed or exhausting the retry budget for the rare Hard score band.

### Golden path

1. Keep the generator-only optimal-move floor in `analyzer.AnalyzeForTier`, so
   candidates below the requested tier are rejected before `solver.Explore`.
2. Reuse move buffers in both passes of `solver.Explore`.
3. Use the calibrated per-seed budgets from `generator.Defaults`: 100 attempts
   for Easy/Hard/Expert and 1000 for Medium.
4. Use bulk retry ceilings of 3× requested count for Easy/Medium and 80× for
   Hard/Expert; generation stops immediately after the requested count.
5. Pass canonical hashes from completed packs through `BulkConfig.ExcludedHashes`
   so deduplication covers the entire generated collection, not just one pack.
6. Run `scripts/generate-levels.sh --packs 6 --pack-size 100 --seed 1 --workers 8`.

### Verification

The full command completed all six packs. Easy took about one second per pack,
Medium about one minute per pack, and Hard about seven/eight minutes. JSON
validation confirmed 600 accepted levels in their requested score bands, with
100 levels per pack and JSON as the only generated data format.
`TestGeneratedPacksAreCanonicallyUnique` independently confirmed 600 unique
canonical hashes. `go test ./...`, `go test -race ./...`, and `go vet ./...`
passed.

### Failure pattern avoided

Running full difficulty analysis for candidates that cannot meet the tier's
minimum move count allocates heavily in `solver.Explore`. Before the early
filter, one fixed Medium seed took more than 35 seconds and 100 attempts
allocated about 1.36 GB; the full Medium pack reported an ETA around 45 minutes.

### Ruled-out approaches

- Tried a 100-attempt cap for Medium; failed because the deterministic
  benchmark seed did not find a valid Medium level within that budget.
- Tried 12× and 30× Hard retry ceilings; failed because a five-level sample
  stopped at 4/5 accepted levels.

### Notes

The fixed Medium benchmark is `BenchmarkGenerateMedium`; run it with
`go test ./internal/generator -run '^$' -bench '^BenchmarkGenerateMedium$' -benchtime=1x -benchmem -count=1`.

## 2026-09-03 — Preview generated SVG sprites with Quick Look

### Goal

Visually verify SVG sprites assembled as strings in `public/app.js` without a
browser automation dependency.

### Golden path

1. Evaluate only the sprite-builder functions with Node and write their SVG
   output to a directory created by `mktemp -d`.
2. Render each SVG with `qlmanage -t -s 600 -o <preview-dir> <sprite.svg>`.
3. Combine the PNGs into a contact sheet with ImageMagick when comparing
   several colors or vehicle lengths.

### Verification

Quick Look rendered the body and glass gradients, lights, wheels, and outlines
of both car and truck sprites correctly; the contact sheet was then inspected.

### Failure pattern avoided

ImageMagick's SVG renderer can display fills using local gradient references
such as `url(#car2t-body)` as black, producing a misleading broken preview even
when WebKit-compatible rendering is correct.

### Ruled-out approaches

- Tried rasterizing the generated SVGs directly with `magick`; both numeric and
  percentage gradient coordinates still rendered referenced gradients black.

### Notes

This workflow uses macOS Quick Look and is intended for local visual checks;
the application still renders the same SVG strings through the browser canvas.
