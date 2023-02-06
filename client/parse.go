package client

import (
	"fmt"
	"strconv"
	"strings"
)

func parseUint(s string) (uint, error) {
	u64, err := strconv.ParseUint(s, 10, 0)
	if err != nil {
		return 0, err
	}
	return uint(u64), nil
}

func parseByte(s string) (byte, error) {
	u64, err := strconv.ParseUint(s, 10, 8)
	if err != nil {
		return 0, err
	}
	return byte(u64), nil
}

func parseByteTuple(s string) (byte, byte, error) {
	values := strings.Split(s, " ")
	if len(values) != 2 {
		return 0, 0, fmt.Errorf("parse byte tuple error - invalid number of values %d", len(values))
	}
	b1, err := parseByte(values[0])
	if err != nil {
		return 0, 0, err
	}
	b2, err := parseByte(values[1])
	if err != nil {
		return 0, 0, err
	}
	return b1, b2, nil
}
