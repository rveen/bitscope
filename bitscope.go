// For the license see the LICENSE file (BSD style)

// Bitscope BSNG API.
//
// This package will expose functionality of the BS10 and BS05 USB oscilloscopes
// from BitScope.
//
// Objectives:
// - Keep it simple
// - Do not expose the VM
//
// See bitscope.com for more information on PC oscilloscopes.
//
package bitscope

import (
	//"log"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

type BitScope struct {
	tty *os.File
}

// Open opens a connection to the BitScope instrument.
//
// Bug: this call is Linux specific for now.
func Open(dev string) (*BitScope, error) {

	const base string = "/dev/ttyUSB"

	switch len(dev) {

	case 0:
		dev = base + "0"
	case 1:
		fallthrough
	case 2:
		dev = base + dev

	}

	tty, err := os.OpenFile(dev, os.O_RDWR | syscall.O_NOCTTY , 0666)

	if err != nil {
		return nil, err
	}

    // Setting serial options must be done in a portable and (ideally) robust 
    // way, in the sense that it doesn't break in future releases of Go. For 
    // now the stty command is an easy way. 
    //
    // TODO: check syscall.Termios. 
	err = exec.Command("stty", "-F", dev, "-icanon", "min", "1", "-icrnl", "-echo").Run()

	if err != nil {
		tty.Close()
		return nil, err
	}

	bs := BitScope{tty}

	return &bs, nil
}

// Close ends the connection to the BisScope
func (bs *BitScope) Close() error {
	return bs.tty.Close()
}

// Reset instructs the BitScope to do a soft reset
func (bs *BitScope) Reset() {
	bs.Call([]byte("!"))
}

// Id returns a string identifying the VM revision
func (bs *BitScope) Id() string {
	b, err := bs.Call([]byte("?"))
	if len(b)==0  || err!=nil {
	    return ""
	}
	return strings.TrimSpace(string(b[1:]))
}

// Stop terminates a command sequence
func (bs *BitScope) Stop() {
	bs.Call([]byte("."))
}

// Led controls the intensity of the 3 LEDs on the BS10, one at a time.
func (bs *BitScope) Led(n, i int) {

	b := []byte("[fa]@[00]s")

	s := strconv.FormatInt(int64(i&0xff), 16)

	if len(s) == 1 {
		b[7] = s[0]
	} else {
		b[6] = s[0]
		b[7] = s[1]
	}

	switch n {
	case 'g': // Green
		b[2] = 'b'
	case 'y': // Yellow
		b[2] = 'c'
	}

	bs.Call(b)
}

/* Capture options

Bit 5 Level/Edge Trigger 0 Trigger operation is Level Sensitive.
 - 0 Trigger operation is Level Sensitive.
 - 1 Trigger operation is Edge Transition Sensitive.

Bit 4 Edge Direction
- 0 Trigger asserted on FALSE -> TRUE
- 1 Trigger asserted on TRUE -> FALSE

Bit 3 Page Selection
- 0 Lower 16K RAM Page and Analogue BNC Input.
- 1 Upper 16K RAM Page and Analogue POD Input.

Bits 2,1 Trig Bit 7 MUX 0
 -0 DD7 : Digital Data Bus Bit 7.
- 1 Comparator : trigger match comparator signal.
- 2 Event 1 : (Pre-scaler output frequency halved).
- 3 Event 2 : (ADC input frequency halved)

Bit 0 Trigger Source
- 0 Digital trigger source.
- 1 Analogue trigger source.

Bits 6 and 7 are reserved and should be set to 0

*/

func (bs *BitScope) CaptureConfig(edge, dir, page, source bool, mode int) {

}

/* Input/ attenuation configuration

R14:

Primary channel: low nibble
Secondary channel: high nibble

Bit 1,0 Attenuation Range (0-3)
Bit 2 Channel Select (A,B)
Bit 3 zz-clk level 1 Always set to one.

*/

func (bs *BitScope) InputConfig(ch1, att1, ch2, att2 int) {
	b := []byte("[e]@[ ]sp")
	b[5] = byte((att1 & 3) | (ch1 & 1 << 2) | (att2 & 3 << 4) | (ch2 & 1 << 6) | 0x88)
	bs.Call(b)
}

/* Trace config

R8 Trace Register Trace mode selection.
R11 Post Trigger Delay (Low Byte) Delay after trigger (low byte).
R12 Post Trigger Delay (High Byte) Delay after trigger (high byte).
R13 Time-base Expansion Time-base expansion factor.
R20 Pre-Trigger Delay Programmed to pre-fill buffer before trigger match.

R8 :
0 Simple Trace Mode: Single Channel, Level Trigger.
1 Simple Trace Mode Channel Chop Enhanced Trigger.
2 Time-base Expansion Single Channel Enhanced Trigger.
3 Time-base Expansion Channel Chop Enhanced Trigger.
4 Slow Clock Mode Channel Chop Enhanced Trigger.
8 Frequency Measurement

*/

func (bs *BitScope) TraceConfig(mode int) {

	b := []byte("[8]@[ ]sp")
	b[5] = byte('0' + mode)

	bs.Call(b)
}

/* Trace (capture actual data)

   States:
   - State 1 Pre Trigger Delay
   - State 2 Trigger Enabled
   - State 3 Post Trigger Delay
*/

func (bs *BitScope) Trace() {

	b := []byte("T")

	bs.Call(b)

}

/* Read data

M 0x4D Mixed memory dump (Binary format, analogue & digital data).
A 0x41 Analog memory dump (Binary format, analogue data).

The size of each dump is determined by the values programmed to
the Dump Size register R15 and ranges from 1 to 256 samples (the value 0
implies 256)

*/

func (bs *BitScope) ReadData() {

}

/* ------------------------------------------------------------------------- */

// Call sends data to the instrument and returns its response.
func (bs *BitScope) Call(b []byte) ([]byte, error) {

	n, err := bs.tty.Write(b)

	if err != nil {
		return nil, err
	}

	if n != len(b) {
		return nil, errors.New("Not all bytes were written")
	}
	
	// We want to block until a response is received (but not forever) and 
	// not rely on the message content to decide the end of this response.
	// 
	// If we set the fd to non blocking, we may return from the call before 
	// we receive a byte. So, a time window is needed, but file reads don't
	// have a timeout option.
	// 
	// Ref: https://groups.google.com/d/msg/golang-nuts/QV-zn2JHNt4/-0YxnL7sBc8J
	//
	// BUG: to do. For now this works for responses with < 64 bytes
	
	r := make([]byte, 64)
	n, err = bs.tty.Read(r)
	
	return r[0:n], err
}
