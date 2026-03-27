package main

import (
	"fmt"
	"os"

	"github.com/ysh86/FCflash"
)

func main() {
	if len(os.Args) < 2 {
		panic(os.Args)
	}

	fileName := os.Args[1]
	data, err := os.ReadFile(fileName)
	if err != nil {
		panic(err)
	}

	// header
	FCflash.ParseHeader(data)

	// global sum
	global := uint32(0)
	for i, d := range data {
		if i != 0x014e && i != 0x014f {
			global += uint32(d)
		}
	}
	fmt.Printf("%s: %04x\n", fileName, global&0xffff)
}
