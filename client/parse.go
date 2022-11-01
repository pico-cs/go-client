package client

import (
	"fmt"
	"strconv"
	"strings"
)

func parseInt(s string) (int, error) {
	i64, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		return 0, err
	}
	return int(i64), nil
}

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

// parser parses string values.
type parser struct {
	values []string
	idx    int
	errIdx int
	err    error
}

func (p *parser) reset(values []string) {
	p.values = values
	p.idx = 0
	p.errIdx = 0
	p.err = nil
}

func (p *parser) Error() string {
	return fmt.Errorf("parser error index %d %w", p.errIdx, p.err).Error()
}

func (p *parser) parseInt() int {
	i, err := parseInt(p.values[p.idx])
	if err != nil {
		p.errIdx = p.idx
		p.err = err
	}
	p.idx++
	return i
}

func (p *parser) parseUInt() uint {
	u, err := parseUint(p.values[p.idx])
	if err != nil {
		p.errIdx = p.idx
		p.err = err
	}
	p.idx++
	return u
}

func (p *parser) parseByte() byte {
	b, err := parseByte(p.values[p.idx])
	if err != nil {
		p.errIdx = p.idx
		p.err = err
	}
	p.idx++
	return b
}
