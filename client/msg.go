package client

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Message kind.
const (
	MkInvalid = iota
	MkInfo
	MkIOIE
)

var msgKindMap = map[string]byte{
	"wifi:": MkInfo,
	"tcp:":  MkInfo,
	"ioie:": MkIOIE,
}

// A Msg represents a push message.
type Msg interface {
	fmt.Stringer
	Kind() int
}

// Kind implements the push message interface.
func (m *InfoMsg) Kind() int { return MkInfo }

// Kind implements the push message interface.
func (m *IOIEMsg) Kind() int { return MkIOIE }

func (m *InfoMsg) String() string { return m.Text }
func (m *IOIEMsg) String() string { return fmt.Sprintf("gpio %d state %t", m.GPIO, m.State) }

// InfoMsg represents an information push message.
type InfoMsg struct {
	Text string
}

func parseInfoMsg(parts []string) (*InfoMsg, error) {
	if len(parts) != 1 {
		return nil, fmt.Errorf("invalid info message %v", parts)
	}
	return &InfoMsg{Text: parts[0]}, nil
}

// IOIEMsg represents a GPIO input event message.
type IOIEMsg struct {
	GPIO  uint
	State bool
}

func parseIOIEMsg(parts []string) (*IOIEMsg, error) {
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid ioie message %v", parts)
	}
	gpio, err := parseUint(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid ioie message %v - %w", parts, err)
	}
	state, err := strconv.ParseBool(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid ioie message %v - %w", parts, err)
	}
	return &IOIEMsg{GPIO: gpio, State: state}, nil
}

func parseMsg(s string) (Msg, error) {
	if len(s) == 0 {
		return nil, errors.New("empty message")
	}
	parts := strings.Split(s, " ")
	switch msgKindMap[parts[0]] {
	case MkInfo:
		return parseInfoMsg(parts[1:])
	case MkIOIE:
		return parseIOIEMsg(parts[1:])
	default:
		return nil, fmt.Errorf("invalid message %s", s)
	}
}
