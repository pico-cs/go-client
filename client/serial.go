package client

import (
	"errors"
	"log"
	"strings"

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

// Serial provides a serial connection to to the Raspberry Pi pico.
type Serial struct {
	portName string
	port     serial.Port
}

// NewSerial returns a new serial connection instance.
func NewSerial(portName string) (*Serial, error) {
	mode := &serial.Mode{
		BaudRate: baudRate,
	}
	port, err := serial.Open(portName, mode)
	if err != nil {
		log.Printf("error opening serial %s", err)
		return nil, err
	}
	return &Serial{portName: portName, port: port}, nil
}

// Read implements the io.Reader interface.
func (s *Serial) Read(p []byte) (n int, err error) {
	return s.port.Read(p)
}

// Write implements the io.Writer interface.
func (s *Serial) Write(p []byte) (n int, err error) {
	return s.port.Write(p)
}

// Close closes the serial port.
func (s *Serial) Close() error {
	return s.port.Close()
}
