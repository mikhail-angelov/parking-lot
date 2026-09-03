package main

import (
	"flag"
	"fmt"
	"os"
	"parking/internal/analyzer"
	"parking/internal/engine"
	"parking/internal/generator"
	"parking/internal/levelio"
	"parking/internal/render"
	"parking/internal/solver"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: parking {render|solve|analyze|generate|generate-packs}")
		return
	}
	switch os.Args[1] {
	case "render":
		renderCmd(os.Args[2:])
	case "solve":
		solveCmd(os.Args[2:])
	case "analyze":
		analyzeCmd(os.Args[2:])
	case "generate":
		generateCmd(os.Args[2:])
	case "generate-packs":
		generatePacksCmd(os.Args[2:])
	default:
		os.Exit(2)
	}
}
func load(args []string) (*engine.PreparedLevel, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("level file required")
	}
	l, e := levelio.Load(args[0])
	if e != nil {
		return nil, e
	}
	return engine.Prepare(l)
}
func renderCmd(args []string) {
	p, e := load(args)
	if e != nil {
		fail(e)
	}
	fmt.Println(render.ASCII(p, p.Level.InitialKey()))
}
func solveCmd(args []string) {
	fs := flag.NewFlagSet("solve", flag.ExitOnError)
	show := fs.Bool("show-boards", false, "show solution boards")
	maxDepth := fs.Int("max-depth", 100, "maximum BFS depth")
	maxStates := fs.Int("max-states", 2000000, "maximum BFS states")
	fs.Parse(args)
	p, e := load(fs.Args())
	if e != nil {
		fail(e)
	}
	r := solver.Solve(p, solver.Options{MaxDepth: *maxDepth, MaxStates: *maxStates})
	fmt.Printf("Solved: %s\nLimit reached: %s\nOptimal moves: %d\nVisited states: %d\nExpanded states: %d\n", yes(r.Solved), yes(r.LimitReached), r.MinMoves, r.VisitedStates, r.ExpandedStates)
	k := p.Level.InitialKey()
	for i, m := range r.Solution {
		fmt.Printf("%d. vehicle %d: %d -> %d\n", i+1, m.Vehicle, m.From, m.To)
		if *show {
			k = engine.ApplyMove(p, k, m)
			fmt.Println(render.ASCII(p, k))
		}
	}
}
func analyzeCmd(args []string) {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	maxDepth := fs.Int("max-depth", 100, "maximum BFS depth")
	maxStates := fs.Int("max-states", 2000000, "maximum BFS states")
	fs.Parse(args)
	p, e := load(fs.Args())
	if e != nil {
		fail(e)
	}
	a := analyzer.Analyze(p, analyzer.Config{MaxDepth: *maxDepth, MaxStates: *maxStates})
	fmt.Printf("Solvable: %s\nDifficulty score: %.2f\nDifficulty tier: %s\nQuality score: %.2f\nMetrics: %+v\n", yes(a.Solvable), a.DifficultyScore, a.DifficultyTier, a.QualityScore, a.Metrics)
	for _, x := range a.RejectReasons {
		fmt.Println("Rejected:", x)
	}
}
func generateCmd(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	d := fs.String("difficulty", "medium", "")
	seed := fs.Int64("seed", 1, "")
	out := fs.String("output", "", "")
	count := fs.Int("count", 1, "")
	workers := fs.Int("workers", 1, "")
	fs.Parse(args)
	c := generator.Defaults(analyzer.DifficultyTier(*d))
	c.Seed = *seed
	if *count < 1 {
		fail(fmt.Errorf("count must be positive"))
	}
	if *count == 1 {
		g, e := generator.Generate(c)
		if e != nil {
			fail(e)
		}
		if *out != "" {
			id := fmt.Sprintf("%s-%d", g.Analysis.DifficultyTier, g.Seed)
			f := levelio.NewGeneratedFile(id, g.Level, g.Analysis, g.Solution, g.Seed)
			if e = levelio.WriteDataset(*out, []levelio.File{f}); e != nil {
				fail(e)
			}
			return
		}
		p, e := engine.Prepare(g.Level)
		if e != nil {
			fail(e)
		}
		fmt.Println(render.ASCII(p, g.Level.InitialKey()))
		return
	}
	levels, e := generator.Bulk(generator.BulkConfig{
		Generator: c,
		Count:     *count,
		Workers:   *workers,
		Progress:  generationProgressReporter("generate"),
	})
	if e != nil {
		fail(e)
	}
	if *out != "" {
		docs := make([]levelio.File, 0, len(levels))
		for i, g := range levels {
			id := fmt.Sprintf("%s-%06d", c.Tier, i)
			docs = append(docs, levelio.NewGeneratedFile(id, g.Level, g.Analysis, g.Solution, g.Seed))
		}
		if e = levelio.WriteDataset(*out, docs); e != nil {
			fail(e)
		}
		return
	}
	for _, g := range levels {
		p, e := engine.Prepare(g.Level)
		if e != nil {
			fail(e)
		}
		fmt.Println(render.ASCII(p, g.Level.InitialKey()))
	}
}

// generatePacksCmd generates a series of level packs with difficulty ramping
// smoothly from easy to hard across the requested number of packs, and
// writes a manifest so the web app can list and switch between them.
func generatePacksCmd(args []string) {
	fs := flag.NewFlagSet("generate-packs", flag.ExitOnError)
	packs := fs.Int("packs", 6, "number of packs to generate")
	size := fs.Int("pack-size", 100, "levels per pack")
	seed := fs.Int64("seed", 1, "base seed")
	workers := fs.Int("workers", 4, "parallel workers")
	dir := fs.String("out", "public/levels", "output directory")
	fs.Parse(args)
	if *packs < 1 || *size < 1 {
		fail(fmt.Errorf("packs and pack-size must be positive"))
	}
	if e := os.MkdirAll(*dir, 0755); e != nil {
		fail(e)
	}
	// Packs cover the playable scale from the former Medium band through the
	// former Expert band. Expert remains available for scores above that ramp.
	tiers := []analyzer.DifficultyTier{analyzer.Easy, analyzer.Medium, analyzer.Hard}
	stride := int64(*size)*1000 + 1
	manifest := make([]levelio.Pack, 0, *packs)
	generatedHashes := make(map[string]bool, *packs**size)
	for i := 0; i < *packs; i++ {
		tier := tiers[i*len(tiers)/(*packs)]
		id := fmt.Sprintf("pack-%03d", i+1)
		c := generator.Defaults(tier)
		c.Seed = *seed + int64(i)*stride
		levels, e := generator.Bulk(generator.BulkConfig{
			Generator:      c,
			Count:          *size,
			Workers:        *workers,
			ExcludedHashes: generatedHashes,
			Progress:       generationProgressReporter(id),
		})
		if e != nil {
			fmt.Fprintf(os.Stderr, "%s: skipped (%v)\n", id, e)
			continue
		}
		docs := make([]levelio.File, 0, len(levels))
		for j, g := range levels {
			id := fmt.Sprintf("%s-%03d-%04d", tier, i+1, j)
			docs = append(docs, levelio.NewGeneratedFile(id, g.Level, g.Analysis, g.Solution, g.Seed))
		}
		if e = levelio.WritePack(*dir, id, docs); e != nil {
			fail(e)
		}
		for _, level := range levels {
			generatedHashes[generator.CanonicalHash(level.Level)] = true
		}
		manifest = append(manifest, levelio.Pack{ID: id, File: id + ".json", Tier: string(tier), Count: len(docs), Index: i + 1})
		fmt.Printf("%s: %d levels (%s)\n", id, len(docs), tier)
	}
	if e := levelio.WriteManifest(*dir, manifest); e != nil {
		fail(e)
	}
}

func generationProgressReporter(label string) func(generator.Progress) {
	started := time.Now()
	lastPrinted := time.Time{}
	return func(progress generator.Progress) {
		now := time.Now()
		if progress.Accepted < progress.Target && !lastPrinted.IsZero() && now.Sub(lastPrinted) < 5*time.Second {
			return
		}
		fmt.Fprintln(os.Stderr, formatGenerationProgress(label, progress, now.Sub(started)))
		lastPrinted = now
	}
}

func formatGenerationProgress(label string, progress generator.Progress, elapsed time.Duration) string {
	eta := "считается"
	if progress.Accepted > 0 {
		remaining := progress.Target - progress.Accepted
		estimate := time.Duration(float64(elapsed) * float64(remaining) / float64(progress.Accepted))
		eta = estimate.Round(time.Second).String()
	}
	return fmt.Sprintf(
		"%s: принято %d/%d, обработано %d/%d, ошибки %d, дубликаты %d, прошло %s, ETA %s",
		label,
		progress.Accepted,
		progress.Target,
		progress.Processed,
		progress.MaxJobs,
		progress.Failures,
		progress.Duplicates,
		elapsed.Round(time.Second),
		eta,
	)
}
func yes(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
func fail(e error) { fmt.Fprintln(os.Stderr, e); os.Exit(1) }
