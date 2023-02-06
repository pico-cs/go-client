// Package rbuf contains refresh buffer types, methods and functions.
package rbuf

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

// refresh buffer indices
const (
	Idx = iota
	MSB
	LSB
	MaxRefreshCmd
	RefreshCmd
	DirSpeed
	F0_4
	F5_8
	F9_12  //lint:ignore ST1003 complains about ALL_CAPS
	F5_12  //lint:ignore ST1003 complains about ALL_CAPS
	F13_20 //lint:ignore ST1003 complains about ALL_CAPS
	F21_28 //lint:ignore ST1003 complains about ALL_CAPS
	F29_36 //lint:ignore ST1003 complains about ALL_CAPS
	F37_44 //lint:ignore ST1003 complains about ALL_CAPS
	F45_52 //lint:ignore ST1003 complains about ALL_CAPS
	F53_60 //lint:ignore ST1003 complains about ALL_CAPS
	F61_68 //lint:ignore ST1003 complains about ALL_CAPS
	Prev
	Next
	NumBytes
)

// Entry represents a command station refresh buffer entry.
type Entry [NumBytes]byte

func (e *Entry) String() string {
	ppDirSpeed := func(fct byte) string { return fmt.Sprintf("%1b-%03d", fct>>7, fct&0x7f) }
	ppF0_4 := func(fct byte) string { return fmt.Sprintf("%1b-%04b", fct>>4, fct&0x0f) }
	ppFct := func(fct byte) string { return fmt.Sprintf("%04b-%04b", fct>>4, fct&0x0f) }

	return fmt.Sprintf(
		"idx %3d addr %5d maxRefreshCmd %3d RefreshCmd %3d dirSpeed %s f0_4 %s f5_8 %04b f9_12 %04b f5_12 %s f13_20 %s f21_28 %s f29_36 %s f37_44 %s f45_52 %s f53_60 %s f61_68 %s prev %3d next %3d",
		e[Idx],
		(uint16(e[MSB])<<8)|uint16(e[LSB]),
		e[MaxRefreshCmd],
		e[RefreshCmd],
		ppDirSpeed(e[DirSpeed]),
		ppF0_4(e[F0_4]),
		e[F5_8],
		e[F9_12],
		ppFct(e[F5_12]),
		ppFct(e[F13_20]),
		ppFct(e[F21_28]),
		ppFct(e[F29_36]),
		ppFct(e[F37_44]),
		ppFct(e[F45_52]),
		ppFct(e[F53_60]),
		ppFct(e[F61_68]),
		e[Prev],
		e[Next],
	)
}

// Buffer represents a command station refresh buffer.
type Buffer struct {
	First, Next int
	Entries     []Entry
}

func (buf *Buffer) String() string {
	return fmt.Sprintf("first %d next %d num entries %d", buf.First, buf.Next, len(buf.Entries))
}

// Parse parses the refresh buffer send by a command station.
func Parse(lines []string) (*Buffer, error) {
	if len(lines) < 1 {
		return nil, fmt.Errorf("parse refresh buffer error - invalid number of lines %d", len(lines))
	}

	values := strings.Split(lines[0], " ")
	if len(values) != 2 {
		return nil, fmt.Errorf("parse refresh buffer error - invalid number of values %d - expected %d", len(values), 2)
	}

	first, err := strconv.ParseInt(values[0], 10, 0)
	if err != nil {
		return nil, err
	}
	next, err := strconv.ParseInt(values[1], 10, 0)
	if err != nil {
		return nil, err
	}
	buf := &Buffer{
		First:   int(first),
		Next:    int(next),
		Entries: make([]Entry, len(lines)-1),
	}

	// entries
	for i := 1; i < len(lines); i++ {
		values := strings.Split(lines[i], " ")
		if len(values) != NumBytes {
			return nil, fmt.Errorf("parse refresh buffer error - invalid number of entry values %d - expected %d", len(values), NumBytes)
		}
		for j := 0; j < NumBytes; j++ {
			u64, err := strconv.ParseUint(values[j], 10, 8)
			if err != nil {
				return nil, err
			}
			buf.Entries[i-1][j] = byte(u64)
		}
		slices.SortFunc(buf.Entries, func(a, b Entry) bool { return a[Idx] < b[Idx] })
	}
	return buf, nil
}
