package engine

import (
	"parking/internal/puzzle"
	"testing"
)

func TestGeneratedMovesAreReversible(t *testing.T) {
	p, err := Prepare(fixture())
	if err != nil {
		t.Fatal(err)
	}
	start := p.Level.InitialKey()
	for _, m := range GenerateMoves(p, start, nil) {
		next := ApplyMove(p, start, m)
		reverse := puzzle.Move{Vehicle: m.Vehicle, From: m.To, To: m.From}
		if _, err := ApplyMoveChecked(p, next, reverse); err != nil {
			t.Fatalf("move %+v is not reversible: %v", m, err)
		}
	}
}
