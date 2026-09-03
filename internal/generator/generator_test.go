package generator

import (
	"parking/internal/analyzer"
	"parking/internal/engine"
	"parking/internal/puzzle"
	"strings"
	"testing"
)

func TestGenerateIsDeterministic(t *testing.T) {
	c := Defaults(analyzer.Easy)
	c.Seed = 7
	a, e := Generate(c)
	if e != nil {
		t.Fatal(e)
	}
	b, e := Generate(c)
	if e != nil {
		t.Fatal(e)
	}
	if CanonicalHash(a.Level) != CanonicalHash(b.Level) {
		t.Fatal("same seed produced different levels")
	}
	if !a.Analysis.Accepted || !analyzer.MatchesTier(a.Analysis.DifficultyScore, analyzer.Easy) {
		t.Fatalf("generated analysis=%+v", a.Analysis)
	}
	p, e := engine.Prepare(a.Level)
	if e != nil {
		t.Fatal(e)
	}
	if !solverValid(p, a.Solution) {
		t.Fatal("generated solution is invalid")
	}
}

func TestCloserIncludesQualityPenalty(t *testing.T) {
	lowQuality := analyzer.Analysis{DifficultyScore: 62.5, QualityScore: 60}
	highQuality := analyzer.Analysis{DifficultyScore: 60, QualityScore: 100}

	if closer(lowQuality, analyzer.Hard, highQuality) {
		t.Fatal("a lower-quality candidate must not win on tier distance alone")
	}
}

func TestGenerateRejectsUnknownDifficulty(t *testing.T) {
	config := Defaults(analyzer.Expert)
	config.Tier = analyzer.DifficultyTier("unknown")
	config.MaxAttempts = 1

	_, err := Generate(config)
	if err == nil || !strings.Contains(err.Error(), "invalid difficulty") {
		t.Fatalf("error=%v, want invalid difficulty", err)
	}
}

func TestGenerateRejectsNonPositiveAttemptLimit(t *testing.T) {
	config := Defaults(analyzer.Easy)
	config.MaxAttempts = 0

	_, err := Generate(config)
	if err == nil || !strings.Contains(err.Error(), "max attempts") {
		t.Fatalf("error=%v, want max attempts error", err)
	}
}

func TestDefaultsBoundWorkPerSeed(t *testing.T) {
	want := map[analyzer.DifficultyTier]int{
		analyzer.Easy:   100,
		analyzer.Medium: 1000,
		analyzer.Hard:   100,
		analyzer.Expert: 100,
	}
	for tier, expected := range want {
		if attempts := Defaults(tier).MaxAttempts; attempts != expected {
			t.Errorf("%s max attempts=%d, want %d", tier, attempts, expected)
		}
	}
}

func TestCanonicalHashIncludesTargetLength(t *testing.T) {
	level := puzzle.Level{
		Width: 6, Height: 6, Target: 0,
		Vehicles: []puzzle.Vehicle{
			{ID: 0, Orientation: puzzle.Horizontal, Length: 2, Fixed: 2},
			{ID: 1, Orientation: puzzle.Vertical, Length: 2, Fixed: 3},
		},
		Initial: puzzle.State{Positions: []uint8{0, 2}},
	}
	longTarget := level
	longTarget.Vehicles = append([]puzzle.Vehicle(nil), level.Vehicles...)
	longTarget.Vehicles[0].Length = 3

	if CanonicalHash(level) == CanonicalHash(longTarget) {
		t.Fatal("target length must participate in the canonical hash")
	}
}

func BenchmarkGenerateMedium(b *testing.B) {
	config := Defaults(analyzer.Medium)
	config.Seed = 200003
	for b.Loop() {
		generated, err := Generate(config)
		if err != nil {
			b.Fatal(err)
		}
		if !generated.Analysis.Accepted || !analyzer.MatchesTier(generated.Analysis.DifficultyScore, analyzer.Medium) {
			b.Fatalf("generated analysis=%+v", generated.Analysis)
		}
	}
}

func solverValid(p *engine.PreparedLevel, solution []puzzle.Move) bool {
	k := p.Level.InitialKey()
	for _, m := range solution {
		next, e := engine.ApplyMoveChecked(p, k, m)
		if e != nil {
			return false
		}
		k = next
	}
	return engine.IsSolved(p, k)
}
