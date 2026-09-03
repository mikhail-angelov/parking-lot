package render

import (
	"parking/internal/engine"
	"parking/internal/puzzle"
	"strings"
)

// ASCII renders a level state as a text board.
func ASCII(p *engine.PreparedLevel, k puzzle.StateKey) string {
	grid := make([][]rune, 6)
	for y := range grid {
		grid[y] = make([]rune, 6)
		for x := range grid[y] {
			grid[y][x] = '.'
		}
	}
	for i, v := range p.Level.Vehicles {
		ch := rune('A' + i)
		if uint8(i) == p.Level.Target {
			ch = 'T'
		}
		pos := puzzle.Position(k, i)
		for n := uint8(0); n < v.Length; n++ {
			x, y := pos+n, v.Fixed
			if v.Orientation == puzzle.Vertical {
				x, y = v.Fixed, pos+n
			}
			grid[y][x] = ch
		}
	}
	var b strings.Builder
	b.WriteString("+------+\n")
	for y := 0; y < 6; y++ {
		b.WriteByte('|')
		for x := 0; x < 6; x++ {
			b.WriteRune(grid[y][x])
		}
		if y == int(p.Level.Vehicles[p.Level.Target].Fixed) {
			b.WriteString("> EXIT")
		}
		b.WriteByte('\n')
	}
	b.WriteString("+------+")
	return b.String()
}
