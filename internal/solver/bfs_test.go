package solver

import (
	"parking/internal/engine"
	"parking/internal/puzzle"
	"testing"
)

func TestSolveFixture(t *testing.T) {
	l := puzzle.Level{Width: 6, Height: 6, Target: 0, Vehicles: []puzzle.Vehicle{{ID: 0, Orientation: puzzle.Horizontal, Length: 2, Fixed: 2}, {ID: 1, Orientation: puzzle.Vertical, Length: 2, Fixed: 2}}, Initial: puzzle.State{Positions: []uint8{0, 2}}}
	p, e := engine.Prepare(l)
	if e != nil {
		t.Fatal(e)
	}
	r := Solve(p, Options{})
	if !r.Solved || r.MinMoves != 1 {
		t.Fatalf("result=%+v", r)
	}
}

func TestExploreCountsMergedOptimalPaths(t *testing.T) {
	p := twoBlockerFixture(t)

	result := Explore(p, ExplorationOptions{MaxDepth: 3})

	if result.MinMoves != 2 {
		t.Fatalf("MinMoves=%d, want 2", result.MinMoves)
	}
	if result.OptimalSolutions != 18 {
		t.Fatalf("OptimalSolutions=%d, want 18", result.OptimalSolutions)
	}
	if len(result.StatesByDepth) < 4 || result.StatesByDepth[3] == 0 {
		t.Fatalf("StatesByDepth=%v, want exploration through depth 3", result.StatesByDepth)
	}
}

func TestSolveHonorsStateLimit(t *testing.T) {
	result := Solve(twoBlockerFixture(t), Options{MaxStates: 1})

	if result.Solved || !result.LimitReached || result.VisitedStates != 1 {
		t.Fatalf("result=%+v, want unsolved at the one-state limit", result)
	}
}

func TestSolveReportsDepthLimit(t *testing.T) {
	result := Solve(twoBlockerFixture(t), Options{MaxDepth: 1})

	if result.Solved || !result.LimitReached {
		t.Fatalf("result=%+v, want depth limit", result)
	}
	if result.MaxFrontier != result.VisitedStates-1 {
		t.Fatalf("max frontier=%d, want %d", result.MaxFrontier, result.VisitedStates-1)
	}
}

func twoBlockerFixture(t *testing.T) *engine.PreparedLevel {
	t.Helper()
	l := puzzle.Level{
		Width: 6, Height: 6, Target: 0,
		Vehicles: []puzzle.Vehicle{
			{ID: 0, Orientation: puzzle.Horizontal, Length: 2, Fixed: 2},
			{ID: 1, Orientation: puzzle.Vertical, Length: 2, Fixed: 2},
			{ID: 2, Orientation: puzzle.Vertical, Length: 2, Fixed: 3},
		},
		Initial: puzzle.State{Positions: []uint8{0, 2, 2}},
	}
	p, err := engine.Prepare(l)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
