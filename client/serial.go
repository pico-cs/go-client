package client

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go.bug.st/serial"
)

const baudRate = 115200 // default baud rate of the Raspberry Pi pico.

// SerialDefaultPortName returns the default serial port name if a detection is possible and an error otherwise.
func SerialDefaultPortName() (string, error) {
	portNames, err := serial.GetPortsList()
	if err != nil {
		return "", err
	}

	for _, name := range portNames {
		if strings.HasPrefix(name, defaultSerialPortPath) {
			return name, nil
		}
	}
	return "", errors.New("default port could not be detected")
}

// Serial provides a serial connection to to the Raspberry Pi Pico.
type Serial struct {
	portName string
	port     serial.Port
	closed   bool
}

// NewSerial returns a new serial connection instance.
func NewSerial(portName string) (*Serial, error) {
	if portName == "" {
		var err error
		if portName, err = SerialDefaultPortName(); err != nil {
			return nil, err
		}
	}

	s := &Serial{portName: portName}
	if err := s.connect(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Serial) connect() error {
	mode := &serial.Mode{
		BaudRate: baudRate,
	}
	var err error
	s.port, err = serial.Open(s.portName, mode)
	if err != nil {
		return fmt.Errorf("error opening serial device: %s - %w", s.portName, err)
	}
	s.port.ResetInputBuffer()
	s.port.ResetOutputBuffer()
	s.closed = false
	return nil
}

// Reconnect tries to reconnect the serial connection.
func (s *Serial) Reconnect() (err error) {
	err = nil
	for i := 0; i < reconnectRetry; i++ {
		time.Sleep(reconnectWait)
		if err = s.connect(); err == nil {
			return nil
		}
	}
	return err
}

// Read implements the Conn interface.
func (s *Serial) Read(p []byte) (n int, err error) {
	return s.port.Read(p)
}

// Write implements the Conn interface.
func (s *Serial) Write(p []byte) (n int, err error) {
	return s.port.Write(p)
}

// Close implements the Conn interface.
func (s *Serial) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return s.port.Close()
}
