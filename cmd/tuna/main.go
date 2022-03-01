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
		com    int
		baud   int
		mapper int
		prg    int
		chr    int
		mirror int
		output string
	)
	flag.IntVar(&com, "com", 5, "com port")
	flag.IntVar(&baud, "baud", 115200, "baud rate")
	flag.IntVar(&mapper, "mapper", 4, "mapper 0:NROM, 4:TxROM")
	flag.IntVar(&prg, "prg", 32, "Size of PRG ROM in 16KB units")
	flag.IntVar(&chr, "chr", 32, "Size of CHR ROM in 8KB units (Value 0 means the board uses CHR RAM)")
	flag.IntVar(&mirror, "mirror", 0, "0:H, 1:V, 2:battery-backed PRG RAM")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		panic(errors.New("no output file"))
	}
	output = args[0]

	// COM
	comport := "/dev/ttyS" + strconv.Itoa(com)
	c := &serial.Config{Name: comport, Baud: baud}
	s, err := serial.OpenPort(c)
	if err != nil {
		panic(err)
	}
	defer s.Close()

	// start
	f, err := os.Create(output)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	buf := make([]uint8, PACKET_SIZE)

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
	if mapper == 0 {
		fmt.Println("NROM: connect PPU /RD(17) to D3/PD0")
	} else {
		fmt.Println("TxROM: connect PPU /RD(17) to D2/PD1")
	}
	fmt.Println("----")
	fmt.Println("connect PPU A13(56) to HIGH")
	fmt.Println("ready?")
	io.ReadAtLeast(os.Stdin, buf[0:1], 1)

	// PRG
	for i := 0; i < prg*16*1024; i += PACKET_SIZE {
		buf[0] = 0 // _reserverd
		buf[1] = uint8(REQ_CPU_READ)
		binary.LittleEndian.PutUint16(buf[2:4], 0x8000|uint16(i))      // Value
		binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
		binary.LittleEndian.PutUint16(buf[6:8], PACKET_SIZE)           // Length
		_, err = s.Write(buf[0:8])
		if err != nil {
			panic(err)
		}

		_, err = io.ReadFull(s, buf)
		if err != nil {
			panic(err)
		}

		_, err = f.Write(buf)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("done PRG")

	fmt.Println("----")
	fmt.Println("connect PPU A13(56) to LOW")
	fmt.Println("ready?")
	io.ReadAtLeast(os.Stdin, buf[0:1], 1)

	// CHR
	for i := 0; i < chr*8*1024; i += PACKET_SIZE {
		buf[0] = 0 // _reserverd
		buf[1] = uint8(REQ_PPU_READ)
		binary.LittleEndian.PutUint16(buf[2:4], 0x2000|uint16(i))      // Value: 0x2000 special flag for the Tuna board :)
		binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
		binary.LittleEndian.PutUint16(buf[6:8], PACKET_SIZE)           // Length
		_, err = s.Write(buf[0:8])
		if err != nil {
			panic(err)
		}

		_, err = io.ReadFull(s, buf)
		if err != nil {
			panic(err)
		}

		_, err = f.Write(buf)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("done CHR")
}
