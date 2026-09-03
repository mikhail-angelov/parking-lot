package puzzle

// Move records one vehicle transition between board positions.
type Move struct {
	Vehicle uint8 `json:"vehicle"`
	From    uint8 `json:"from"`
	To      uint8 `json:"to"`
}

// Distance returns the number of cells traversed by a move.
func (m Move) Distance() uint8 {
	if m.To > m.From {
		return m.To - m.From
	}
	return m.From - m.To
}

// Direction returns 1 for forward moves and -1 for backward moves.
func (m Move) Direction() int8 {
	if m.To > m.From {
		return 1
	}
	return -1
}
