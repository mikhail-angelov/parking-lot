package analyzer

import (
	"parking/internal/engine"
	"parking/internal/puzzle"
	"parking/internal/solver"
	"testing"
)

func analysisFixture(pos uint8) *engine.PreparedLevel {
	l := puzzle.Level{Width: 6, Height: 6, Target: 0, Vehicles: []puzzle.Vehicle{{ID: 0, Orientation: puzzle.Horizontal, Length: 2, Fixed: 2}, {ID: 1, Orientation: puzzle.Vertical, Length: 2, Fixed: 2}}, Initial: puzzle.State{Positions: []uint8{0, pos}}}
	p, _ := engine.Prepare(l)
	return p
}
func TestAnalysisDirectBlocker(t *testing.T) {
	a := Analyze(analysisFixture(2), Config{})
	if !a.Solvable {
		t.Fatal("fixture should solve")
	}
	if a.Metrics.InitialBlockers != 1 {
		t.Fatalf("blockers=%d", a.Metrics.InitialBlockers)
	}
	if a.Metrics.OptimalMoves != 1 {
		t.Fatalf("moves=%d", a.Metrics.OptimalMoves)
	}
	if a.Metrics.DistractingRatio != 0.25 {
		t.Fatalf("distracting ratio=%f, want 0.25", a.Metrics.DistractingRatio)
	}
	if a.Metrics.ForcedMoveRatio != 0 {
		t.Fatalf("forced move ratio=%f, want 0", a.Metrics.ForcedMoveRatio)
	}
}
func TestAnalysisRejectsInitiallySolved(t *testing.T) {
	p := analysisFixture(0)
	a := Analyze(p, Config{})
	if !a.Solvable || a.QualityScore != 0 || len(a.RejectReasons) == 0 {
		t.Fatalf("analysis=%+v", a)
	}
}

func TestAnalysisFindsDependencyDepthTwo(t *testing.T) {
	l := puzzle.Level{
		Width: 6, Height: 6, Target: 0,
		Vehicles: []puzzle.Vehicle{
			{ID: 0, Orientation: puzzle.Horizontal, Length: 2, Fixed: 2},
			{ID: 1, Orientation: puzzle.Vertical, Length: 2, Fixed: 2},
			{ID: 2, Orientation: puzzle.Horizontal, Length: 2, Fixed: 0},
			{ID: 3, Orientation: puzzle.Horizontal, Length: 2, Fixed: 3},
		},
		Initial: puzzle.State{Positions: []uint8{0, 1, 1, 1}},
	}
	p, err := engine.Prepare(l)
	if err != nil {
		t.Fatal(err)
	}

	analysis := Analyze(p, Config{})

	if !analysis.Solvable {
		t.Fatal("fixture should solve")
	}
	if analysis.Metrics.DependencyDepth != 2 {
		t.Fatalf("dependency depth=%d, want 2", analysis.Metrics.DependencyDepth)
	}
}

func TestAnalysisReportsSearchLimit(t *testing.T) {
	analysis := Analyze(analysisFixture(2), Config{MaxStates: 1})

	if !analysis.LimitReached {
		t.Fatal("expected search limit to be reported")
	}
	if len(analysis.RejectReasons) != 1 || analysis.RejectReasons[0] != "search limit reached" {
		t.Fatalf("reject reasons=%v", analysis.RejectReasons)
	}
}

func TestMatchesTierUsesShiftedDifficultyScale(t *testing.T) {
	tests := []struct {
		score float64
		tier  DifficultyTier
		want  bool
	}{
		{score: 24.99, tier: Easy, want: false},
		{score: 25, tier: Easy, want: true},
		{score: 49.99, tier: Easy, want: true},
		{score: 50, tier: Medium, want: true},
		{score: 74.99, tier: Medium, want: true},
		{score: 75, tier: Hard, want: true},
		{score: 89.99, tier: Hard, want: true},
		{score: 90, tier: Expert, want: true},
	}

	for _, test := range tests {
		if got := MatchesTier(test.score, test.tier); got != test.want {
			t.Errorf("MatchesTier(%v, %q)=%v, want %v", test.score, test.tier, got, test.want)
		}
	}
}

func TestAnalysisHonorsExplorationDepthLimit(t *testing.T) {
	level := puzzle.Level{
		Width: 6, Height: 6, Target: 0,
		Vehicles: []puzzle.Vehicle{
			{ID: 0, Orientation: puzzle.Horizontal, Length: 2, Fixed: 2},
			{ID: 1, Orientation: puzzle.Vertical, Length: 2, Fixed: 2},
			{ID: 2, Orientation: puzzle.Horizontal, Length: 2, Fixed: 0},
		},
		Initial: puzzle.State{Positions: []uint8{0, 2, 0}},
	}
	prepared, err := engine.Prepare(level)
	if err != nil {
		t.Fatal(err)
	}
	want := solver.Explore(prepared, solver.ExplorationOptions{MaxDepth: 1}).ReachableStates

	analysis := Analyze(prepared, Config{MaxDepth: 1})

	if analysis.Metrics.ReachableStates != want {
		t.Fatalf("reachable states=%d, want %d at configured depth", analysis.Metrics.ReachableStates, want)
	}
}

func TestTierAnalysisRejectsMoveFloorBeforeExploration(t *testing.T) {
	analysis := AnalyzeForTier(analysisFixture(2), Config{}, Easy)

	if analysis.Accepted || analysis.Metrics.OptimalMoves != 1 {
		t.Fatalf("analysis=%+v", analysis)
	}
	if analysis.Metrics.ReachableStates != 0 {
		t.Fatalf("reachable states=%d, want no expensive exploration", analysis.Metrics.ReachableStates)
	}
	if len(analysis.RejectReasons) != 1 || analysis.RejectReasons[0] != "too few optimal moves" {
		t.Fatalf("reject reasons=%v", analysis.RejectReasons)
	}
}

func TestAnalysisKeepsCompleteMetricsWithQualityMoveFloor(t *testing.T) {
	analysis := Analyze(analysisFixture(2), Config{MinOptimalMoves: 2})

	if analysis.Metrics.ReachableStates == 0 {
		t.Fatal("general analysis must not return partial metrics")
	}
}
