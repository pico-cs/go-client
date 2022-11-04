package client

import (
	"encoding/binary"
	"fmt"
	"strconv"
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
	numIDParts    = 8
	numBoardValue = numIDParts + 1
)

// BoardID represents the unique id of a borad.
type BoardID uint64

func (id BoardID) String() string {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(id))
	return fmt.Sprintf("% x", b)
}

// Board hold information of the Pico board.
type Board struct {
	Type BoardType
	ID   BoardID
}

func parseBoard(s string) (*Board, error) {
	values := strings.Split(s, " ")
	if len(values) != numBoardValue {
		return nil, fmt.Errorf("parse board error - invalid number of values %d - expected %d", len(values), numBoardValue)
	}
	board := &Board{}
	board.Type = btValues[values[0]]

	shift := 56
	for i := 0; i < numIDParts; i++ {
		u64, err := strconv.ParseUint(values[i+1], 16, 8)
		if err != nil {
			return nil, err
		}
		board.ID |= BoardID(u64 << shift)
		shift -= 8
	}

	return board, nil
}
