package client

import (
	"fmt"
	"strings"
)

// Board types.
const (
	BtUnknown BoardType = iota
	BtPico
	BtPicoW
)

var btTexts = []string{"Unknown", "Raspberry Pi Pico", "Raspberry Pi Pico W"}

var btValues = map[string]BoardType{"pico": BtPico, "pico_w": BtPicoW}

// BoardType represents the type of a board.
type BoardType byte

func (t BoardType) String() string {
	if int(t) >= len(btTexts) {
		return btTexts[BtUnknown]
	}
	return btTexts[t]
}

const (
	numBoardMinValue = 2
	numBoardMaxValue = 3
)

// Board hold information of the Pico board.
type Board struct {
	Type BoardType
	ID   string
	MAC  string
}

func parseBoard(s string) (*Board, error) {
	values := strings.Split(s, " ")
	l := len(values)
	if l < numBoardMinValue || l > numBoardMaxValue {
		return nil, fmt.Errorf("parse board error - invalid number of values %d - expected %d-%d", l, numBoardMinValue, numBoardMaxValue)
	}
	board := &Board{}
	board.Type = btValues[values[0]]

	board.ID = values[1]
	if l > 2 {
		board.MAC = values[2]
	}
	return board, nil
}
