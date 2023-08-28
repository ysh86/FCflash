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
	buf, err := os.ReadFile(fileName)
	if err != nil {
		panic(err)
	}

	// header
	FCflash.ParseHeader(buf)

	// checksum
	checkSum := uint32(0)
	for i, b := range buf {
		if i != 0x014e && i != 0x014f {
			checkSum += uint32(b)
		}
	}
	fmt.Printf("%s: %04x\n", fileName, checkSum&0xffff)
}
