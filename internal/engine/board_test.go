package engine

import (
	"parking/internal/puzzle"
	"testing"
)

func fixture() puzzle.Level {
	return puzzle.Level{Width: 6, Height: 6, Target: 0, Vehicles: []puzzle.Vehicle{{ID: 0, Orientation: puzzle.Horizontal, Length: 2, Fixed: 2}, {ID: 1, Orientation: puzzle.Vertical, Length: 2, Fixed: 2}}, Initial: puzzle.State{Positions: []uint8{0, 2}}}
}
func TestMovesAndGoal(t *testing.T) {
	p, e := Prepare(fixture())
	if e != nil {
		t.Fatal(e)
	}
	ms := GenerateMoves(p, p.Level.InitialKey(), nil)
	if len(ms) != 4 {
		t.Fatalf("moves=%d, want 4", len(ms))
	}
	if IsSolved(p, p.Level.InitialKey()) {
		t.Fatal("fixture should be blocked")
	}
	next := ApplyMove(p, p.Level.InitialKey(), puzzle.Move{Vehicle: 1, From: 2, To: 0})
	if !IsSolved(p, next) {
		t.Fatal("expected solved state")
	}
}
func TestStateCoordinate(t *testing.T) {
	k := puzzle.EncodeState([]uint8{1, 6, 3})
	k = puzzle.WithPosition(k, 1, 2)
	if puzzle.Position(k, 0) != 1 || puzzle.Position(k, 1) != 2 || puzzle.Position(k, 2) != 3 {
		t.Fatal("packed state corrupted")
	}
}
