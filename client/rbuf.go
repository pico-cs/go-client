package client

import (
	"fmt"
	"strings"

	"golang.org/x/exp/slices"
)

// RBuf represents a command station refresh buffer.
type RBuf struct {
	First, Next int
	Entries     []*RBufEntry
}

func (buf *RBuf) String() string {
	return fmt.Sprintf("first %d next %d num entries %d", buf.First, buf.Next, len(buf.Entries))
}

// RBufEntry represents a command station refresh buffer entry.
type RBufEntry struct {
	Idx             int
	Addr            uint
	NumRefreshCycle byte
	RefreshCycle    byte
	DirSpeed        byte
	F0_4            byte
	F5_8            byte
	F9_12           byte //lint:ignore ST1003 complains about ALL_CAPS
	F5_12           byte //lint:ignore ST1003 complains about ALL_CAPS
	F13_20          byte //lint:ignore ST1003 complains about ALL_CAPS
	F21_28          byte //lint:ignore ST1003 complains about ALL_CAPS
	F29_36          byte //lint:ignore ST1003 complains about ALL_CAPS
	F37_44          byte //lint:ignore ST1003 complains about ALL_CAPS
	F45_52          byte //lint:ignore ST1003 complains about ALL_CAPS
	F53_60          byte //lint:ignore ST1003 complains about ALL_CAPS
	F61_68          byte //lint:ignore ST1003 complains about ALL_CAPS
	Prev            byte
	Next            byte
}

func (e *RBufEntry) String() string {
	ppDirSpeed := func(fct byte) string { return fmt.Sprintf("%1b-%03d", fct>>7, fct&0x7f) }
	ppF0_4 := func(fct byte) string { return fmt.Sprintf("%1b-%04b", fct>>4, fct&0x0f) }
	ppFct := func(fct byte) string { return fmt.Sprintf("%04b-%04b", fct>>4, fct&0x0f) }

	return fmt.Sprintf(
		"idx %3d addr %5d numRefreshCycle %3d RefreshCycle %3d dirSpeed %s f0_4 %s f5_8 %04b f9_12 %04b f5_12 %s f13_20 %s f21_28 %s f29_36 %s f37_44 %s f45_52 %s f53_60 %s f61_68 %s prev %3d next %3d",
		e.Idx,
		e.Addr,
		e.NumRefreshCycle,
		e.RefreshCycle,
		ppDirSpeed(e.DirSpeed),
		ppF0_4(e.F0_4),
		e.F5_8,
		e.F9_12,
		ppFct(e.F5_12),
		ppFct(e.F13_20),
		ppFct(e.F21_28),
		ppFct(e.F29_36),
		ppFct(e.F37_44),
		ppFct(e.F45_52),
		ppFct(e.F53_60),
		ppFct(e.F61_68),
		e.Prev,
		e.Next,
	)
}

const (
	numHeaderValue = 2
	numEntryValue  = 18
)

func parseRBuf(lines []string) (*RBuf, error) {
	if len(lines) < 1 {
		return nil, fmt.Errorf("parse refresh buffer error - invalid number of lines %d", len(lines))
	}

	values := strings.Split(lines[0], " ")
	if len(values) != numHeaderValue {
		return nil, fmt.Errorf("parse refresh buffer error - invalid number of header values %d -expected %d", len(values), numHeaderValue)
	}

	p := &parser{values: values}
	buf := &RBuf{
		First: p.parseInt(),
		Next:  p.parseInt(),
	}
	if p.err != nil {
		return nil, p
	}

	// entries
	for i := 1; i < len(lines); i++ {
		values := strings.Split(lines[i], " ")
		if len(values) != numEntryValue {
			return nil, fmt.Errorf("parse refresh buffer error - invalid number of entry values %d - expected %d", len(values), numEntryValue)
		}

		p.reset(values)
		buf.Entries = append(buf.Entries, &RBufEntry{
			Idx:             p.parseInt(),
			Addr:            p.parseUInt(),
			NumRefreshCycle: p.parseByte(),
			RefreshCycle:    p.parseByte(),
			DirSpeed:        p.parseByte(),
			F0_4:            p.parseByte(),
			F5_8:            p.parseByte(),
			F9_12:           p.parseByte(),
			F5_12:           p.parseByte(),
			F13_20:          p.parseByte(),
			F21_28:          p.parseByte(),
			F29_36:          p.parseByte(),
			F37_44:          p.parseByte(),
			F45_52:          p.parseByte(),
			F53_60:          p.parseByte(),
			F61_68:          p.parseByte(),
			Prev:            p.parseByte(),
			Next:            p.parseByte(),
		})
		if p.err != nil {
			return nil, p
		}
		slices.SortFunc(buf.Entries, func(a, b *RBufEntry) bool { return a.Idx < b.Idx })
	}
	return buf, nil
}
