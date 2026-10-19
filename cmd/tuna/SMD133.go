package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/marcinbor85/gohex"
)

func dumpSmd133aPRG(f io.Writer, s io.ReadWriter, prg int) (err error) {
	l0 := uint8(0) //uint8((begin & 0b11000000) >> 6)
	k0 := uint8(0) //uint8((begin & 0b00111000) >> 3)
	j0 := uint8(0) //uint8((begin & 0b00000111))

	// outer bank select
	//
	// $5000: A24,23 -> 8MB x4, A19,18,17 -> 128KB x8
	// $5001: A A20,22,21 -> 1MB x8
	//        B A20,21,22 -> 1MB x8
	// $5003: A16,15 -> 32KB x4
	all := prg*16*1024 - 1
	A2423 := uint8((all >> 23) & 0b11)
	A222120 := uint8((all >> 20) & 0b111)
	A191817 := uint8((all >> 17) & 0b111)
	A1615 := uint8((all >> 15) & 0b11)

	// fmt.Fprintf(os.Stderr, "\nA24-23: %02b, A22-20: %03b, A19-17: %03b, A16-15: %02b\n", A2423, A222120, A191817, A1615)

	//  7  bit  0
	//  ---- ----
	//  ABCC DEEE
	//  |||| ||||
	//  |||| |+++-- PRG offset (PRG A19, A18, A17)
	//  |||| +----- Alternate CHR A17
	//  ||++------- PRG offset (PRG A24, A23)
	//  ||++------- CHR offset (CHR A19, A18) on CHR-ROM-having boards
	//  |+--------- PRG mask (PRG A17 from 0: MMC3; 1: offset)
	//  +---------- CHR mask (CHR A17 from 0: MMC3; 1: alternate)
	reg5000 := uint8(0b0100_0000)

	//  7  bit  0
	//  ---- ----
	//  GHIJ KKLS
	//  |||| |||+-- SC0 bit (see below)
	//  |||| ||+--- GNROM mode bank PRG size (1: 32 KiB bank, PRG A14=CPU A14; 0: 16 KiB bank, PRG A14=offset A14)
	//  |||+-++---- PRG offset (in order: PRG A20, A22, A21)
	//  ||+-------- PRG mask (PRG A20 from 0: offset; 1: MMC3)
	//  |+--------- PRG mask (PRG A19 from 0: offset; 1: MMC3)
	//  +---------- PRG mask (PRG A18 from 0: MMC3; 1: offset)
	reg5001a := uint8(0b1000_0010)

	//  7  bit  0
	//  ---- ----
	//  NPZP QQRS
	//  |||| |||+-- SC1 bit (see below)
	//  |||| +++--- PRG offset for GNROM mode (PRG A16, A15, A14)
	//  |||+------- 1: GNROM mode; 0: MMC3 mode
	//  ||||         (1: PRG A16...13 from QQ, L, R, CPU A14, A13 + CHR A16...10 from MMMM, PPU A12...10;
	//  ||||          0: PRG A16...13 from MMC3 + CHR A16...A10 from MMC3 )
	//  ||+-------- 1: Also enable PRG RAM in $5000-$5FFF
	//  |+-+------- Banking mode
	//  |+--------- "Weird MMC3 mode"
	//  +---------- Lockout (prevent further writes to all registers but the one at $xxx2, only works in MMC3 mode)
	reg5003 := uint8(0b0001_0000)

	// address of registers
	reg0 := "2f"
	reg1 := "30"
	reg3 := "32"

	// ext. ROM select
	setReg("4000", 0, s)

	// bank select
	// Max 32MB: l(4) x k(8) x j(8) x i(4) x 32KB
	fmt.Fprintln(os.Stderr, "")
	for l := l0; l <= A2423; l++ {
		r00 := reg5000 | (l << 4)
		for k := k0; k <= A222120; k++ {
			// k: A22, A21, A20 -> A20, A22, A21
			kk := ((k & 0b001) << 2) | ((k & 0b010) >> 1) | ((k & 0b100) >> 1)
			r1 := reg5001a | (kk << 2)
			setReg(reg1, r1, s)
			for j := j0; j <= A191817; j++ {
				r0 := r00 | j
				setReg(reg0, r0, s)
				for i := uint8(0); i <= A1615; i++ {
					r3 := reg5003 | (i << 2)
					setReg(reg3, r3, s)

					fmt.Fprintf(os.Stderr, "regs $5001=%02X, $5000=%02X, $5003=%02X\n", r1, r0, r3)
					kickoffDump32K(f, s)
				}
			}
		}
	}

	return nil
}

func writeSmd133aPRG(s io.ReadWriter, f io.ReadSeeker, prg int) (err error) {
	// skip: 128KB xN
	begin := (prg >> 3) - 1
	l0 := uint8((begin & 0b11000000) >> 6)
	k0 := uint8((begin & 0b00111000) >> 3)
	j0 := uint8((begin & 0b00000111))
	f.Seek(int64(begin)*128*1024, io.SeekCurrent)

	// outer bank select
	//
	// $5000: A24,23 -> 8MB x4, A19,18,17 -> 128KB x8
	// $5001: A A20,22,21 -> 1MB x8
	//        B A20,21,22 -> 1MB x8
	all := prg*16*1024 - 1
	A2423 := uint8((all >> 23) & 0b11)
	A222120 := uint8((all >> 20) & 0b111)
	A191817 := uint8((all >> 17) & 0b111)

	// fmt.Fprintf(os.Stderr, "\nA24-23: %02b, A22-20: %03b, A19-17: %03b\n", A2423, A222120, A191817)

	//  7  bit  0
	//  ---- ----
	//  ABCC DEEE
	//  |||| ||||
	//  |||| |+++-- PRG offset (PRG A19, A18, A17)
	//  |||| +----- Alternate CHR A17
	//  ||++------- PRG offset (PRG A24, A23)
	//  ||++------- CHR offset (CHR A19, A18) on CHR-ROM-having boards
	//  |+--------- PRG mask (PRG A17 from 0: MMC3; 1: offset)
	//  +---------- CHR mask (CHR A17 from 0: MMC3; 1: alternate)
	reg5000 := uint8(0b0100_0000)

	//  7  bit  0
	//  ---- ----
	//  GHIJ KKLS
	//  |||| |||+-- SC0 bit (see below)
	//  |||| ||+--- GNROM mode bank PRG size (1: 32 KiB bank, PRG A14=CPU A14; 0: 16 KiB bank, PRG A14=offset A14)
	//  |||+-++---- PRG offset (in order: PRG A20, A22, A21)
	//  ||+-------- PRG mask (PRG A20 from 0: offset; 1: MMC3)
	//  |+--------- PRG mask (PRG A19 from 0: offset; 1: MMC3)
	//  +---------- PRG mask (PRG A18 from 0: MMC3; 1: offset)
	reg5001a := uint8(0b1000_0000)

	//  7  bit  0
	//  ---- ----
	//  NPZP QQRS
	//  |||| |||+-- SC1 bit (see below)
	//  |||| +++--- PRG offset for GNROM mode (PRG A16, A15, A14)
	//  |||+------- 1: GNROM mode; 0: MMC3 mode
	//  ||||         (1: PRG A16...13 from QQ, L, R, CPU A14, A13 + CHR A16...10 from MMMM, PPU A12...10;
	//  ||||          0: PRG A16...13 from MMC3 + CHR A16...A10 from MMC3 )
	//  ||+-------- 1: Also enable PRG RAM in $5000-$5FFF
	//  |+-+------- Banking mode
	//  |+--------- "Weird MMC3 mode"
	//  +---------- Lockout (prevent further writes to all registers but the one at $xxx2, only works in MMC3 mode)
	reg5003 := uint8(0b0000_0000)

	buf8K := make([]byte, 8*1024)

	// ext. ROM select
	setReg("4000", 0, s)

	// bank select
	// Max 32MB: l(4) x k(8) x j(8) x 128KB, MMC3 i(16) x 8KB
	fmt.Fprintln(os.Stderr, "")
	for l := l0; l <= A2423; l++ {
		r00 := reg5000 | (l << 4)
		for k := k0; k <= A222120; k++ {
			// k: A22, A21, A20 -> A20, A22, A21
			kk := ((k & 0b001) << 2) | ((k & 0b010) >> 1) | ((k & 0b100) >> 1)
			r1 := reg5001a | (kk << 2)
			setReg("5001", r1, s)
			for j := j0; j <= A191817; j++ {
				r0 := r00 | j
				setReg("5000", r0, s)
				r3 := reg5003
				setReg("5003", r3, s)
				fmt.Fprintf(os.Stderr, "regs $5001=%02X, $5000=%02X, $5003=%02X\n", r1, r0, r3)
				// MMC3: 16 x 8KB blocks @ $A000-$BFFF
				// - $8,9: MMC3 bank select regs
				// - $A,B: (MMC3 regs), flash regs & write window
				// - $C,D: (MMC3 regs)
				// - $E,F: (MMC3 regs)
				// erase 128KB block
				cur := (l << (23 - 16)) | (k << (20 - 16)) | (j << (17 - 16))
				fmt.Fprintf(os.Stderr, "%04X: ", cur)
				fmt.Fprintf(os.Stderr, "e128")
				eraseFlash(s)
				fmt.Fprintf(os.Stderr, ".")
				for i := uint8(0); i < 16; i++ {
					// prepare src data
					_, err := io.ReadFull(f, buf8K)
					if err != nil {
						return err
					}
					fmt.Fprintf(os.Stderr, "u")
					upload8K(s, buf8K)

					// write 8KB block
					fmt.Fprintf(os.Stderr, "p8")
					kickoffFlash(s, i)
					fmt.Fprintf(os.Stderr, ".")
				}
				fmt.Fprintln(os.Stderr, "")
			}
		}
	}

	return nil
}

func dumpSmd133PRG(f io.Writer, s io.ReadWriter, prg int) (err error) {
	// outer bank select
	//
	// $5000: A24,23 -> 8MB x4, A19,18,17 -> 128KB x8
	// $5001: A A20,22,21 -> 1MB x8
	//        B A20,21,22 -> 1MB x8
	// $5003: A16,15 -> 32KB x4
	all := prg*16*1024 - 1
	A2423 := uint8((all >> 23) & 0b11)
	A222120 := uint8((all >> 20) & 0b111)
	A191817 := uint8((all >> 17) & 0b111)
	A1615 := uint8((all >> 15) & 0b11)

	// fmt.Fprintf(os.Stderr, "\nA24-23: %02b, A22-20: %03b, A19-17: %03b, A16-15: %02b\n", A2423, A222120, A191817, A1615)

	//  7  bit  0
	//  ---- ----
	//  ABCC DEEE
	//  |||| ||||
	//  |||| |+++-- PRG offset (PRG A19, A18, A17)
	//  |||| +----- Alternate CHR A17
	//  ||++------- PRG offset (PRG A24, A23)
	//  ||++------- CHR offset (CHR A19, A18) on CHR-ROM-having boards
	//  |+--------- PRG mask (PRG A17 from 0: MMC3; 1: offset)
	//  +---------- CHR mask (CHR A17 from 0: MMC3; 1: alternate)
	reg5000 := uint8(0b0100_0000)

	//  7  bit  0
	//  ---- ----
	//  GHIL JKKx
	//  |||| |||
	//  |||| +++--- PRG offset (in order: A20, A21, A22)
	//  |||+------- GNROM mode bank PRG size (0: 32 KiB bank, PRG A14=CPU A14; 1: 16 KiB bank, PRG A14=offset A14)
	//  ||+-------- PRG mask (PRG A20 from 0: offset; 1: MMC3)
	//  |+--------- PRG mask (PRG A19 from 0: offset; 1: MMC3)
	//  +---------- PRG mask (PRG A18 from 0: MMC3; 1: offset)
	reg5001 := uint8(0b1000_0000)

	//  7  bit  0
	//  ---- ----
	//  NPZP QQRS
	//  |||| |||+-- SC1 bit (see below)
	//  |||| +++--- PRG offset for GNROM mode (PRG A16, A15, A14)
	//  |||+------- 1: GNROM mode; 0: MMC3 mode
	//  ||||         (1: PRG A16...13 from QQ, L, R, CPU A14, A13 + CHR A16...10 from MMMM, PPU A12...10;
	//  ||||          0: PRG A16...13 from MMC3 + CHR A16...A10 from MMC3 )
	//  ||+-------- 1: Also enable PRG RAM in $5000-$5FFF
	//  |+-+------- Banking mode
	//  |+--------- "Weird MMC3 mode"
	//  +---------- Lockout (prevent further writes to all registers but the one at $xxx2, only works in MMC3 mode)
	reg5003 := uint8(0b0001_0000)

	// address of registers
	reg0 := "2f"
	reg1 := "30"
	reg3 := "32"

	// ext. ROM select
	setReg("4000", 0, s)

	// bank select
	// Max 32MB: l(4) x k(8) x j(8) x i(4) x 32KB
	fmt.Fprintln(os.Stderr, "")
	for l := uint8(0); l <= A2423; l++ {
		r00 := reg5000 | (l << 4)
		for k := uint8(0); k <= A222120; k++ {
			// k: A22, A21, A20 -> A20, A21, A22
			kk := ((k & 0b001) << 2) | (k & 0b010) | ((k & 0b100) >> 2)
			r1 := reg5001 | (kk << 1)
			setReg(reg1, r1, s)
			for j := uint8(0); j <= A191817; j++ {
				r0 := r00 | j
				setReg(reg0, r0, s)
				for i := uint8(0); i <= A1615; i++ {
					r3 := reg5003 | (i << 2)
					setReg(reg3, r3, s)

					fmt.Fprintf(os.Stderr, "regs $5001=%02X, $5000=%02X, $5003=%02X\n", r1, r0, r3)
					kickoffDump32K(f, s)
				}
			}
		}
	}

	return nil
}

func writeSmd133PRG(s io.ReadWriter, f io.ReadSeeker, prg int) (err error) {
	// skip: 128KB xN
	l0 := uint8(0)
	k0 := uint8(0)
	j0 := uint8(0)
	if false {
		begin := (prg >> 3) - 1
		l0 = uint8((begin & 0b11000000) >> 6)
		k0 = uint8((begin & 0b00111000) >> 3)
		j0 = uint8((begin & 0b00000111))
		f.Seek(int64(begin)*128*1024, io.SeekCurrent)
	}

	// outer bank select
	//
	// $5000: A24,23 -> 8MB x4, A19,18,17 -> 128KB x8
	// $5001: A A20,22,21 -> 1MB x8
	//        B A20,21,22 -> 1MB x8
	all := prg*16*1024 - 1
	A2423 := uint8((all >> 23) & 0b11)
	A222120 := uint8((all >> 20) & 0b111)
	A191817 := uint8((all >> 17) & 0b111)

	// fmt.Fprintf(os.Stderr, "\nA24-23: %02b, A22-20: %03b, A19-17: %03b\n", A2423, A222120, A191817)

	//  7  bit  0
	//  ---- ----
	//  ABCC DEEE
	//  |||| ||||
	//  |||| |+++-- PRG offset (PRG A19, A18, A17)
	//  |||| +----- Alternate CHR A17
	//  ||++------- PRG offset (PRG A24, A23)
	//  ||++------- CHR offset (CHR A19, A18) on CHR-ROM-having boards
	//  |+--------- PRG mask (PRG A17 from 0: MMC3; 1: offset)
	//  +---------- CHR mask (CHR A17 from 0: MMC3; 1: alternate)
	reg5000 := uint8(0b0100_0000)

	//  7  bit  0
	//  ---- ----
	//  GHIL JKKx
	//  |||| |||
	//  |||| +++--- PRG offset (in order: A20, A21, A22)
	//  |||+------- GNROM mode bank PRG size (0: 32 KiB bank, PRG A14=CPU A14; 1: 16 KiB bank, PRG A14=offset A14)
	//  ||+-------- PRG mask (PRG A20 from 0: offset; 1: MMC3)
	//  |+--------- PRG mask (PRG A19 from 0: offset; 1: MMC3)
	//  +---------- PRG mask (PRG A18 from 0: MMC3; 1: offset)
	reg5001 := uint8(0b1000_0000)

	//  7  bit  0
	//  ---- ----
	//  NPZP QQRS
	//  |||| |||+-- SC1 bit (see below)
	//  |||| +++--- PRG offset for GNROM mode (PRG A16, A15, A14)
	//  |||+------- 1: GNROM mode; 0: MMC3 mode
	//  ||||         (1: PRG A16...13 from QQ, L, R, CPU A14, A13 + CHR A16...10 from MMMM, PPU A12...10;
	//  ||||          0: PRG A16...13 from MMC3 + CHR A16...A10 from MMC3 )
	//  ||+-------- 1: Also enable PRG RAM in $5000-$5FFF
	//  |+-+------- Banking mode
	//  |+--------- "Weird MMC3 mode"
	//  +---------- Lockout (prevent further writes to all registers but the one at $xxx2, only works in MMC3 mode)
	reg5003 := uint8(0b0000_0000)

	buf8K := make([]byte, 8*1024)

	// ext. ROM select
	setReg("4000", 0, s)

	// bank select
	// Max 32MB: l(4) x k(8) x j(8) x 128KB, MMC3 i(16) x 8KB
	fmt.Fprintln(os.Stderr, "")
	for l := l0; l <= A2423; l++ {
		r00 := reg5000 | (l << 4)
		for k := k0; k <= A222120; k++ {
			// k: A22, A21, A20 -> A20, A21, A22
			kk := ((k & 0b001) << 2) | (k & 0b010) | ((k & 0b100) >> 2)
			r1 := reg5001 | (kk << 1)
			setReg("5001", r1, s)
			for j := j0; j <= A191817; j++ {
				r0 := r00 | j
				setReg("5000", r0, s)
				r3 := reg5003
				setReg("5003", r3, s)
				fmt.Fprintf(os.Stderr, "regs $5001=%02X, $5000=%02X, $5003=%02X\n", r1, r0, r3)
				// MMC3: 16 x 8KB blocks @ $A000-$BFFF
				// - $8,9: MMC3 bank select regs
				// - $A,B: (MMC3 regs), flash regs & write window
				// - $C,D: (MMC3 regs)
				// - $E,F: (MMC3 regs)
				// erase 128KB block
				cur := (l << (23 - 16)) | (k << (20 - 16)) | (j << (17 - 16))
				fmt.Fprintf(os.Stderr, "%04X: ", cur)
				if true {
					fmt.Fprintf(os.Stderr, "e128")
					eraseFlash(s)
					fmt.Fprintf(os.Stderr, ".")
				}
				for i := uint8(0); i < 16; i++ {
					// prepare src data
					_, err := io.ReadFull(f, buf8K)
					if err != nil {
						return err
					}
					fmt.Fprintf(os.Stderr, "u")
					upload8K(s, buf8K)

					// write 8KB block
					fmt.Fprintf(os.Stderr, "p8")
					kickoffFlash(s, i)
					fmt.Fprintf(os.Stderr, ".")
				}
				fmt.Fprintln(os.Stderr, "")
			}
		}
	}

	return nil
}

func setReg(addr string, val uint8, s io.ReadWriter) {
	for {
		doRetry := false

		var wg sync.WaitGroup
		wg.Go(func() {
			buf := make([]byte, 1)
			for {
				_, err := io.ReadFull(s, buf)
				if err != nil {
					panic(err)
				}
				// fmt.Fprintf(os.Stderr, "%c", buf[0])
				if buf[0] == '.' {
					break
				}
			}
			for _, want := range []byte{'\r', '\n', ']', ' '} {
				_, err := io.ReadFull(s, buf)
				if err != nil {
					panic(err)
				}
				// fmt.Fprintf(os.Stderr, "%c", buf[0])
				if buf[0] != want {
					fmt.Fprintf(os.Stderr, "retry setReg!\n")
					doRetry = true
					return
				}
			}
		})

		valStr := fmt.Sprintf("%02X\r", val)

		time.Sleep(100 * time.Millisecond)
		s.Write([]byte("s" + addr + "\r"))
		time.Sleep(50 * time.Millisecond)
		s.Write([]byte(valStr))
		time.Sleep(50 * time.Millisecond)
		s.Write([]byte(".\r"))

		wg.Wait()

		if !doRetry {
			return
		}
	}
}

func kickoffDump32K(f io.Writer, s io.ReadWriter) {
	buf := make([]byte, 32*1024)

	for {
		doRetry := false

		var wg sync.WaitGroup
		wg.Go(func() {
			for {
				_, err := io.ReadFull(s, buf[0:1])
				if err != nil {
					panic(err)
				}
				fmt.Fprintf(os.Stderr, "%c", buf[0])
				if buf[0] == '\n' {
					break
				}
			}

			n, err := io.ReadFull(s, buf)
			if err != nil {
				panic(err)
			}
			if n != len(buf) {
				panic(fmt.Sprintf("expected %d bytes, got %d bytes", len(buf), n))
			}

			buf0 := make([]byte, 1)
			for _, want := range []byte{']', ' '} {
				_, err := io.ReadFull(s, buf0)
				if err != nil {
					panic(err)
				}
				fmt.Fprintf(os.Stderr, "%c", buf0[0])
				if buf0[0] != want {
					fmt.Fprintf(os.Stderr, "retry dump!\n")
					doRetry = true
					return
				}
			}
		})

		s.Write([]byte("ged7\r"))
		wg.Wait()

		if !doRetry {
			break
		}
	}

	f.Write(buf)
}

func kickoffDumpHex(start, end string, f io.Writer, s io.ReadWriter) {
	var wg sync.WaitGroup
	wg.Go(func() {
		scanner := bufio.NewScanner(s)
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) < 6+48 {
				continue
			}
			bytes := hexStringToBytes(line[6 : 6+48])
			f.Write(bytes)
			if line[0:3] == end[0:3] {
				break
			}
		}
	})

	cmd := fmt.Sprintf("d%s,%s\r", start, end)
	s.Write([]byte(cmd))

	wg.Wait()
}

func hexStringToBytes(hexStr string) []byte {
	bytes := []byte{}

	// " XX" x16
	for i := 0; i < len(hexStr); i += 3 {
		if i+2 > len(hexStr) {
			break
		}
		var byteValue byte
		fmt.Sscanf(hexStr[i+1:i+3], "%02X", &byteValue)
		bytes = append(bytes, byteValue)
	}

	return bytes
}

func eraseFlash(s io.ReadWriter) {
	setBank(s, 0)
	execCmd(s, "ge00")
}

func clear8K(s io.ReadWriter) {
	execCmd(s, "ge90")
}

func upload8KHex(s io.ReadWriter, buf8K []byte, addr uint32) {
	var buf bytes.Buffer

	mem := gohex.NewMemory()
	err := mem.AddBinary(addr, buf8K)
	if err != nil {
		panic(err)
	}
	err = mem.DumpIntelHex(&buf, 16)
	if err != nil {
		panic(err)
	}

	// debug
	/*
		{
			io.Copy(os.Stderr, &buf)
			panic("debug")
		}
	*/

	// go!
	s.Write([]byte("l\r"))
	time.Sleep(50 * time.Millisecond)

	var wg sync.WaitGroup
	wg.Go(func() {
		scanner := bufio.NewScanner(s)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "] " {
				break
			}
			// fmt.Fprintf(os.Stderr, "%s\n", line)
		}
	})

	isFirst := true
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		line := scanner.Text()
		if isFirst {
			isFirst = false
			continue
		}
		s.Write([]byte(line + "\r"))
		time.Sleep(50 * time.Millisecond)
	}

	wg.Wait()
}

func upload8K(s io.ReadWriter, buf8K []byte) {
	// go!
	s.Write([]byte("geb3\r"))
	time.Sleep(50 * time.Millisecond)

	var wg sync.WaitGroup
	wg.Go(func() {
		scanner := bufio.NewScanner(s)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "] " {
				break
			}
			// fmt.Fprintf(os.Stderr, "%s\n", line)
		}
	})

	/*
		for _, b := range buf8K {
			s.Write([]byte{b})
			time.Sleep(100 * time.Microsecond)
		}
	*/
	s.Write(buf8K)
	time.Sleep(50 * time.Millisecond)
	s.Write([]byte("\r"))

	wg.Wait()
}

func kickoffFlash(s io.ReadWriter, bank uint8) {
	setBank(s, bank)
	execCmd(s, "ge3c")
}

func setBank(s io.ReadWriter, bank uint8) {
	setReg("2e", bank, s)
}

func execCmd(s io.ReadWriter, cmd string) {
	var wg sync.WaitGroup
	wg.Go(func() {
		scanner := bufio.NewScanner(s)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "] " {
				break
			}
			// fmt.Fprintf(os.Stderr, "%s\n", line)
		}
	})

	s.Write([]byte(cmd))
	s.Write([]byte("\r"))
	time.Sleep(50 * time.Millisecond)
	s.Write([]byte("\r"))

	wg.Wait()
}
