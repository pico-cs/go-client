package client

import (
	"net"
	"time"
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
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *TCPClient) connect() (err error) {
	c.conn, err = net.Dial("tcp", net.JoinHostPort(c.host, c.port))
	return err
}

// Reconnect tries to reconnect the TCP client.
func (c *TCPClient) Reconnect() (err error) {
	err = nil
	for i := 0; i < reconnectRetry; i++ {
		time.Sleep(reconnectWait)
		if err = c.connect(); err == nil {
			return nil
		}
	}
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
