package client

import (
	"io"
	"time"
)

const (
	reconnectRetry = 10
	reconnectWait  = 500 * time.Millisecond
)

// Conn is a stream oriented connection to the pico board.
type Conn interface {
	Connect() error
	io.ReadWriteCloser
}
