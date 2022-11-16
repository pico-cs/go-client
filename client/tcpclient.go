package client

import "net"

// DefaultTCPPort is the default TCP Port used by Pico W.
const DefaultTCPPort = "4242"

// TCPClient provides a TCP/IP connection to to the Raspberry Pi Pico W.
type TCPClient struct {
	conn net.Conn
}

// NewTCPClient returns a new TCP/IP connection instance.
func NewTCPClient(host, port string) (*TCPClient, error) {
	if port == "" {
		port = DefaultTCPPort
	}

	addr := net.JoinHostPort(host, port)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &TCPClient{conn: conn}, nil
}

// Read implements the Conn interface.
func (c *TCPClient) Read(p []byte) (n int, err error) {
	return c.conn.Read(p)
}

// Write implements the Conn interface.
func (c *TCPClient) Write(p []byte) (n int, err error) {
	return c.conn.Write(p)
}

// Close implements the Conn interface.
func (c *TCPClient) Close() error {
	return c.conn.Close()
}
