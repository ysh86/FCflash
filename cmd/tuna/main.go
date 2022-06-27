package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/tarm/serial"
)

const PACKET_SIZE = 0x400

type Request uint8

const (
	REQ_ECHO Request = iota
	REQ_PHI2_INIT

	REQ_CPU_READ_6502
	REQ_CPU_READ
	REQ_CPU_WRITE_6502
	REQ_CPU_WRITE_FLASH

	REQ_PPU_READ
	REQ_PPU_WRITE

	REQ_CPU_WRITE_EEP = 16
	REQ_PPU_WRITE_EEP = 17

	REQ_RAW_READ        = 32
	REQ_RAW_ERASE_FLASH = 33
	REQ_RAW_WRITE_FLASH = 34

	REQ_CPU_WRITE_5BITS_6502 = 35
)

type Index uint16

const (
	INDEX_IMPLIED Index = iota
	INDEX_CPU
	INDEX_PPU
	INDEX_BOTH
)

type Message struct {
	_reserverd uint8
	Request    Request
	Value      uint16
	index      Index
	Length     uint16
}

func main() {
	// args
	var (
		com      int
		baud     int
		mapper   int
		prg      int
		chr      int
		mirror   int
		raw      bool
		eeprom   bool
		flash    bool
		fileName string
	)
	flag.IntVar(&com, "com", 5, "com port")
	flag.IntVar(&baud, "baud", 115200, "baud rate")
	flag.IntVar(&mapper, "mapper", 1, "mapper 0:NROM, 1:SxROM, 4:TxROM")
	flag.IntVar(&prg, "prg", 16, "Size of PRG ROM in 16KB units")
	flag.IntVar(&chr, "chr", 0, "Size of CHR ROM in 8KB units (Value 0 means the board uses CHR RAM)")
	flag.IntVar(&mirror, "mirror", 2, "0:H, 1:V, 2:battery-backed PRG RAM")
	flag.BoolVar(&raw, "raw", false, "raw access to ROM/RAM/EEPROM/Flash ICs")
	flag.BoolVar(&eeprom, "eeprom", false, "write NROM EEPROM")
	flag.BoolVar(&flash, "flash", false, "write Flash")
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

	// start
	buf := make([]uint8, PACKET_SIZE)

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
		if mapper != 1 {
			panic(fmt.Errorf("write mapper:%d Flash is NOT implemented", mapper))
		}
		if chr > 0 {
			panic(errors.New("write CHR Flash is NOT implemented"))
		}
		fmt.Printf("write SxROM: prg:%d, chr:%d\n", prg, chr)
		fmt.Println("----")
		fmt.Println("ready?")
		io.ReadAtLeast(os.Stdin, buf[0:1], 1)
		err := writeSxromPRG(s, fileName, prg, buf)
		if err != nil {
			panic(err)
		}
		fmt.Println("done")
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
	buf[4] = uint8(prg)
	buf[5] = uint8(chr)
	buf[6] = uint8((mapper << 4) | mirror)
	if !raw {
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
		if mapper == 0 {
			err = dumpNromPRG(f, s, prg, buf)
		} else if mapper == 1 {
			err = dumpSxromPRG(f, s, prg, buf)
		} else {
			err = dumpTxromPRG(f, s, prg, buf)
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
		if mapper == 0 {
			err = dumpNromCHR(f, s, chr, buf)
		} else if mapper == 1 {
			panic(fmt.Errorf("mapper:%d CHR is NOT implemented", mapper))
		} else {
			err = dumpTxromCHR(f, s, chr, buf)
		}
		if err != nil {
			panic(err)
		}
		fmt.Println(" done")
	} else {
		fmt.Println("CHR: skip")
	}
}
