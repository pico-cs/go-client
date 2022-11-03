package client_test

import (
	"testing"

	"github.com/pico-cs/go-client/client"
)

func testHelp(client *client.Client, t *testing.T) {
	lines, err := client.Help()
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range lines {
		t.Log(line)
	}
}

func testBoard(client *client.Client, t *testing.T) {
	board, err := client.Board()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Type: %s ID: %s", board.Type, board.ID)
}

func testTemp(client *client.Client, t *testing.T) {
	numTest := 10 // read numTest temperature values

	for i := 0; i < numTest; i++ {
		temp, err := client.Temp()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Temperature: %f", temp)
	}
}

const (
	minSyncBits = 17
	maxSyncBits = 32
)

func testDCCSyncBits(client *client.Client, t *testing.T) {
	defaultSyncBits, err := client.DCCSyncBits()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("default DCC sync bits %d", defaultSyncBits)

	// test sync bits range minSyncBits <= sync bits <= maxSyncBits
	syncBits, err := client.SetDCCSyncBits(minSyncBits - 1)
	if err != nil {
		t.Fatal(err)
	}
	if syncBits != minSyncBits {
		t.Errorf("invalid number of sync bits %d - expected %d", syncBits, minSyncBits)
	}
	syncBits, err = client.SetDCCSyncBits(maxSyncBits + 1)
	if err != nil {
		t.Fatal(err)
	}
	if syncBits != maxSyncBits {
		t.Errorf("invalid number of sync bits %d - expected %d", syncBits, maxSyncBits)
	}

	// set back to default
	syncBits, err = client.SetDCCSyncBits(defaultSyncBits)
	if err != nil {
		t.Fatal(err)
	}
	if syncBits != defaultSyncBits {
		t.Errorf("invalid number of sync bits %d - expected %d", syncBits, defaultSyncBits)
	}
}

func testEnabled(client *client.Client, t *testing.T) {
	enabled, err := client.Enabled()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("enabled %t", enabled)
	enabled, err = client.SetEnabled(true)
	if err != nil {
		t.Fatal(err)
	}
	if enabled != true {
		t.Errorf("invalid enabled value %t - expected %t", enabled, true)
	}
}

func testRBuf(client *client.Client, t *testing.T) {
	rbuf, err := client.RBuf()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("refresh buffer")
	t.Log(rbuf)
	for _, entry := range rbuf.Entries {
		t.Log(entry)
	}
}

func TestClient(t *testing.T) {
	tests := []struct {
		name string
		fct  func(client *client.Client, t *testing.T)
	}{
		{"Help", testHelp},
		{"Board", testBoard},
		{"Temp", testTemp},
		{"DCCSyncBits", testDCCSyncBits},
		{"Enabled", testEnabled},
		{"RefreshBuffer", testRBuf},
	}

	defaultPortName, err := client.SerialDefaultPortName()
	if err != nil {
		t.Fatal(err)
	}

	conn, err := client.NewSerial(defaultPortName)
	if err != nil {
		t.Fatal(err)
	}

	client := client.NewClient(conn, nil)
	defer client.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.fct(client, t)
		})
	}
}
