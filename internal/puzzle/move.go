package puzzle

type Move struct {
	Vehicle uint8 `json:"vehicle"`
	From    uint8 `json:"from"`
	To      uint8 `json:"to"`
}

func (m Move) Distance() uint8 {
	if m.To > m.From {
		return m.To - m.From
	}
	return m.From - m.To
}
func (m Move) Direction() int8 {
	if m.To > m.From {
		return 1
	}
	return -1
}
