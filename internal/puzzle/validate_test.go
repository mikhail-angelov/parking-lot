package puzzle

import "testing"

func TestValidateLevelRejectsUnknownOrientation(t *testing.T) {
	level := Level{
		Width:  6,
		Height: 6,
		Target: 0,
		Vehicles: []Vehicle{
			{ID: 0, Orientation: Horizontal, Length: 2, Fixed: 2},
			{ID: 1, Orientation: Orientation(2), Length: 2, Fixed: 2},
		},
		Initial: State{Positions: []uint8{0, 2}},
	}

	if err := ValidateLevel(level); err == nil {
		t.Fatal("expected unknown orientation to be rejected")
	}
}
