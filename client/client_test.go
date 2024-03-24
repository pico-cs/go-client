package client_test

import (
	"os"
	"testing"

	"github.com/pico-cs/go-client/client"
	"github.com/pico-cs/go-client/client/rbuf"
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
	enabled, err := c.MTE()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("enabled %t", enabled)
	enabled, err = c.SetMTE(true)
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

	defaultSyncBits, err := c.CV(client.CVNumSyncBit)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("default DCC sync bits %d", defaultSyncBits)

	// test sync bits range minSyncBits <= sync bits <= maxSyncBits
	syncBits, err := c.SetCV(client.CVNumSyncBit, minSyncBits-1)
	if err != nil {
		t.Fatal(err)
	}
	if syncBits != minSyncBits {
		t.Errorf("invalid number of sync bits %d - expected %d", syncBits, minSyncBits)
	}
	syncBits, err = c.SetCV(client.CVNumSyncBit, maxSyncBits+1)
	if err != nil {
		t.Fatal(err)
	}
	if syncBits != maxSyncBits {
		t.Errorf("invalid number of sync bits %d - expected %d", syncBits, maxSyncBits)
	}

	// set back to default
	syncBits, err = c.SetCV(client.CVNumSyncBit, defaultSyncBits)
	if err != nil {
		t.Fatal(err)
	}
	if syncBits != defaultSyncBits {
		t.Errorf("invalid number of sync bits %d - expected %d", syncBits, defaultSyncBits)
	}
}

func testRefreshBuffer(c *client.Client, t *testing.T) {
	buf, err := c.RefreshBuffer()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("refresh buffer")
	t.Log(buf)
	for _, entry := range buf.Entries {
		t.Log(entry)
	}
}

func testRefreshBufferDelete(c *client.Client, t *testing.T) {
	// reset refresh buffer
	if _, err := c.RefreshBufferReset(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.SetLocoSpeed128(3, 12); err != nil { // add loco to buffer
		t.Fatal(err)
	}
	t.Logf("added loco: %d", 3)
	if _, err := c.SetLocoSpeed128(10, 33); err != nil { // add loco to buffer
		t.Fatal(err)
	}
	t.Logf("added loco: %d", 10)

	addr, err := c.RefreshBufferDelete(10)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("deleted loco: %d", addr)

	buf, err := c.RefreshBuffer()
	if err != nil {
		t.Fatal(err)
	}

	if len(buf.Entries) != 1 {
		t.Fatalf("invalid number of refresh buffer entries %d - expected 1", len(buf.Entries))
	}
	entry := buf.Entries[0]
	if buf.First != int(entry[rbuf.Idx]) || buf.Next != int(entry[rbuf.Idx]) {
		t.Fatalf("invalid refresh buffer header first %d next %d - expected %d", buf.First, buf.Next, entry[rbuf.Idx])
	}
	if entry[rbuf.Prev] != entry[rbuf.Idx] || entry[rbuf.Next] != entry[rbuf.Idx] {
		t.Fatalf("invalid refresh buffer entry prev %d next %d - expected %d", entry[rbuf.Prev], entry[rbuf.Next], entry[rbuf.Idx])
	}
}

func testFlash(c *client.Client, t *testing.T) {
	flash, err := c.Flash()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("flash")
	t.Log(flash)
}

func testReboot(c *client.Client, t *testing.T) {
	if err := c.Reboot(); err != nil {
		t.Fatal(err)
	}
	if err := c.Reconnect(); err != nil {
		if c.IsSerialConn() { // after reboot connection port might be different.
			t.Log(err)
			return
		}
		t.Fatal(err)
	}
	testBoard(c, t)
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
		{"RefreshBuffer", testRefreshBuffer},
		{"RefreshBufferDelete", testRefreshBufferDelete},
		{"Flash", testFlash},
		{"Reboot", testReboot},
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
