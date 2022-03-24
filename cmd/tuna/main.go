package main

import (
	"encoding/binary"
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
		eeprom   bool
		fileName string
	)
	flag.IntVar(&com, "com", 5, "com port")
	flag.IntVar(&baud, "baud", 115200, "baud rate")
	flag.IntVar(&mapper, "mapper", 4, "mapper 0:NROM, 4:TxROM")
	flag.IntVar(&prg, "prg", 32, "Size of PRG ROM in 16KB units")
	flag.IntVar(&chr, "chr", 32, "Size of CHR ROM in 8KB units (Value 0 means the board uses CHR RAM)")
	flag.IntVar(&mirror, "mirror", 0, "0:H, 1:V, 2:battery-backed PRG RAM")
	flag.BoolVar(&eeprom, "eeprom", false, "write NROM EEPROM")
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
		f, err := os.Open(fileName)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		// skip header
		_, err = f.Seek(16, io.SeekStart)
		if err != nil {
			panic(err)
		}

		// PRG
		for i := 0; i < 32*1024; i += PACKET_SIZE {
			buf[0] = 0 // _reserverd
			buf[1] = uint8(REQ_CPU_WRITE_EEP)
			binary.LittleEndian.PutUint16(buf[2:4], 0x8000|uint16(i))      // Value
			binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
			binary.LittleEndian.PutUint16(buf[6:8], PACKET_SIZE)           // Length
			_, err = s.Write(buf[0:8])
			if err != nil {
				panic(err)
			}

			_, err = io.ReadFull(f, buf)
			if err != nil {
				panic(err)
			}

			_, err = s.Write(buf)
			if err != nil {
				panic(err)
			}
		}
		// CHR
		for i := 0; i < 8*1024; i += PACKET_SIZE {
			buf[0] = 0 // _reserverd
			buf[1] = uint8(REQ_PPU_WRITE_EEP)
			binary.LittleEndian.PutUint16(buf[2:4], uint16(i))             // Value
			binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
			binary.LittleEndian.PutUint16(buf[6:8], PACKET_SIZE)           // Length
			_, err = s.Write(buf[0:8])
			if err != nil {
				panic(err)
			}

			_, err = io.ReadFull(f, buf)
			if err != nil {
				panic(err)
			}

			_, err = s.Write(buf)
			if err != nil {
				panic(err)
			}
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
	buf[4] = uint8(prg)
	buf[5] = uint8(chr)
	buf[6] = uint8((mapper << 4) | mirror)
	_, err = f.Write(buf[0:16])
	if err != nil {
		panic(err)
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

	// PRG
	if mapper == 0 {
		err = dumpNromPRG(f, s, prg, buf)
	} else {
		err = dumpTxromPRG(f, s, prg, buf)
	}
	if err != nil {
		panic(err)
	}
	fmt.Println("done PRG")

	// CHR
	if mapper == 0 {
		err = dumpNromCHR(f, s, chr, buf)
	} else {
		err = dumpTxromCHR(f, s, chr, buf)
	}
	if err != nil {
		panic(err)
	}
	fmt.Println("done CHR")
}
