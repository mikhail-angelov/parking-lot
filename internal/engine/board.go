package engine

import (
	"fmt"
	"parking/internal/puzzle"
)

// BoardMask is a bit set of occupied cells on the fixed 6x6 board.
type BoardMask uint64

// PreparedLevel stores a validated level and its precomputed vehicle masks.
type PreparedLevel struct {
	Level puzzle.Level
	masks [][]BoardMask
}

// BlockingVehicle identifies a vehicle and the cells where it blocks a move.
type BlockingVehicle struct {
	Vehicle int
	Cells   BoardMask
}

// Prepare validates a level and precomputes every vehicle placement.
func Prepare(l puzzle.Level) (*PreparedLevel, error) {
	if err := puzzle.ValidateLevelAllowSolved(l); err != nil {
		return nil, fmt.Errorf("validate level: %w", err)
	}
	p := &PreparedLevel{Level: l, masks: make([][]BoardMask, len(l.Vehicles))}
	for i, v := range l.Vehicles {
		count := int(puzzle.BoardSize - v.Length + 1)
		p.masks[i] = make([]BoardMask, count)
		for pos := 0; pos < count; pos++ {
			var m BoardMask
			for n := 0; n < int(v.Length); n++ {
				x, y := pos+n, int(v.Fixed)
				if v.Orientation == puzzle.Vertical {
					x, y = int(v.Fixed), pos+n
				}
				m |= BoardMask(1) << uint(y*6+x)
			}
			p.masks[i][pos] = m
		}
	}
	return p, nil
}

// Occupancy returns all cells occupied in a state.
func Occupancy(p *PreparedLevel, k puzzle.StateKey) BoardMask {
	var m BoardMask
	for i := range p.Level.Vehicles {
		m |= p.masks[i][puzzle.Position(k, i)]
	}
	return m
}

// GenerateMoves appends all legal moves to a reusable destination slice.
func GenerateMoves(p *PreparedLevel, k puzzle.StateKey, dst []puzzle.Move) []puzzle.Move {
	dst = dst[:0]
	occ := Occupancy(p, k)
	for i := range p.Level.Vehicles {
		pos := puzzle.Position(k, i)
		own := p.masks[i][pos]
		other := occ &^ own
		for q := pos; q > 0; {
			q--
			if p.masks[i][q]&other != 0 {
				break
			}
			dst = append(dst, puzzle.Move{Vehicle: p.Level.Vehicles[i].ID, From: pos, To: q})
		}
		for q := pos + 1; int(q) < len(p.masks[i]); q++ {
			if p.masks[i][q]&other != 0 {
				break
			}
			dst = append(dst, puzzle.Move{Vehicle: p.Level.Vehicles[i].ID, From: pos, To: q})
		}
	}
	return dst
}

// PositionCount returns the number of placements available to a vehicle.
func PositionCount(p *PreparedLevel, vehicle int) int {
	if vehicle < 0 || vehicle >= len(p.masks) {
		return 0
	}
	return len(p.masks[vehicle])
}

// PlacementClears reports whether a placement avoids the supplied cells.
func PlacementClears(p *PreparedLevel, vehicle int, position uint8, cells BoardMask) bool {
	return vehicle >= 0 && vehicle < len(p.masks) && int(position) < len(p.masks[vehicle]) && p.masks[vehicle][position]&cells == 0
}

// BlockingVehiclesForMove returns every vehicle intersecting a move's path.
func BlockingVehiclesForMove(p *PreparedLevel, k puzzle.StateKey, vehicle int, to uint8) []BlockingVehicle {
	if vehicle < 0 || vehicle >= len(p.masks) || int(to) >= len(p.masks[vehicle]) {
		return nil
	}
	from := puzzle.Position(k, vehicle)
	if from == to {
		return nil
	}
	step := 1
	if to < from {
		step = -1
	}
	path := BoardMask(0)
	for position := int(from) + step; ; position += step {
		path |= p.masks[vehicle][position]
		if position == int(to) {
			break
		}
	}
	path &^= p.masks[vehicle][from]

	blockers := make([]BlockingVehicle, 0)
	for other := range p.Level.Vehicles {
		if other == vehicle {
			continue
		}
		cells := p.masks[other][puzzle.Position(k, other)] & path
		if cells != 0 {
			blockers = append(blockers, BlockingVehicle{Vehicle: other, Cells: cells})
		}
	}
	return blockers
}

// TargetBlockingVehicles returns the vehicles between the target and the exit.
func TargetBlockingVehicles(p *PreparedLevel, k puzzle.StateKey) []BlockingVehicle {
	target := int(p.Level.Target)
	exitPosition := puzzle.BoardSize - p.Level.Vehicles[target].Length
	return BlockingVehiclesForMove(p, k, target, exitPosition)
}

// ApplyMove applies a known legal move to a state.
func ApplyMove(_ *PreparedLevel, k puzzle.StateKey, m puzzle.Move) puzzle.StateKey {
	return puzzle.WithPosition(k, int(m.Vehicle), m.To)
}

// ApplyMoveChecked validates a move before applying it.
func ApplyMoveChecked(p *PreparedLevel, k puzzle.StateKey, m puzzle.Move) (puzzle.StateKey, error) {
	for _, legal := range GenerateMoves(p, k, nil) {
		if legal == m {
			return ApplyMove(p, k, m), nil
		}
	}
	return k, fmt.Errorf("illegal move: vehicle %d from %d to %d", m.Vehicle, m.From, m.To)
}

// IsSolved reports whether the target has a clear path to the exit.
func IsSolved(p *PreparedLevel, k puzzle.StateKey) bool {
	t := p.Level.Vehicles[p.Level.Target]
	pos := puzzle.Position(k, int(p.Level.Target))
	corridor := BoardMask(0)
	for x := int(pos + t.Length); x < 6; x++ {
		corridor |= BoardMask(1) << uint(int(t.Fixed)*6+x)
	}
	return Occupancy(p, k)&corridor == 0
}
