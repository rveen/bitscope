// For the license see the LICENSE file (BSD style)

// Bitscope BSNG API.
//
// This package will expose functionality of the BS10 and BS05 USB oscilloscopes
// from BitScope, while hiding the the virtual machine inside them.
//
// See https://bitscope.com for more information on these PC oscilloscopes.
//
package bitscope

import (
	"errors"
	"strconv"
	"strings"
)

// Reset instructs the BitScope to do a soft reset
func (bs *Scope) Reset() {
	bs.call([]byte("!"))
}

// Stop terminates a command sequence
func (bs *Scope) Stop() {
	bs.call([]byte("."))
}

// Led controls the intensity of the 3 LEDs of the BS10, one at a time.
func (bs *Scope) Led(n, i uint) {

	b := []byte("fa@00s")
	hex1(i, b, 3)

	switch n {
	case 'g': // Green
		b[1] = 'b'
	case 'y': // Yellow
		b[1] = 'c'
	}

	bs.call(b)
}

/* -------------------------------------------------------------------------
   Trace
   -------------------------------------------------------------------------*/

// TraceTerminate is used to 'manually' end the data acquisition, instead
// of using a trigger event.
func (bs *Scope) TraceTerminate() {
	bs.call([]byte("K"))
}

// Trace starts the data acquisition and waits until it has completed.
// The parameters pre and post are the pre-trigger and post-trigger number
// of samples, and the delay is specified in us. The delay is a time window
// after the trigger in which no samples are recorded.
func (bs *Scope) Trace(pre, post, delay uint) ([]byte, error) {

	bs.call([]byte("[7b]@[80]s")) // KitchenSinkA (enable hardware comparators)
	bs.call([]byte("[7c]@[80]s")) // KitchenSinkB (enable analog filter)
	bs.call([]byte("[37]@[01]s")) // AnalogEnable (enable CHA input circuits)
	bs.call([]byte("[31]@[00]s")) // buffer mode
	bs.call([]byte("[21]@[00]s")) // trace mode

	// delay, pre, post
	a := []byte("22@00z00z00z00s")
	b := []byte("26@00z00s")
	c := []byte("2a@00z00s")
	hex4(delay, a, 3)
	hex2(pre, b, 3)
	hex2(post, c, 3)
	bs.call(a)
	bs.call(b)
	bs.call(c)

	// Logic trigger
	bs.call([]byte("[06]@[7f]s"))           // TriggerMask (set the trigger logic mask)
	bs.call([]byte("[05]@[80]s"))           // TriggerLogic (program the trigger logic)
	bs.call([]byte("[44]@[00]s[45]@[00]s")) // TriggerValue (set digital trigger level, optional)
	bs.call([]byte("[68]@[f5]s[69]@[68]s")) // TriggerLevel (set analog trigger level)
	bs.call([]byte("[07]@[21]s"))           // SpockOption (choose edge triggered comparator mode)
	bs.call([]byte("[3a]@[00]s[3b]@[00]s")) // Prelude (set the buffer default value; “zero”)

	// trace start address
	bs.call([]byte("[08]@[00]s[09]@[00]s[0a]@[00]s"))

	bs.call([]byte(">"))
	bs.call([]byte("U"))

	b = []byte("D")
	return bs.callCr(b, 5, 256)
}

/* -------------------------------------------------------------------------
   Dump
   -------------------------------------------------------------------------*/

// Dump reads the data buffer from the BitScope into a byte array. This buffer
// contains the data acquired during the trace phase.
func (bs *Scope) Dump(size uint) ([]byte, error) {

	b := []byte("[31]@[00]s" + // BufferMode
		"[08]@[cc]s[09]@[00]s[0a]@[00]s" + // Start address
		"[1e]@[00]s" + // DumpMode (raw)
		"[30]@[00]s") // DumpChan
	bs.call(b)

	// Set the dump size (number of data bytes to return)
	b = []byte("1c@00z00s")
	hex2(size, b, 3)
	bs.call(b)

	b = []byte("[16]@[01]s[17]@[00]s" + // DumpRepeat
		"[18]@[01]s[19]@[00]s" + // DumpSend
		"[1a]@[ff]s[1b]@[ff]s" + // DumpSkip
		">")
	bs.call(b)

	return bs.callWait([]byte("A"), 100, size+256)
}

/* -------------------------------------------------------------------------
   Horizontal
   -------------------------------------------------------------------------*/

// Horizontal sets the time base/scale of the trace.
func (bs *Scope) Horizontal(pre, div uint) error {

	// Prescaler, divisor
	b := []byte("14@00z00s" + "2e@00z00s")

	hex2(pre, b, 3)
	hex2(div, b, 12)

	_, err := bs.call(b)
	return err
}

/* -------------------------------------------------------------------------
   Vertical
   -------------------------------------------------------------------------*/

// Vertical sets the voltage range of the trace.
func (bs *Scope) Vertical(rng string) error {

	var a string

	mv := false

	// expect v, mv or nothing
	rng = strings.ToLower(rng)

	if strings.HasSuffix(rng, "v") {
		rng = rng[0 : len(rng)-1]
	}

	if strings.HasSuffix(rng, "m") {
		rng = rng[0 : len(rng)-1]
		mv = true
	}

	v, _ := strconv.ParseFloat(rng, 32)
	if mv {
		v = v / 1000.0
	}

	switch bs.Model {

	case "bs10":
		switch {
		case v <= 0.52:
			a = "64@54z65s" + "66@96x6cs"
		case v <= 1.1:
			a = "64@47z61s" + "66@a2z70s"
		case v <= 3.5:
			a = "64@86z50s" + "66@64z81s"
		case v <= 5.2:
			a = "64@a7z44s" + "66@42z8ds"
		case v <= 11:
			a = "64@28z1cs" + "66@c1zb5s"
		default:
			return errors.New("Unsupported vertical range")
		}

	case "bs05":
		switch {
		case v <= 1.1:
			a = "64@d6z65s" + "66@bcz69s"
		case v <= 3.5:
			a = "64@62z52s" + "66@3fz7ds"
		case v <= 5.2:
			a = "64@68z44s" + "66@ffz8as"
		case v <= 11:
			a = "64@6az12s" + "66@8czbas"
		default:
			return errors.New("Unsupported vertical range")
		}

	default:
		return errors.New("Unsupported model")
	}

	bs.call([]byte(a))

	return nil
}

/* -------------------------------------------------------------------------
   Trigger
   -------------------------------------------------------------------------*/

// Trigger sets the analog trigger to the specified channel and voltage threshold.
func (bs *Scope) Trigger(src, level uint) {

	bs.trigSrc = src

	b := []byte("68@00z00s") // TriggerLevel (set analog trigger level)
	hex2(level, b, 3)
	bs.call(b)
}

// TriggerLogic sets the trigger to logic mode with the given bit levels and
// mask. The mask parameter identifies bits whose state is to be ignored by
// the trigger comparator.
func (bs *Scope) TriggerLogic(level, mask uint) {

	// TriggerMask, TriggerLogic, Level ???
	b := []byte("05@00s" + "06@00s")

	hex1(level, b, 3)
	hex1(mask, b, 9)

	bs.call(b)
}

/*

6 Trigger Invert 0 => normal, 1 => invert
5 Trigger Mode 0 => LEVEL, 1 => EDGE
4 Trigger Edge 0 => FALSE to TRUE, 1 => TRUE to FALSE
2 Trigger Source 0 => CHA, 1 => CHB
1 Trigger Swap 0 => normal, 1 => swap upon trigger
0 Trigger Type 0 => sampled analog, 1 => hardware comparator

*/

// TriggerMode sets the mode (level or edge), edge (0->1 or 1->0), and hardware
// comparator (active or not).
//
// TODO: invert, swap: ??
func (bs *Scope) TriggerMode(mod, edge, comp bool) {

	var mode uint

	if mod {
		mode |= 0x20
	}

	if edge {
		mode |= 0x10
	}

	if comp {
		mode |= 1
	}

	if bs.trigSrc == 'b' {
		mode |= 4
	}

	b := []byte("07@00s")
	hex1(mode, b, 3)
	bs.call(b)
}

// TriggerTiming sets the timing parameters associated with a trigger.
//
// For a trigger to be valid, the trigger condition must be false during
// hold-off, then true during hold-on. Also a timeout can be specified,
// for the case that the trigger event never happens.
//
// Hold-off time: 0 .. 2^16; tick = 1/Fs
// Hold-on time: 0 .. 2^16; tick = 1/Fs
// Timeout: 0 .. 2^16; tick = 6.4 us. 0 = no timeout.
func (bs *Scope) TriggerTiming(hoff, hon, timeout uint) {

	// TriggerIntro, TriggerOutro, vrTimeout
	b := []byte("32@00z00s" + "34@00z00s" + "2c@00z00s")

	hex2(hoff, b, 3)
	hex2(hon, b, 12)
	hex2(timeout, b, 21)

	bs.call(b)
}
