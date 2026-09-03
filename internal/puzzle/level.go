package puzzle

import "fmt"

// BoardSize is the width and height of the square game board.
const BoardSize uint8 = 6

// Orientation is the axis along which a vehicle moves.
type Orientation uint8

// Supported vehicle orientations.
const (
	Horizontal Orientation = iota
	Vertical
)

// Vehicle describes a vehicle's fixed geometry.
type Vehicle struct {
	ID          uint8       `json:"id"`
	Orientation Orientation `json:"-"`
	Length      uint8       `json:"length"`
	Fixed       uint8       `json:"fixed"`
}

// State contains the variable position of every vehicle.
type State struct{ Positions []uint8 }

// StateKey is the compact representation used by search algorithms.
type StateKey uint64

// Level contains the board, vehicles, target, and initial state.
type Level struct {
	Width    uint8     `json:"width"`
	Height   uint8     `json:"height"`
	Target   uint8     `json:"target"`
	Vehicles []Vehicle `json:"vehicles"`
	Initial  State     `json:"-"`
}

func (o Orientation) String() string {
	if o == Vertical {
		return "vertical"
	}
	return "horizontal"
}

// ParseOrientation parses the JSON representation of an orientation.
func ParseOrientation(s string) (Orientation, error) {
	if s == "horizontal" {
		return Horizontal, nil
	}
	if s == "vertical" {
		return Vertical, nil
	}
	return 0, fmt.Errorf("invalid orientation %q", s)
}

// Span returns the number of cells occupied by a vehicle.
func (v Vehicle) Span() uint8 { return v.Length }

// EncodeState packs vehicle positions into a state key.
func EncodeState(ps []uint8) StateKey {
	var k StateKey
	for i, p := range ps {
		k |= StateKey(p&7) << (i * 3)
	}
	return k
}

// Position extracts one vehicle's position from a state key.
func Position(k StateKey, vehicle int) uint8 { return uint8((k >> (vehicle * 3)) & 7) }

// WithPosition returns a state key with one vehicle moved.
func WithPosition(k StateKey, vehicle int, pos uint8) StateKey {
	mask := StateKey(7) << (vehicle * 3)
	return (k &^ mask) | (StateKey(pos&7) << (vehicle * 3))
}

// InitialKey returns the compact initial state.
func (l Level) InitialKey() StateKey { return EncodeState(l.Initial.Positions) }
