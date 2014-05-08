// For the license see the LICENSE file (BSD style)

package bitscope

import (
	"strings"
	"testing"
)

func TestId(t *testing.T) {
	bs, err := Open("")

	if err != nil {
		t.Fatal(err)
	}

	defer bs.Close()

	bs.Reset()

	id := bs.Id()
	if !strings.HasPrefix(id, "BS00") || len(id) != 8 {
		t.Error("Incorrect ID")
	}

	// If the serial channel is not properly configured, a second call to Id()
	// may return characters from the previous request.
	id = bs.Id()
	if !strings.HasPrefix(id, "BS00") || len(id) != 8 {
		t.Error("Incorrect ID")
	}
}

func TestLeds(t *testing.T) {
	bs, err := Open("")

	if err != nil {
		t.Fatal(err)
	}

	defer bs.Close()

	bs.Led('r', 0x10)
	bs.Led('g', 0x80)
	bs.Led('y', 0xc0)
}
