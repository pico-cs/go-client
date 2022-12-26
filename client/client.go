// Package client provides a pico-cs go command station client.
package client

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"sync"
	"time"
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
	cmdHelp         = "h"
	cmdBoard        = "b"
	cmdTemp         = "ct"
	cmdMTEnabled    = "mte"
	cmdMTCV         = "mtcv"
	cmdLocoDir      = "ld"
	cmdLocoSpeed128 = "ls"
	cmdLocoFct      = "lf"
	cmdLocoCVByte   = "lcvbyte"
	cmdLocoCVBit    = "lcvbit"
	cmdLocoCV29Bit5 = "lcv29bit5"
	cmdLocoLaddr    = "lladdr"
	cmdLocoCV1718   = "lcv1718"
	cmdIOADC        = "ioadc"
	cmdIOVal        = "ioval"
	cmdIODir        = "iodir"
	cmdIOUp         = "ioup"
	cmdIODown       = "iodown"
	cmdRBuf         = "r"
	cmdRBufReset    = "rr"
	cmdRBufDel      = "rd"
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
}

// MTCVIdx represents a main track CV index.
type MTCVIdx byte

// Main track CV index constants.
const (
	MTCVNumSyncBit   MTCVIdx = iota // Number of DCC synchronization bits
	MTCVNumRepeat                   // DCC command repetitions
	MTCVNumRepeatCV                 // DCC command repetitions for main track CV programming
	MTCVNumRepeatAcc                // DCC command repetitions for accessory decoders
)

const (
	replyChSize = 1
	pushChSize  = 100
	timeout     = 5
)

// Conn is a stream oriented connection to the pico board.
type Conn interface {
	io.ReadWriteCloser
}

// Client represents a command station client instance.
type Client struct {
	conn        Conn
	mu          sync.Mutex // mutex for call
	w           *bufio.Writer
	wg          *sync.WaitGroup
	replyCh     <-chan any
	lastReadErr error
}

// New returns a new client instance.
func New(conn Conn, handler func(msg Msg, err error)) *Client {
	c := &Client{
		conn: conn,
		w:    bufio.NewWriter(conn),
		wg:   new(sync.WaitGroup),
	}
	var pushCh <-chan string
	c.replyCh, pushCh = c.reader(c.wg)
	c.pusher(c.wg, pushCh, handler)
	return c
}

// Close closes the client connection.
func (c *Client) Close() error {
	err := c.conn.Close()
	c.wg.Wait()
	return err
}

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

func (c *Client) call(cmd string, args ...any) (any, error) {
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

func (c *Client) callSingle(cmd string, args ...any) (string, error) {
	res, err := c.call(cmd, args...)
	if err != nil {
		return "", err
	}
	v, ok := res.(string)
	if !ok {
		return "", fmt.Errorf("invalid reply message type %T", res)
	}
	return string(v), nil
}

func (c *Client) callMulti(cmd string, args ...any) ([]string, error) {
	res, err := c.call(cmd, args...)
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
	v, err := c.callMulti(cmdHelp)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// Board returns bord information like controller type and unique id.
func (c *Client) Board() (*Board, error) {
	v, err := c.callSingle(cmdBoard)
	if err != nil {
		return nil, err
	}
	return parseBoard(v)
}

// Temp returns the temperature of the command station.
func (c *Client) Temp() (float64, error) {
	v, err := c.callSingle(cmdTemp)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(v, 64)
}

// MTEnabled returns true if the main track DCC sigal generation is enabled, false otherwise.
func (c *Client) MTEnabled() (bool, error) {
	v, err := c.callSingle(cmdMTEnabled)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetMTEnabled sets main track DCC sigal generation whether to enabled or disabled.
func (c *Client) SetMTEnabled(enabled bool) (bool, error) {
	v, err := c.callSingle(cmdMTEnabled, enabled)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// MTCV returns the value of a main track CV.
func (c *Client) MTCV(idx MTCVIdx) (byte, error) {
	v, err := c.callSingle(cmdMTCV, idx)
	if err != nil {
		return 0, err
	}
	return parseByte(v)
}

// SetMTCV sets the value of a main track CV.
func (c *Client) SetMTCV(idx MTCVIdx, val byte) (byte, error) {
	v, err := c.callSingle(cmdMTCV, idx, val)
	if err != nil {
		return 0, err
	}
	return parseByte(v)
}

// LocoDir returns the direction of a loco.
// true : forward direction
// false: backward direction
func (c *Client) LocoDir(addr uint) (bool, error) {
	v, err := c.callSingle(cmdLocoDir, addr)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetLocoDir sets the direction of a loco.
// true : forward direction
// false: backward direction
func (c *Client) SetLocoDir(addr uint, dir bool) (bool, error) {
	v, err := c.callSingle(cmdLocoDir, addr, dir)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// ToggleLocoDir toggles the direction of a loco.
func (c *Client) ToggleLocoDir(addr uint) (bool, error) {
	v, err := c.callSingle(cmdLocoDir, addr, charToggle)
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
	v, err := c.callSingle(cmdLocoSpeed128, addr)
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
	v, err := c.callSingle(cmdLocoSpeed128, addr, speed)
	if err != nil {
		return 0, err
	}
	return parseUint(v)
}

// LocoFct returns a function value of a loco.
func (c *Client) LocoFct(addr, no uint) (bool, error) {
	v, err := c.callSingle(cmdLocoFct, addr, no)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetLocoFct sets a function value of a loco.
func (c *Client) SetLocoFct(addr, no uint, fct bool) (bool, error) {
	v, err := c.callSingle(cmdLocoFct, addr, no, fct)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// ToggleLocoFct toggles a function value of a loco.
func (c *Client) ToggleLocoFct(addr, no uint) (bool, error) {
	v, err := c.callSingle(cmdLocoFct, addr, no, charToggle)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetLocoCVByte sets the indexed CV byte value of a loco.
func (c *Client) SetLocoCVByte(addr, idx uint, val byte) (byte, error) {
	v, err := c.callSingle(cmdLocoCVByte, addr, idx, val)
	if err != nil {
		return 0, err
	}
	return parseByte(v)
}

// SetLocoCVBit sets the indexed CV bit value of a loco.
func (c *Client) SetLocoCVBit(addr, idx uint, bit byte, val bool) (bool, error) {
	v, err := c.callSingle(cmdLocoCVBit, addr, idx, bit, val)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetLocoCV29Bit5 sets the CV 29 bit 5 value of a loco.
func (c *Client) SetLocoCV29Bit5(addr uint, bit bool) (bool, error) {
	v, err := c.callSingle(cmdLocoCV29Bit5, addr, bit)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetLocoLaddr sets the long address of a loco.
func (c *Client) SetLocoLaddr(addr, laddr uint) (uint, error) {
	v, err := c.callSingle(cmdLocoLaddr, addr, laddr)
	if err != nil {
		return 0, err
	}
	return parseUint(v)
}

// LocoCV1718 returns (calculates) the CV17 and CV18 values (long address) from a loco address.
func (c *Client) LocoCV1718(addr uint) (byte, byte, error) {
	v, err := c.callSingle(cmdLocoCV1718, addr)
	if err != nil {
		return 0, 0, err
	}
	return parseByteTuple(v)
}

// IOADC returns the 'raw' value of the ADC input.
func (c *Client) IOADC(input uint) (float64, error) {
	v, err := c.callSingle(cmdIOADC, input)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(v, 64)
}

// IOVal returns the boolean value of the GPIO.
func (c *Client) IOVal(cmd, gpio uint) (bool, error) {
	v, err := c.callSingle(cmdIOVal, cmd, gpio)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetIOVal sets the boolean value of the GPIO.
func (c *Client) SetIOCmdb(cmd, gpio uint, value bool) (bool, error) {
	v, err := c.callSingle(cmdIOVal, cmd, gpio, value)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// ToggleIOVal toggles the value of the GPIO.
func (c *Client) ToggleIOCmdb(cmd, gpio uint) (bool, error) {
	v, err := c.callSingle(cmdIOVal, cmd, gpio, charToggle)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// IODir returns the direction of the GPIO.
// false: in
// true:  out
func (c *Client) IODir(cmd, gpio uint) (bool, error) {
	v, err := c.callSingle(cmdIODir, cmd, gpio)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetIODir sets the direction of the GPIO.
// false: in
// true:  out
func (c *Client) SetIODir(cmd, gpio uint, value bool) (bool, error) {
	v, err := c.callSingle(cmdIODir, cmd, gpio, value)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// ToggleIODir toggles the direction of the GPIO.
func (c *Client) ToggleIODir(cmd, gpio uint) (bool, error) {
	v, err := c.callSingle(cmdIODir, cmd, gpio, charToggle)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// IOUp returns the pull-up status of the GPIO.
func (c *Client) IOUp(cmd, gpio uint) (bool, error) {
	v, err := c.callSingle(cmdIOUp, cmd, gpio)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetIOUp sets the pull-up status of the GPIO.
func (c *Client) SetIOUp(cmd, gpio uint, value bool) (bool, error) {
	v, err := c.callSingle(cmdIOUp, cmd, gpio, value)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// ToggleIOUp toggles the pull-up status of the GPIO.
func (c *Client) ToggleIOUp(cmd, gpio uint) (bool, error) {
	v, err := c.callSingle(cmdIOUp, cmd, gpio, charToggle)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// IODown returns the pull-down status of the GPIO.
func (c *Client) IODown(cmd, gpio uint) (bool, error) {
	v, err := c.callSingle(cmdIODown, cmd, gpio)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// SetIODown sets the pull-down status of the GPIO.
func (c *Client) SetIODown(cmd, gpio uint, value bool) (bool, error) {
	v, err := c.callSingle(cmdIODown, cmd, gpio, value)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// ToggleIODown toggles the pull-down status of the GPIO.
func (c *Client) ToggleIODown(cmd, gpio uint) (bool, error) {
	v, err := c.callSingle(cmdIODown, cmd, gpio, charToggle)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// RBuf returns the command station refresh buffer (debugging).
func (c *Client) RBuf() (*RBuf, error) {
	v, err := c.callMulti(cmdRBuf)
	if err != nil {
		return nil, err
	}
	return parseRBuf(v)
}

// RBufReset resets the refresh buffer (debugging).
func (c *Client) RBufReset() (bool, error) {
	v, err := c.callSingle(cmdRBufReset)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

// RBufDel deletes address addr from refresh buffer (debugging).
func (c *Client) RBufDel(addr uint) (uint, error) {
	v, err := c.callSingle(cmdRBufDel, addr)
	if err != nil {
		return 0, err
	}
	return parseUint(v)
}
