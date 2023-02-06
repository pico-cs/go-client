// Package client provides a pico-cs go command station client.
package client

import (
	"bufio"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/pico-cs/go-client/client/flash"
	"github.com/pico-cs/go-client/client/rbuf"
)

const (
	tagStart     = '+'
	tagSuccess   = '='
	tagNoSuccess = '?'
	tagMulti     = '-'
	tagEOR       = '.'
	tagPush      = '!'
)

const (
	charTrue   = 't'
	charFalse  = 'f'
	charToggle = "~"
)

func formatBool(b bool) byte {
	if b {
		return charTrue
	}
	return charFalse
}

const (
	cmdHelp                = "h"
	cmdBoard               = "b"
	cmdStore               = "s"
	cmdTemp                = "t"
	cmdCV                  = "cv"
	cmdMTE                 = "mte"
	cmdLocoDir             = "ld"
	cmdLocoSpeed128        = "ls"
	cmdLocoFct             = "lf"
	cmdLocoCVByte          = "lcvbyte"
	cmdLocoCVBit           = "lcvbit"
	cmdLocoCV29Bit5        = "lcv29bit5"
	cmdLocoLaddr           = "lladdr"
	cmdLocoCV1718          = "lcv1718"
	cmdAccFct              = "af"
	cmdAccTime             = "at"
	cmdAccStatus           = "as"
	cmdIOADC               = "ioadc"
	cmdIOVal               = "ioval"
	cmdIODir               = "iodir"
	cmdIOUp                = "ioup"
	cmdIODown              = "iodown"
	cmdRefreshBuffer       = "r"
	cmdRefreshBufferReset  = "rr"
	cmdRefreshBufferDelete = "rd"
	cmdFlash               = "f"
	cmdFlashFormat         = "ff"
	cmdReboot              = "reboot"
)

// error texts.
const (
	etInvCmd    = "invcmd"
	etInvPrm    = "invprm"
	etInvNumPrm = "invnumprm"
	etNoData    = "nodata"
	etNoChange  = "nochange"
	etInvGPIO   = "invgpio"
	etNotImpl   = "notimpl"
	etNotExec   = "notexec"
	etIOErr     = "ioerr"
)

// Command station error definitions.
var (
	ErrInvCmd    = errors.New("invalid command")
	ErrInvPrm    = errors.New("invalid parameter")
	ErrInvNumPrm = errors.New("invalid number of parameters")
	ErrNoData    = errors.New("no data")
	ErrNoChange  = errors.New("no change")
	ErrInvGPIO   = errors.New("invalid GPIO")
	ErrNotImpl   = errors.New("not implemented")
	ErrIO        = errors.New("io error")
	ErrNotExec   = errors.New("not executed")
	ErrUnknown   = errors.New("unknown error")
)

var errorMap = map[string]error{
	etInvCmd:    ErrInvCmd,
	etInvPrm:    ErrInvPrm,
	etInvNumPrm: ErrInvNumPrm,
	etNoData:    ErrNoData,
	etNoChange:  ErrNoChange,
	etInvGPIO:   ErrInvGPIO,
	etNotImpl:   ErrNotImpl,
	etIOErr:     ErrIO,
}

// CVIdx represents a command station CV index.
type CVIdx byte

// Main track CV index constants.
const (
	CVMT           CVIdx = iota // Main track configuration flags
	CVNumSyncBit                // Number of DCC synchronization bits
	CVNumRepeat                 // DCC command repetitions
	CVNumRepeatCV               // DCC command repetitions for main track CV programming
	CVNumRepeatAcc              // DCC command repetitions for accessory decoders
	CVBidiTS                    // BiDi (microseconds until power off after end bit)
	CVBidiTE                    // BiDi (microseconds to power on before start of 5th sync bit)
)

const (
	replyChSize = 1
	pushChSize  = 100
	timeout     = 5
)

// Client represents a command station client instance.
type Client struct {
	conn        Conn
	handler     func(msg Msg, err error)
	mu          sync.Mutex // mutex for call
	w           *bufio.Writer
	wg          *sync.WaitGroup
	replyCh     <-chan any
	lastReadErr error
}

// New returns a new client instance.
func New(conn Conn, handler func(msg Msg, err error)) *Client {
	c := &Client{
		conn:    conn,
		handler: handler,
		w:       bufio.NewWriter(conn),
	}
	c.startup()
	return c
}

func (c *Client) startup() {
	c.wg = new(sync.WaitGroup)
	var pushCh <-chan string
	c.replyCh, pushCh = c.reader(c.wg)
	c.pusher(c.wg, pushCh, c.handler)
}

func (c *Client) shutdown() error {
	err := c.conn.Close()
	c.wg.Wait()
	return err
}

// Reconnect tries to reconnect the client.
func (c *Client) Reconnect() error {
	c.shutdown() // ignore error
	if err := c.conn.Reconnect(); err != nil {
		return err
	}
	c.startup()
	return nil
}

// Close closes the client connection.
func (c *Client) Close() error { return c.shutdown() }

type replyKind int

const (
	rkNone replyKind = iota
	rkSingle
	rkMulti
	rkEOR
	rkPush
	rkError
)

func (c *Client) parseReply(buf []byte) (replyKind, string) {
	for i, b := range buf {
		switch b {
		case tagSuccess:
			return rkSingle, string(buf[i+1:])
		case tagNoSuccess:
			return rkError, string(buf[i+1:])
		case tagMulti:
			return rkMulti, string(buf[i+1:])
		case tagEOR:
			return rkEOR, ""
		case tagPush:
			return rkPush, string(buf[i+1:])
		}
	}
	return rkNone, ""
}

func (c *Client) reader(wg *sync.WaitGroup) (<-chan any, <-chan string) {

	replyCh := make(chan any, replyChSize)
	pushCh := make(chan string, pushChSize)

	go func() {
		defer wg.Done()

		scanner := bufio.NewScanner(c.conn)

		//TODO check scanner.Error()

		multi := false
		var multiMsg []string

		for scanner.Scan() {
			//log.Printf("message: %s", scanner.Text())

			rk, msg := c.parseReply(scanner.Bytes())
			switch rk {
			default: // ignore
			case rkError:
				if err, ok := errorMap[msg]; ok {
					replyCh <- err
				} else {
					replyCh <- ErrUnknown
				}
			case rkSingle:
				replyCh <- msg
			case rkPush:
				pushCh <- msg
			case rkMulti:
				if !multi {
					multiMsg = []string{}
					multi = true
				}
				multiMsg = append(multiMsg, msg)
			case rkEOR:
				replyCh <- multiMsg
				multi = false
			}
		}
		//if err := scanner.Err(); err != nil {
		//	fmt.Fprintln(os.Stderr, "reading standard input:", err)
		//}

		close(replyCh)
		close(pushCh)
	}()

	wg.Add(1)
	return replyCh, pushCh
}

func (c *Client) pusher(wg *sync.WaitGroup, pushCh <-chan string, handler func(Msg, error)) {
	go func() {
		defer wg.Done()

		for s := range pushCh {
			if handler != nil {
				handler(parseMsg(s))
			}
		}
	}()
	wg.Add(1)
}

func (c *Client) write(cmd string, args []any) error {
	c.w.WriteByte(tagStart)
	c.w.WriteString(cmd)
	for _, arg := range args {
		c.w.WriteByte(' ') // argument separator

		rv := reflect.ValueOf(arg)
		switch rv.Kind() {
		case reflect.Bool:
			c.w.WriteByte(formatBool(rv.Bool()))
		case reflect.Uint8, reflect.Uint:
			c.w.WriteString(strconv.FormatUint(rv.Uint(), 10))
		case reflect.String:
			c.w.WriteString(rv.String())
		default:
			panic(fmt.Sprintf("invalid argument %[1]v type %[1]T", arg)) // should never happen
		}
	}
	c.w.WriteByte('\r')
	if err := c.w.Flush(); err != nil {
		return err
	}
	return nil
}

func (c *Client) read() (any, error) {
	select {
	case reply, ok := <-c.replyCh:
		if !ok {
			return nil, c.lastReadErr
		}
		if err, ok := reply.(error); ok { // is error reply?
			return nil, err
		}
		return reply, nil

	case <-time.After(timeout * time.Second):
		return nil, fmt.Errorf("read timeout after %d seconds", timeout)
	}
}

func (c *Client) call(cmd string, args ...any) error {
	// guarantee:
	// - writing is not 'interleaved' and
	// - reply order
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.write(cmd, args)
}

func (c *Client) callReply(cmd string, args ...any) (any, error) {
	// guarantee:
	// - writing is not 'interleaved' and
	// - reply order
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.write(cmd, args); err != nil {
		return nil, err
	}
	return c.read()
}

func (c *Client) singleReply(cmd string, args ...any) (string, error) {
	res, err := c.callReply(cmd, args...)
	if err != nil {
		return "", err
	}
	v, ok := res.(string)
	if !ok {
		return "", fmt.Errorf("invalid reply message type %T", res)
	}
	return string(v), nil
}

func (c *Client) multiReply(cmd string, args ...any) ([]string, error) {
	res, err := c.callReply(cmd, args...)
	if err != nil {
		return nil, err
	}
	v, ok := res.([]string)
	if !ok {
		return nil, fmt.Errorf("invalid reply message type %T", res)
	}
	return []string(v), nil
}

// Help returns the help texts of the command station.
func (c *Client) Help() ([]string, error) {
	v, err := c.multiReply(cmdHelp)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// Board returns board information like controller type and unique id.
func (c *Client) Board() (*Board, error) {
	v, err := c.singleReply(cmdBoard)
	if err != nil {
		return nil, err
	}
	return parseBoard(v)
}

// Store stores the command station CVs on flash.
func (c *Client) Store() (bool, error) {
	v, err := c.singleReply(cmdStore)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// Temp returns the temperature of the command station.
func (c *Client) Temp() (float64, error) {
	v, err := c.singleReply(cmdTemp)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(v, 64)
}

// CV returns the value of a command station CV.
func (c *Client) CV(idx CVIdx) (byte, error) {
	v, err := c.singleReply(cmdCV, idx)
	if err != nil {
		return 0, err
	}
	return parseByte(v)
}

// SetCV sets the value of a command station CV.
func (c *Client) SetCV(idx CVIdx, val byte) (byte, error) {
	v, err := c.singleReply(cmdCV, idx, val)
	if err != nil {
		return 0, err
	}
	return parseByte(v)
}

// MTE returns true if the main track DCC sigal generation is enabled, false otherwise.
func (c *Client) MTE() (bool, error) {
	v, err := c.singleReply(cmdMTE)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetMTE sets main track DCC sigal generation whether to enabled or disabled.
func (c *Client) SetMTE(enabled bool) (bool, error) {
	v, err := c.singleReply(cmdMTE, enabled)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// LocoDir returns the direction of a loco.
// true : forward direction
// false: backward direction
func (c *Client) LocoDir(addr uint) (bool, error) {
	v, err := c.singleReply(cmdLocoDir, addr)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetLocoDir sets the direction of a loco.
// true : forward direction
// false: backward direction
func (c *Client) SetLocoDir(addr uint, dir bool) (bool, error) {
	v, err := c.singleReply(cmdLocoDir, addr, dir)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// ToggleLocoDir toggles the direction of a loco.
func (c *Client) ToggleLocoDir(addr uint) (bool, error) {
	v, err := c.singleReply(cmdLocoDir, addr, charToggle)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// LocoSpeed128 returns the speed of a loco.
// 0    : stop
// 1    : emergency stop
// 2-127: 126 speed steps
func (c *Client) LocoSpeed128(addr uint) (uint, error) {
	v, err := c.singleReply(cmdLocoSpeed128, addr)
	if err != nil {
		return 0, err
	}
	return parseUint(v)
}

// SetLocoSpeed128 sets the speed of a loco.
// 0    : stop
// 1    : emergency stop
// 2-127: 126 speed steps
func (c *Client) SetLocoSpeed128(addr, speed uint) (uint, error) {
	v, err := c.singleReply(cmdLocoSpeed128, addr, speed)
	if err != nil {
		return 0, err
	}
	return parseUint(v)
}

// LocoFct returns a function value of a loco.
func (c *Client) LocoFct(addr, no uint) (bool, error) {
	v, err := c.singleReply(cmdLocoFct, addr, no)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetLocoFct sets a function value of a loco.
func (c *Client) SetLocoFct(addr, no uint, fct bool) (bool, error) {
	v, err := c.singleReply(cmdLocoFct, addr, no, fct)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// ToggleLocoFct toggles a function value of a loco.
func (c *Client) ToggleLocoFct(addr, no uint) (bool, error) {
	v, err := c.singleReply(cmdLocoFct, addr, no, charToggle)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetLocoCVByte sets the indexed CV byte value of a loco.
func (c *Client) SetLocoCVByte(addr, idx uint, val byte) (byte, error) {
	v, err := c.singleReply(cmdLocoCVByte, addr, idx, val)
	if err != nil {
		return 0, err
	}
	return parseByte(v)
}

// SetLocoCVBit sets the indexed CV bit value of a loco.
func (c *Client) SetLocoCVBit(addr, idx uint, bit byte, val bool) (bool, error) {
	v, err := c.singleReply(cmdLocoCVBit, addr, idx, bit, val)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetLocoCV29Bit5 sets the CV 29 bit 5 value of a loco.
func (c *Client) SetLocoCV29Bit5(addr uint, bit bool) (bool, error) {
	v, err := c.singleReply(cmdLocoCV29Bit5, addr, bit)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetLocoLaddr sets the long address of a loco.
func (c *Client) SetLocoLaddr(addr, laddr uint) (uint, error) {
	v, err := c.singleReply(cmdLocoLaddr, addr, laddr)
	if err != nil {
		return 0, err
	}
	return parseUint(v)
}

// LocoCV1718 returns (calculates) the CV17 and CV18 values (long address) from a loco address.
func (c *Client) LocoCV1718(addr uint) (byte, byte, error) {
	v, err := c.singleReply(cmdLocoCV1718, addr)
	if err != nil {
		return 0, 0, err
	}
	return parseByteTuple(v)
}

// IOADC returns the 'raw' value of the ADC input.
func (c *Client) IOADC(input uint) (float64, error) {
	v, err := c.singleReply(cmdIOADC, input)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(v, 64)
}

// SetAccFct sets the function value of an accessory decoder on output out.
func (c *Client) SetAccFct(addr uint, out byte, fct bool) (bool, error) {
	v, err := c.singleReply(cmdAccFct, addr, out, fct)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetAccTime sets the activation time of an accessory decoder on output out.
func (c *Client) SetAccTime(addr uint, out, time byte) (bool, error) {
	v, err := c.singleReply(cmdAccTime, addr, out, time)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetAccStatus sets the status byte of an extended accessory decoder.
func (c *Client) SetAccStatus(addr uint, status byte) (bool, error) {
	v, err := c.singleReply(cmdAccStatus, addr, status)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// IOVal returns the boolean value of the GPIO.
func (c *Client) IOVal(cmd, gpio uint) (bool, error) {
	v, err := c.singleReply(cmdIOVal, cmd, gpio)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetIOVal sets the boolean value of the GPIO.
func (c *Client) SetIOVal(cmd, gpio uint, value bool) (bool, error) {
	v, err := c.singleReply(cmdIOVal, cmd, gpio, value)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// ToggleIOVal toggles the value of the GPIO.
func (c *Client) ToggleIOVal(cmd, gpio uint) (bool, error) {
	v, err := c.singleReply(cmdIOVal, cmd, gpio, charToggle)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// IODir returns the direction of the GPIO.
// false: in
// true:  out
func (c *Client) IODir(cmd, gpio uint) (bool, error) {
	v, err := c.singleReply(cmdIODir, cmd, gpio)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetIODir sets the direction of the GPIO.
// false: in
// true:  out
func (c *Client) SetIODir(cmd, gpio uint, value bool) (bool, error) {
	v, err := c.singleReply(cmdIODir, cmd, gpio, value)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// ToggleIODir toggles the direction of the GPIO.
func (c *Client) ToggleIODir(cmd, gpio uint) (bool, error) {
	v, err := c.singleReply(cmdIODir, cmd, gpio, charToggle)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// IOUp returns the pull-up status of the GPIO.
func (c *Client) IOUp(cmd, gpio uint) (bool, error) {
	v, err := c.singleReply(cmdIOUp, cmd, gpio)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetIOUp sets the pull-up status of the GPIO.
func (c *Client) SetIOUp(cmd, gpio uint, value bool) (bool, error) {
	v, err := c.singleReply(cmdIOUp, cmd, gpio, value)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// ToggleIOUp toggles the pull-up status of the GPIO.
func (c *Client) ToggleIOUp(cmd, gpio uint) (bool, error) {
	v, err := c.singleReply(cmdIOUp, cmd, gpio, charToggle)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// IODown returns the pull-down status of the GPIO.
func (c *Client) IODown(cmd, gpio uint) (bool, error) {
	v, err := c.singleReply(cmdIODown, cmd, gpio)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetIODown sets the pull-down status of the GPIO.
func (c *Client) SetIODown(cmd, gpio uint, value bool) (bool, error) {
	v, err := c.singleReply(cmdIODown, cmd, gpio, value)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// ToggleIODown toggles the pull-down status of the GPIO.
func (c *Client) ToggleIODown(cmd, gpio uint) (bool, error) {
	v, err := c.singleReply(cmdIODown, cmd, gpio, charToggle)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// RefreshBuffer returns the command station refresh buffer (debugging).
func (c *Client) RefreshBuffer() (*rbuf.Buffer, error) {
	v, err := c.multiReply(cmdRefreshBuffer)
	if err != nil {
		return nil, err
	}
	return rbuf.Parse(v)
}

// RefreshBufferReset resets the refresh buffer (debugging).
func (c *Client) RefreshBufferReset() (bool, error) {
	v, err := c.singleReply(cmdRefreshBufferReset)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// RefreshBufferDelete deletes address addr from refresh buffer (debugging).
func (c *Client) RefreshBufferDelete(addr uint) (uint, error) {
	v, err := c.singleReply(cmdRefreshBufferDelete, addr)
	if err != nil {
		return 0, err
	}
	return parseUint(v)
}

// Flash returns the command station flash data (debugging).
func (c *Client) Flash() (*flash.Flash, error) {
	v, err := c.multiReply(cmdFlash)
	if err != nil {
		return nil, err
	}
	return flash.Parse(v)
}

// FlashFormat formats the command station flash (debugging).
func (c *Client) FlashFormat() (bool, error) {
	v, err := c.singleReply(cmdFlashFormat)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// Reboot reboots the command station (debugging).
func (c *Client) Reboot() error {
	return c.call(cmdReboot)
}
