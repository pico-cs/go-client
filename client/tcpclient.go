package client

import (
	"net"
)

// DefaultTCPPort is the default TCP Port used by Pico W.
const DefaultTCPPort = "4242"

// TCPClient provides a TCP/IP connection to to the Raspberry Pi Pico W.
type TCPClient struct {
	host, port string
	conn       net.Conn
}

// NewTCPClient returns a new TCP/IP connection instance.
func NewTCPClient(host, port string) (*TCPClient, error) {
	if port == "" {
		port = DefaultTCPPort
	}

	c := &TCPClient{host: host, port: port}
	if err := c.Connect(); err != nil {
		return nil, err
	}
	return c, nil
}

// Connect connects to the tcp address.
func (c *TCPClient) Connect() (err error) {
	c.conn, err = net.Dial("tcp", net.JoinHostPort(c.host, c.port))
	return err
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
