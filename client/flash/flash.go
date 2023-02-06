// Package flash contains flash types, methods and functions.
package flash

import (
	"fmt"
	"strconv"
	"strings"
)

// Flash represents a command station flash memory.
type Flash struct {
	ReadIdx, WriteIdx, PageNo uint
	Content                   []byte
}

func (f *Flash) String() string {
	b := strings.Builder{}
	b.WriteString(fmt.Sprintf("read idx %d write idx %d page no %d content:", f.ReadIdx, f.WriteIdx, f.PageNo))
	for i, v := range f.Content {
		if i%32 == 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("%02x ", v))
	}
	return b.String()
}

// Parse parses the flash memory send by a command station.
func Parse(lines []string) (*Flash, error) {
	if len(lines) < 1 {
		return nil, fmt.Errorf("parse flash error - invalid number of lines %d", len(lines))
	}

	values := strings.Split(lines[0], " ")
	if len(values) != 3 {
		return nil, fmt.Errorf("parse flash error - invalid number of values %d - expected %d", len(values), 3)
	}

	readIdx, err := strconv.ParseUint(values[0], 10, 0)
	if err != nil {
		return nil, err
	}
	writeIdx, err := strconv.ParseUint(values[1], 10, 0)
	if err != nil {
		return nil, err
	}
	pageNo, err := strconv.ParseUint(values[2], 10, 0)
	if err != nil {
		return nil, err
	}
	flash := &Flash{
		ReadIdx:  uint(readIdx),
		WriteIdx: uint(writeIdx),
		PageNo:   uint(pageNo),
		Content:  []byte{},
	}

	// content
	for i := 1; i < len(lines); i++ {
		values := strings.Split(strings.TrimSpace(lines[i]), " ")
		for _, value := range values {
			u64, err := strconv.ParseUint(value, 16, 8)
			if err != nil {
				return nil, err
			}
			flash.Content = append(flash.Content, byte(u64))
		}
	}
	return flash, nil
}
