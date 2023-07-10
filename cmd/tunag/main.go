package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
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

	REQ_RAW_WRITE = 64
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
		ram      bool
		fileName string
		ramName  string
	)
	flag.IntVar(&com, "com", 5, "com port")
	flag.IntVar(&baud, "baud", 115200, "baud rate")
	flag.BoolVar(&ram, "ram", false, "write RAM in cartridge")
	flag.Parse()
	if ram {
		args := flag.Args()
		if len(args) < 1 {
			panic(errors.New("no file name"))
		}
		ramName = args[0]
	}

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

	// header
	_, err = readFull(s, 0, buf)
	if err != nil {
		panic(err)
	}
	title, cgb, cartType, romSize, ramSize := parseHeader(buf)
	if !ram {
		if cgb == 0xc0 {
			fileName = title + ".gbc"
		} else {
			fileName = title + ".gb"
		}
		ramName = title + ".sav"
	}

	numBanks := 2 << romSize // 16[KB/bank]
	fmt.Println(fileName, ramName, cartType, numBanks, ramSize)
}

func readFull(s io.ReadWriter, offset uint32, buf []byte) (n int, err error) {
	n = len(buf)
	if n > PACKET_SIZE {
		return 0, io.ErrShortWrite
	}

	buf[0] = 0 // _reserverd
	buf[1] = uint8(REQ_RAW_READ)
	binary.LittleEndian.PutUint16(buf[2:4], uint16(offset>>8))     // Value
	binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(buf[6:8], uint16(n))             // Length
	_, err = s.Write(buf[0:8])
	if err != nil {
		return 0, err
	}

	_, err = io.ReadFull(s, buf)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func parseHeader(buf []byte) (title string, cgb, cartType, romSize, ramSize byte) {
	begin := 0x0134
	header := buf[begin:0x0150]

	title = string(header[0:(0x0143 - begin)])
	cgb = header[0x0143-begin]
	cartType = header[0x0147-begin]
	romSize = header[0x0148-begin]
	ramSize = header[0x0149-begin]

	fmt.Printf("title:   %s\n", title)
	fmt.Printf("isCGB:   %02x\n", header[0x0143-begin])
	fmt.Printf("licensee:%02x%02x\n", header[0x0144-begin], header[0x0145-begin])
	fmt.Printf("isSGB:   %02x\n", header[0x0146-begin])
	fmt.Printf("type:    %02x\n", cartType)
	fmt.Printf("ROMsize: %02x\n", romSize)
	fmt.Printf("RAMsize: %02x\n", ramSize)
	fmt.Printf("dest:    %02x\n", header[0x014A-begin])
	fmt.Printf("old:     %02x\n", header[0x014B-begin])
	fmt.Printf("version: %02x\n", header[0x014C-begin])
	fmt.Printf("complement: %02x\n", header[0x014D-begin])
	fmt.Printf("checksum:   %02x%02x\n", header[0x014E-begin], header[0x014F-begin])
	fmt.Printf("\n")

	return
}
