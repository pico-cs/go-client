package client_test

import (
	"log"
	"time"

	"github.com/pico-cs/go-client/client"
)

// ExampleClient shows how to establish a pico-cs command station client.
func ExampleClient() {

	time.Sleep(2 * time.Second)

	defaultPortName, err := client.SerialDefaultPortName()
	if err != nil {
		log.Fatal(err)
	}

	conn, err := client.NewSerial(defaultPortName)
	if err != nil {
		log.Fatal(err)
	}

	client := client.New(conn, func(msg client.Msg, err error) {
		// handle push messages
		if err != nil {
			log.Printf("push message: %s", msg)
		} else {
			log.Printf("push message error: %s", err)
		}
	})
	defer client.Close()

	// read board information.
	board, err := client.Board()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s ID %s MAC %s", board.Type, board.ID, board.MAC)

	// read command station temperature.
	temp, err := client.Temp()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("temperature %f", temp)

	// output:
}
