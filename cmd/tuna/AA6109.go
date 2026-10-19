package main

import (
	"fmt"
	"io"
	"os"
)

func dumpAA6109PRG(f io.Writer, s io.ReadWriter, prg int) (err error) {
	// outer bank select
	//
	// $5001: A20-17,16-13 -> 8KB x16 x16 -> 2MB
	// $5002: A28-21       -> 2MB x256    -> 512MB
	all := prg*16*1024 - 1
	A24_21 := uint8((all >> 21) & 0x0f)
	A20_13 := uint8((all >> 13) & 0xff)

	// fmt.Fprintf(os.Stderr, "\nA24-21: %04b, A20-13: %08b\n", A24_21, A20_13)

	// D~[L..M MMMM]
	//    |  +-++++-- Mapper for the FPGA to emulate
	//    +---------- 1=Lock supervisor registers until reset
	reg5000 := uint8(0) // UNROM $8-$B 16KB bank

	// D~[BBBB BBBB]
	//    ++++-++++- PRG A20..A13
	reg5001 := uint8(0)

	// D~[BBBB BBBB]
	//    ++++-++++- PRG A28..A21
	reg5002 := uint8(0)

	// D~[MMMM MMMM]
	//    ++++-++++- PRG A20..A13 mask
	reg5003 := uint8(0xff)

	// ext. ROM select
	setReg("4000", 0, s)

	setReg("5003", reg5003, s)
	setReg("5000", reg5000, s)

	// bank select
	// Max 32MB: j(16) x i(128) x (8KB x2)
	fmt.Fprintln(os.Stderr, "")
	for j := uint8(0); j <= A24_21; j++ {
		r2 := reg5002 | j
		setReg("5002", r2, s)
		for i := 0; i <= int(A20_13-1); i += 2 {
			r1 := reg5001 | uint8(i)
			setReg("5001", r1, s)

			fmt.Fprintf(os.Stderr, "regs $5002,1=%02X%02X\n", r2, r1)
			kickoffDumpHex("8000", "BFFF", f, s)
		}
	}

	return nil
}
