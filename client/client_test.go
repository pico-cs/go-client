package client_test

import (
	"log"
	"os"
	"testing"

	"github.com/pico-cs/go-client/client"
)

const (
	envHost = "PICO_W_HOST"
	envPort = "PICO_W_PORT"
)

func testHelp(c *client.Client, t *testing.T) {
	lines, err := c.Help()
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range lines {
		t.Log(line)
	}
}

func testBoard(c *client.Client, t *testing.T) {
	board, err := c.Board()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s ID %s MAC %s", board.Type, board.ID, board.MAC)
}

func testTemp(c *client.Client, t *testing.T) {
	numTest := 10 // read numTest temperature values

	for i := 0; i < numTest; i++ {
		temp, err := c.Temp()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Temperature %f", temp)
	}
}

func testMTEnabled(c *client.Client, t *testing.T) {
	enabled, err := c.MTEnabled()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("enabled %t", enabled)
	enabled, err = c.SetMTEnabled(true)
	if err != nil {
		t.Fatal(err)
	}
	if enabled != true {
		t.Errorf("invalid enabled value %t - expected %t", enabled, true)
	}
}

const (
	minSyncBits = 17
	maxSyncBits = 32
)

func testDCCSyncBits(c *client.Client, t *testing.T) {

	defaultSyncBits, err := c.MTCV(client.MTCVNumSyncBit)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("default DCC sync bits %d", defaultSyncBits)

	// test sync bits range minSyncBits <= sync bits <= maxSyncBits
	syncBits, err := c.SetMTCV(client.MTCVNumSyncBit, minSyncBits-1)
	if err != nil {
		t.Fatal(err)
	}
	if syncBits != minSyncBits {
		t.Errorf("invalid number of sync bits %d - expected %d", syncBits, minSyncBits)
	}
	syncBits, err = c.SetMTCV(client.MTCVNumSyncBit, maxSyncBits+1)
	if err != nil {
		t.Fatal(err)
	}
	if syncBits != maxSyncBits {
		t.Errorf("invalid number of sync bits %d - expected %d", syncBits, maxSyncBits)
	}

	// set back to default
	syncBits, err = c.SetMTCV(client.MTCVNumSyncBit, defaultSyncBits)
	if err != nil {
		t.Fatal(err)
	}
	if syncBits != defaultSyncBits {
		t.Errorf("invalid number of sync bits %d - expected %d", syncBits, defaultSyncBits)
	}
}

func testRBuf(c *client.Client, t *testing.T) {
	rbuf, err := c.RBuf()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("refresh buffer")
	t.Log(rbuf)
	for _, entry := range rbuf.Entries {
		t.Log(entry)
	}
}

func testRBufDel(c *client.Client, t *testing.T) {
	// reset refresh buffer
	c.RBufReset()

	if _, err := c.SetLocoSpeed128(3, 12); err != nil { // add loco to buffer
		log.Fatal(err)
	}
	t.Logf("added loco: %d", 3)
	if _, err := c.SetLocoSpeed128(10, 33); err != nil { // add loco to buffer
		log.Fatal(err)
	}
	t.Logf("added loco: %d", 10)

	addr, err := c.RBufDel(10)
	if err != nil {
		log.Fatal(err)
	}
	t.Logf("deleted loco: %d", addr)

	rbuf, err := c.RBuf()
	if err != nil {
		t.Fatal(err)
	}

	if len(rbuf.Entries) != 1 {
		log.Fatalf("invalid number of refresh buffer entries %d - expected 1", len(rbuf.Entries))
	}
	entry := rbuf.Entries[0]
	if rbuf.First != int(entry.Idx) || rbuf.Next != int(entry.Idx) {
		log.Fatalf("invalid refresh buffer header first %d next %d - expected %d", rbuf.First, rbuf.Next, entry.Idx)
	}
	if entry.Prev != entry.Idx || entry.Next != entry.Idx {
		log.Fatalf("invalid refresh buffer entry prev %d next %d - expected %d", entry.Prev, entry.Next, entry.Idx)
	}
}

func testRun(conn client.Conn, t *testing.T) {
	tests := []struct {
		name string
		fct  func(c *client.Client, t *testing.T)
	}{
		{"Help", testHelp},
		{"Board", testBoard},
		{"Temp", testTemp},
		{"DCCSyncBits", testDCCSyncBits},
		{"MTEnabled", testMTEnabled},
		{"RefreshBuffer", testRBuf},
		{"RefreshBufferDel", testRBufDel},
	}

	c := client.New(conn, nil)
	defer c.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.fct(c, t)
		})
	}
}

func testSerial(t *testing.T) {
	defaultPortName, err := client.SerialDefaultPortName()
	if err != nil {
		t.Fatal(err)
	}

	conn, err := client.NewSerial(defaultPortName)
	if err != nil {
		t.Fatal(err)
	}

	testRun(conn, t)
}

func testTCPClient(t *testing.T) {
	host, ok := os.LookupEnv(envHost)
	if !ok {
		t.Logf("host environment variable %s not found", envHost)
		return
	}
	port, ok := os.LookupEnv(envPort)
	if !ok {
		t.Logf("port environment variable %s not found - default %s used", envPort, client.DefaultTCPPort)
		port = client.DefaultTCPPort
	}
	conn, err := client.NewTCPClient(host, port)
	if err != nil {
		t.Fatal(err)
	}

	testRun(conn, t)
}

func TestClient(t *testing.T) {
	tests := []struct {
		name string
		fct  func(t *testing.T)
	}{
		{"Serial", testSerial},
		{"TCPClient", testTCPClient},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.fct(t)
		})
	}
}
