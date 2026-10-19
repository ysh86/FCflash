package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/tarm/serial"
	"github.com/ysh86/FCflash"
)

func main() {
	// args
	var (
		com       int
		baud      int
		mapperStr string
		mapper    int
		prg       int
		chr       int
		mirror    int
		raw       bool
		eeprom    bool
		flash     bool
		noHeader  bool
		fileName  string
	)
	flag.IntVar(&com, "com", 4, "com port")
	flag.IntVar(&baud, "baud", 38400, "baud rate")
	flag.StringVar(&mapperStr, "mapper", "1", "mapper 0:NROM, 1:SxROM, 4:TxROM, 268a: SMD133A, 268b: SMD133, 446: AA6109")
	flag.IntVar(&prg, "prg", 16, "Size of PRG ROM in 16KB units")
	flag.IntVar(&chr, "chr", 0, "Size of CHR ROM in 8KB units (Value 0 means the board uses CHR RAM)")
	flag.IntVar(&mirror, "mirror", 2, "0:H, 1:V, 2:battery-backed PRG RAM")
	flag.BoolVar(&raw, "raw", false, "raw access to ROM/RAM/EEPROM/Flash ICs")
	flag.BoolVar(&eeprom, "eeprom", false, "write NROM EEPROM")
	flag.BoolVar(&flash, "flash", false, "write Flash")
	flag.BoolVar(&noHeader, "noheader", false, "skip writing iNES header")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		panic(errors.New("no file name"))
	}
	fileName = args[0]

	// COM
	comport := "/dev/ttyS" + strconv.Itoa(com)
	c := &serial.Config{Name: comport, Baud: baud}
	s, err := serial.OpenPort(c)
	if err != nil {
		panic(err)
	}
	defer s.Close()

	// mapper
	mapper, err = strconv.Atoi(mapperStr)
	if err != nil {
		switch mapperStr {
		case "268a":
			mapper = 268
		case "268b":
			mapper = 268 + 1
		default:
			panic(err)
		}
		err = nil
	}

	// start
	buf := make([]uint8, FCflash.PACKET_SIZE)

	// EEPROM
	if eeprom {
		if raw {
			// RAW EEPROM
			panic(errors.New("write RAW EEPROM is NOT implemented"))
		}
		if mapper != 0 {
			panic(fmt.Errorf("write mapper:%d EEPROM is NOT implemented", mapper))
		}
		fmt.Printf("write NROM EEPROM: prg:%d, chr:%d\n", prg, chr)
		fmt.Println("----")
		fmt.Println("ready?")
		io.ReadAtLeast(os.Stdin, buf[0:1], 1)
		err := writeEEPROM(s, fileName, prg, chr, buf)
		if err != nil {
			panic(err)
		}
		fmt.Println("done")
		return
	}

	// Flash
	if flash {
		if raw {
			fmt.Printf("write RAW Flash: prg:%d, chr:%d\n", prg, chr)
			fmt.Println("----")
			fmt.Println("ready?")
			io.ReadAtLeast(os.Stdin, buf[0:1], 1)
			size := prg*16*1024 + chr*8*1024
			err := writeFlash(s, fileName, size, buf)
			if err != nil {
				panic(err)
			}
			fmt.Println("done")
			return
		}
		// Flash with mapper
		switch mapper {
		case 1:
			// Handle SxROM mapper
			if chr > 0 {
				panic(errors.New("write CHR Flash is NOT implemented"))
			}

			fmt.Printf("write SxROM: prg:%d, chr:%d, src:%s\n", prg, chr, fileName)
			fmt.Println("----")
			fmt.Println("ready?")
			io.ReadAtLeast(os.Stdin, buf[0:1], 1)

			err := writeSxromPRG(s, fileName, prg, buf)
			if err != nil {
				panic(err)
			}
			fmt.Println("done")
		case 268, 269:
			// Handle SMD133A/B mappers
			if chr > 0 {
				panic(errors.New("write CHR Flash is NOT implemented"))
			}
			f, err := os.Open(fileName)
			if err != nil {
				panic(err)
			}
			defer f.Close()
			// skip header
			if !noHeader {
				_, err = f.Seek(16, io.SeekStart)
				if err != nil {
					panic(err)
				}
			}

			fmt.Printf("write SMD133A/B: prg:%d, chr:%d, src:%s\n", prg, chr, fileName)
			fmt.Println("----")
			fmt.Println("ready?")
			io.ReadAtLeast(os.Stdin, buf[0:1], 1)

			fmt.Print("PRG: . . .")
			if mapper == 268 {
				err = writeSmd133aPRG(s, f, prg)
			} else {
				err = writeSmd133PRG(s, f, prg)
			}
			if err != nil {
				panic(err)
			}
			fmt.Println("done")
		default:
			panic(fmt.Errorf("write mapper:%d Flash is NOT implemented", mapper))
		}
		return
	}

	// dump
	f, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// header
	buf[0] = 'N'
	buf[1] = 'E'
	buf[2] = 'S'
	buf[3] = 0x1a
	buf[4] = uint8(prg & 0xff)
	buf[5] = uint8(chr & 0xff)
	switch mapperStr {
	case "268a":
		m := 268
		sub := 1
		buf[6] = uint8(((m & 0x0f) << 4) | mirror)
		buf[7] = uint8((m & 0xf0) | 8)
		buf[8] = uint8((sub << 4) | ((m & 0xf00) >> 8))
		buf[9] = uint8((prg & 0xf00) >> 8)
		buf[10] = 0x70 // PRG-NVRAM/EEPROM 8KB: 64 << 7
		buf[11] = 0x0c // CHR-RAM 256KB: 64 << 12
	case "268b":
		m := 268
		sub := 3
		buf[6] = uint8(((m & 0x0f) << 4) | mirror)
		buf[7] = uint8((m & 0xf0) | 8)
		buf[8] = uint8((sub << 4) | ((m & 0xf00) >> 8))
		buf[9] = uint8((prg & 0xf00) >> 8)
		buf[10] = 0x70 // PRG-NVRAM/EEPROM 8KB: 64 << 7
		buf[11] = 0x0c // CHR-RAM 256KB: 64 << 12
	case "446":
		m := 446
		sub := 1
		buf[6] = uint8(((m & 0x0f) << 4) | mirror)
		buf[7] = uint8((m & 0xf0) | 8)
		buf[8] = uint8((sub << 4) | ((m & 0xf00) >> 8))
		buf[9] = uint8((prg & 0xf00) >> 8)
		buf[10] = 0x00 // no PRG-NVRAM/EEPROM
		buf[11] = 0x0c // CHR-RAM 256KB: 64 << 12
	default:
		m := mapper
		buf[6] = uint8(((m & 0x0f) << 4) | mirror)
		buf[7] = uint8(m & 0xf0)
	}
	if !raw && !noHeader {
		_, err = f.Write(buf[0:16])
		if err != nil {
			panic(err)
		}
	}

	// info
	fmt.Println("COM:", comport, "@", baud)
	fmt.Println("Mapper:", mapper)
	fmt.Println("PRG ROM:", prg*16, "[KB]")
	fmt.Println("CHR ROM:", chr*8, "[KB]")
	var m string
	if mirror&1 == 0 {
		if mirror&2 == 0 {
			m = "H"
		} else {
			m = "H battery"
		}
	} else {
		if mirror&2 == 0 {
			m = "V"
		} else {
			m = "V battery"
		}
	}
	fmt.Println("Mirroring:", m)
	fmt.Println("----")
	fmt.Println("ready?")
	io.ReadAtLeast(os.Stdin, buf[0:1], 1)

	// raw mode
	if raw {
		fmt.Print("RAW: . . .")
		size := prg*16*1024 + chr*8*1024
		err = dumpRAW(f, s, size, buf)
		if err != nil {
			panic(err)
		}
		fmt.Println(" done")
		return
	}

	// PRG
	if prg != 0 {
		fmt.Print("PRG: . . .")
		switch mapper {
		case 0:
			err = dumpNromPRG(f, s, prg, buf)
		case 1:
			err = dumpSxromPRG(f, s, prg, buf)
		case 4:
			err = dumpTxromPRG(f, s, prg, buf)
		case 268:
			err = dumpSmd133aPRG(f, s, prg)
		case 268 + 1:
			err = dumpSmd133PRG(f, s, prg)
		case 446:
			err = dumpAA6109PRG(f, s, prg)
		default:
			panic(fmt.Errorf("mapper:%d PRG is NOT implemented", mapper))
		}
		if err != nil {
			panic(err)
		}
		fmt.Println(" done")
	} else {
		fmt.Println("PRG: skip")
	}

	// CHR
	if chr != 0 {
		fmt.Print("CHR: . . .")
		switch mapper {
		case 0:
			err = dumpNromCHR(f, s, chr, buf)
		case 4:
			err = dumpTxromCHR(f, s, chr, buf)
		default:
			panic(fmt.Errorf("mapper:%d CHR is NOT implemented", mapper))
		}
		if err != nil {
			panic(err)
		}
		fmt.Println(" done")
	} else {
		fmt.Println("CHR: skip")
	}
}
