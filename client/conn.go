package client

// Conn is a stream orianted connection to the pico board.
type Conn interface {
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	Close() error
}
