package puzzle

import "fmt"

func ValidateLevel(l Level) error            { return validateLevel(l, false) }
func ValidateLevelAllowSolved(l Level) error { return validateLevel(l, true) }
func validateLevel(l Level, allowSolved bool) error {
	if l.Width != BoardSize || l.Height != BoardSize {
		return fmt.Errorf("board must be 6x6")
	}
	if len(l.Vehicles) < 1 || len(l.Vehicles) > 18 {
		return fmt.Errorf("vehicle count must be 1..18")
	}
	if int(l.Target) >= len(l.Vehicles) {
		return fmt.Errorf("target index out of range")
	}
	if len(l.Initial.Positions) != len(l.Vehicles) {
		return fmt.Errorf("position count does not match vehicles")
	}
	mask := uint64(0)
	for i, v := range l.Vehicles {
		if v.ID != uint8(i) {
			return fmt.Errorf("vehicle IDs must be dense and ordered")
		}
		if v.Orientation != Horizontal && v.Orientation != Vertical {
			return fmt.Errorf("vehicle %d has invalid orientation", i)
		}
		if v.Length != 2 && v.Length != 3 {
			return fmt.Errorf("vehicle %d has invalid length", i)
		}
		if v.Fixed >= BoardSize {
			return fmt.Errorf("vehicle %d fixed coordinate out of range", i)
		}
		limit := BoardSize - v.Length
		if l.Initial.Positions[i] > limit {
			return fmt.Errorf("vehicle %d position out of range", i)
		}
		for n := uint8(0); n < v.Length; n++ {
			x, y := l.Initial.Positions[i]+n, v.Fixed
			if v.Orientation == Vertical {
				x, y = v.Fixed, l.Initial.Positions[i]+n
			}
			bit := uint64(1) << (y*6 + x)
			if mask&bit != 0 {
				return fmt.Errorf("vehicles overlap at %d,%d", x, y)
			}
			mask |= bit
		}
	}
	if l.Vehicles[l.Target].Orientation != Horizontal {
		return fmt.Errorf("target must be horizontal")
	}
	if !allowSolved && targetClear(l, l.Initial.Positions) {
		return fmt.Errorf("level is already solved")
	}
	return nil
}
func targetClear(l Level, ps []uint8) bool {
	t := l.Vehicles[l.Target]
	end := ps[l.Target] + t.Length
	for x := end; x < 6; x++ {
		occupied := false
		for i, v := range l.Vehicles {
			if uint8(i) == l.Target {
				continue
			}
			p := ps[i]
			for n := uint8(0); n < v.Length; n++ {
				xx, yy := p+n, v.Fixed
				if v.Orientation == Vertical {
					xx, yy = v.Fixed, p+n
				}
				if xx == x && yy == t.Fixed {
					occupied = true
				}
			}
		}
		if occupied {
			return false
		}
	}
	return true
}
