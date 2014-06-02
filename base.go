// For the license see the LICENSE file (BSD style)

package bitscope

import (
	"errors"
	"fmt"
	"github.com/pkg/term"
	"strings"
	"time"
)

type Scope struct {
	tty *term.Term
	// The ID string returned by the BitScope
	ID string
	// The model of the attached scope ('bs10' or 'bs05')
	Model   string
	trigSrc uint
}

// Open opens a connection to a BitScope instrument.
//
// If the ID string returned by the BitScope is not recognized as one of the
// supported ones, an error is returned.
func Open(dev string) (*Scope, error) {

	const base string = "/dev/ttyUSB"

	switch len(dev) {

	case 0:
		dev = base + "0"
	case 1:
		fallthrough
	case 2:
		dev = base + dev

	}

	tty, err := term.Open(dev)

	if err != nil {
		return nil, err
	}

	tty.SetRaw()

	bs := Scope{tty, "", "", 0}

	bs.ID = bs.Id()
	if strings.HasPrefix(bs.ID, "BS0010") {
		bs.Model = "bs10"
	} else if strings.HasPrefix(bs.ID, "BS0005") {
		bs.Model = "bs05"
	} else {
		tty.Close()
		return nil, errors.New("Unsupported model: " + bs.ID)
	}

	return &bs, nil
}

// Close ends the connection to the BisScope
func (bs *Scope) Close() error {
	return bs.tty.Close()
}

// Id returns a string identifying the VM revision
//
// Use bs.ID instead of this function unless you want a to explicitly ask the
// BitScope for its ID.
func (bs *Scope) Id() string {
	b, err := bs.call([]byte("?"))
	if len(b) == 0 || err != nil {
		return ""
	}
	return strings.TrimSpace(string(b[1:]))
}

// call sends data to the instrument and returns its response. Its waits a
// fixed time of 2ms for the response to arrive.
func (bs *Scope) call(b []byte) ([]byte, error) {

	n, err := bs.tty.Write(b)

	if err != nil {
		return nil, err
	}

	// BUG: to do. For now this works for responses with < 256 bytes

	time.Sleep(time.Millisecond * 2)

	r := make([]byte, 256)
	n, err = bs.tty.Read(r)

	var c byte

	for i := 0; i < n; i++ {

		c = r[i]
		if c < 32 {
			c = '_'
		}
		fmt.Printf("%c", c)

	}
	fmt.Println("")

	return r[0:n], err
}

// call sends data to the instrument and returns its response.
func (bs *Scope) callWait(b []byte, ms int) ([]byte, error) {

	n, err := bs.tty.Write(b)

	if err != nil {
		return nil, err
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
	// BUG: to do. For now this works for responses with < 256 bytes

	time.Sleep(time.Millisecond * time.Duration(ms))

	r := make([]byte, 256)
	n, err = bs.tty.Read(r)

	var c byte

	for i := 0; i < n; i++ {

		c = r[i]
		if c < 32 {
			c = '_'
		}
		fmt.Printf("%c", c)

	}
	fmt.Println("")

	return r[0:n], err
}

// call sends data to the instrument and returns its response. It waits until
// it receives the specified number of CR characters (ASCII 13).
func (bs *Scope) callCr(b []byte, cr int) ([]byte, error) {

	n, err := bs.tty.Write(b)

	if err != nil {
		return nil, err
	}

	if n != len(b) {
		return nil, errors.New("Not all bytes were written")
	}

	// Read until the specified number of CRs have been read.

	var res []byte
	r := make([]byte, 256)

	for {

		n, err = bs.tty.Read(r)
		if err != nil {
			break
		}

		if n > 0 {
			res = append(res, r[0:n]...)
		}

		// Count CR's
		n := 0
		for i := 0; i < len(res); i++ {
			if res[i] == 13 {
				n++
			}
		}
		if n >= cr {
			break
		}
	}

	var c byte
	for i := 0; i < len(res); i++ {

		c = res[i]
		if c < 32 {
			c = '_'
		}
		fmt.Printf("%c", c)

	}
	fmt.Println("")

	return res, err
}

// hex converts a small unsigned integer (0-255) into its hex alphanumeric
// representation, at a speficied position in the array given.

const h string = "0123456789abcdef"

func hex1(n uint, b []byte, offset int) {
	n = n & 0xff

	b[offset+1] = h[n&15]
	b[offset] = h[(n>>4)&15]
}

func hex2(n uint, b []byte, offset int) {
	n = n & 0xffff

	b[offset+1] = h[n&15]
	b[offset] = h[(n>>4)&15]
	b[offset+4] = h[(n>>8)&15]
	b[offset+3] = h[(n>>12)&15]
}

func hex3(n uint, b []byte, offset int) {
	n = n & 0xffffff

	b[offset+1] = h[n&15]
	b[offset] = h[(n>>4)&15]
	b[offset+4] = h[(n>>8)&15]
	b[offset+3] = h[(n>>12)&15]
	b[offset+7] = h[(n>>16)&15]
	b[offset+6] = h[(n>>20)&15]
}

func hex4(n uint, b []byte, offset int) {
	n = n & 0xffffffff

	b[offset+1] = h[n&15]
	b[offset] = h[(n>>4)&15]
	b[offset+4] = h[(n>>8)&15]
	b[offset+3] = h[(n>>12)&15]
	b[offset+7] = h[(n>>16)&15]
	b[offset+6] = h[(n>>20)&15]
	b[offset+10] = h[(n>>24)&15]
	b[offset+9] = h[(n>>32)&15]
}
