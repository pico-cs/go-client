package client

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Message kind.
const (
	MkUnknown = iota
	MkWifi
	MkTCP
	MkIOIE
)

// Message class.
const (
	mcUnknown = ""
	mcWifi    = "wifi:"
	mcTCP     = "tcp:"
	mcIOIE    = "ioie:"
)

var msgKindMap = map[string]byte{
	mcUnknown: MkUnknown,
	mcWifi:    MkWifi,
	mcTCP:     MkTCP,
	mcIOIE:    MkIOIE,
}

// A Msg represents a push message.
type Msg interface {
	fmt.Stringer
	Kind() int
}

// Kind implements the push message interface.
func (m *WifiMsg) Kind() int { return MkWifi }

// Kind implements the push message interface.
func (m *TCPMsg) Kind() int { return MkTCP }

// Kind implements the push message interface.
func (m *IOIEMsg) Kind() int { return MkIOIE }

func (m *WifiMsg) String() string { return fmt.Sprintf("%s %s", mcWifi, m.Text) }
func (m *TCPMsg) String() string  { return fmt.Sprintf("%s %s", mcTCP, m.Text) }
func (m *IOIEMsg) String() string { return fmt.Sprintf("%s gpio %d state %t", mcIOIE, m.GPIO, m.State) }

// WifiMsg represents a Wifi info message.
type WifiMsg struct {
	Text string
}

func parseWifiMsg(parts []string) (*WifiMsg, error) {
	if len(parts) != 1 {
		return nil, fmt.Errorf("invalid %s message %v", mcWifi, parts)
	}
	return &WifiMsg{Text: parts[0]}, nil
}

// TCPMsg represents a tcp info message.
type TCPMsg struct {
	Text string
}

func parseTCPMsg(parts []string) (*TCPMsg, error) {
	if len(parts) != 1 {
		return nil, fmt.Errorf("invalid %s info message %v", mcTCP, parts)
	}
	return &TCPMsg{Text: parts[0]}, nil
}

// IOIEMsg represents a GPIO input event message.
type IOIEMsg struct {
	GPIO  uint
	State bool
}

func parseIOIEMsg(parts []string) (*IOIEMsg, error) {
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid %s message %v", mcIOIE, parts)
	}
	gpio, err := parseUint(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid %s message %v - %w", mcIOIE, parts, err)
	}
	state, err := strconv.ParseBool(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid %s message %v - %w", mcIOIE, parts, err)
	}
	return &IOIEMsg{GPIO: gpio, State: state}, nil
}

func parseMsg(s string) (Msg, error) {
	if len(s) == 0 {
		return nil, errors.New("empty message")
	}
	parts := strings.Split(s, " ")
	switch msgKindMap[parts[0]] {
	case MkWifi:
		return parseWifiMsg(parts[1:])
	case MkTCP:
		return parseTCPMsg(parts[1:])
	case MkIOIE:
		return parseIOIEMsg(parts[1:])
	default:
		return nil, fmt.Errorf("invalid message %s", s)
	}
}
