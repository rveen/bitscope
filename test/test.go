package main

import (
    "bitscope"
    "log"
    "fmt"
)

func main() {
    bs, err := bitscope.Open("")

	if err != nil {
		log.Fatal(err)
	}

	defer bs.Close()

	bs.Reset()

	bs.Vertical("10v")
	bs.Horizontal(1, 40) // 40/1 = 1MHz
	bs.TriggerTiming(0, 0, 1)

	bs.Trace(0, 1000, 0)

	b, _ := bs.Dump(1024)

	println(len(b))

	for i := 0; i < len(b); i++ {
		fmt.Printf("%02x ", b[i])
	}
	println("")
}
