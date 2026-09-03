package puzzle

import "fmt"

const BoardSize uint8 = 6

type Orientation uint8

const (
	Horizontal Orientation = iota
	Vertical
)

type Vehicle struct {
	ID          uint8       `json:"id"`
	Orientation Orientation `json:"-"`
	Length      uint8       `json:"length"`
	Fixed       uint8       `json:"fixed"`
}

type State struct{ Positions []uint8 }
type StateKey uint64

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
func ParseOrientation(s string) (Orientation, error) {
	if s == "horizontal" {
		return Horizontal, nil
	}
	if s == "vertical" {
		return Vertical, nil
	}
	return 0, fmt.Errorf("invalid orientation %q", s)
}
func (v Vehicle) Span() uint8 { return v.Length }
func EncodeState(ps []uint8) StateKey {
	var k StateKey
	for i, p := range ps {
		k |= StateKey(p&7) << (i * 3)
	}
	return k
}
func Position(k StateKey, vehicle int) uint8 { return uint8((k >> (vehicle * 3)) & 7) }
func WithPosition(k StateKey, vehicle int, pos uint8) StateKey {
	mask := StateKey(7) << (vehicle * 3)
	return (k &^ mask) | (StateKey(pos&7) << (vehicle * 3))
}
func (l Level) InitialKey() StateKey { return EncodeState(l.Initial.Positions) }
