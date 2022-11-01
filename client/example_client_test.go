package client_test

import (
	"log"

	"github.com/pico-cs/go-client/client"
)

// ExampleClient shows how to establish a pico-cs command station client.
func ExampleClient() {

	defaultPortName, err := client.SerialDefaultPortName()
	if err != nil {
		log.Fatal(err)
	}

	conn, err := client.NewSerial(defaultPortName)
	if err != nil {
		log.Fatal(err)
	}

	client := client.NewClient(conn, func(msg string) {
		// handle push messages
		log.Printf("push message: %s", msg)
	})
	defer client.Close()

	// read command station temperature.
	temp, err := client.Temp()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("teperature %f", temp)

	// output:
}
