// For the license see the LICENSE file (BSD style)

package bitscope

import (
	"fmt"
	"log"
	"strings"
	// "testing"
)

func ExampleScope_Id() {
	bs, err := Open("")

	if err != nil {
		log.Fatal(err)
	}

	defer bs.Close()

	bs.Reset()

	id := bs.Id()
	if !strings.HasPrefix(id, "BS00") || len(id) != 8 {
		log.Fatal("Incorrect ID")
	}

	// If the serial channel is not properly configured, a second call to Id()
	// may return characters from the previous request.
	id = bs.Id()
	if !strings.HasPrefix(id, "BS00") || len(id) != 8 {
		log.Fatal("Incorrect ID")
	}
}

func ExampleScope_Led() {
	bs, err := Open("")

	if err != nil {
		log.Fatal(err)
	}

	defer bs.Close()

	bs.Led('r', 0x10)
	bs.Led('g', 0x80)
	bs.Led('y', 0xc0)
}

func Example() {

	bs, err := Open("")

	if err != nil {
		log.Fatal(err)
	}

	defer bs.Close()

	bs.Reset()

	bs.Vertical("2v")
	bs.Horizontal(1, 40) // 40/1 = 1MHz
	bs.TriggerTiming(0, 0, 1)

	bs.Trace(0, 1000, 0)

	b, _ := bs.Dump(256)

	println(len(b))

	for i := 0; i < len(b); i++ {
		fmt.Printf("%02x ", b[i])
	}
	println("")
}
