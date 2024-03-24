package client

import (
	"fmt"
	"runtime"
	"strings"

	"go.bug.st/serial"
)

const baudRate = 115200 // default baud rate of the Raspberry Pi pico.

// Serial default port errors.
var (
	ErrSerialDefaultPortPathMissing = fmt.Errorf("missing default serial port path for %s", runtime.GOOS)
	ErrSerialDefaultPortNotFound    = fmt.Errorf("default port could not be detected for %s", defaultSerialPortPath)
)

func defaultPortsList() ([]string, error) {
	if defaultSerialPortPath == "" {
		return nil, ErrSerialDefaultPortPathMissing
	}
	portNames, err := serial.GetPortsList()
	if err != nil {
		return nil, err
	}
	var result []string
	for _, name := range portNames {
		if strings.HasPrefix(name, defaultSerialPortPath) {
			result = append(result, name)
		}
	}
	return result, nil
}

// SerialDefaultPortName returns the default serial port name if a detection is possible and an error otherwise.
func SerialDefaultPortName() (string, error) {
	portNames, err := defaultPortsList()
	if err != nil {
		return "", err
	}
	switch len(portNames) {
	case 0:
		return "", ErrSerialDefaultPortNotFound
	case 1:
		return portNames[0], nil
	default: // more than one.
		return "", fmt.Errorf("default serial port not unique %v", portNames)

	}
}

// Serial provides a serial connection to to the Raspberry Pi Pico.
type Serial struct {
	portName string
	port     serial.Port
	closed   bool
}

// NewSerial returns a new serial connection instance.
func NewSerial(portName string) (*Serial, error) {
	s := &Serial{portName: portName}
	if err := s.Connect(); err != nil {
		return nil, err
	}
	return s, nil
}

// Connect connect the serial port.s
func (s *Serial) Connect() error {
	mode := &serial.Mode{
		BaudRate: baudRate,
	}
	var err error
	s.port, err = serial.Open(s.portName, mode)
	if err != nil {
		return fmt.Errorf("error opening serial device: %s - %w", s.portName, err)
	}
	s.port.ResetInputBuffer()  //nolint: errcheck
	s.port.ResetOutputBuffer() //nolint: errcheck
	s.closed = false
	return nil
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
