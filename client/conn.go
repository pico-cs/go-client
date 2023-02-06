package client

import (
	"io"
	"time"
)

const (
	reconnectRetry = 10
	reconnectWait  = 5 * time.Second // wait some time to reconnect
)

// Conn is a stream oriented connection to the pico board.
type Conn interface {
	Reconnect() error
	io.ReadWriteCloser
}
