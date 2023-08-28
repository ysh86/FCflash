package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var hi bool
	flag.BoolVar(&hi, "HiROM", false, "is HiROM")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	addr := 0x7fdc
	if hi {
		addr = 0xffdc
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		panic(err)
	}

	var sum uint32
	for i, d := range data {
		if i == addr+0 || i == addr+1 {
			// checksumC
			d = 0xff
		}
		if i == addr+2 || i == addr+3 {
			// checksum
			d = 0x00
		}
		sum += uint32(d)
	}

	// TODO: mirroring

	fmt.Printf("sum:       %04X\n", sum&0xffff)
	fmt.Printf("sumC:      %04X\n", (sum&0xffff)^0xffff)
	fmt.Printf("checksum:  %02X%02X\n", data[addr+3], data[addr+2])
	fmt.Printf("checksumC: %02X%02X\n", data[addr+1], data[addr+0])
}
