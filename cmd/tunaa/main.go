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

const PACKET_SIZE = 0x10000

type Request uint32

const (
	REQ_READ16 Request = iota
	REQ_WRITE16
	REQ_WRITE16_RND
	REQ_READ8_CS2
	REQ_WRITE8_CS2
	REQ_WRITE8_CS2_RND
)

type Message struct {
	Request  Request
	Value    uint32
	Length   uint32
	reserved uint32
}

func main() {
	// args
	var (
		com      int
		baud     int
		ram      bool
		flash    bool
		fileName string
		ramName  string
	)
	flag.IntVar(&com, "com", 7, "com port")
	flag.IntVar(&baud, "baud", 500000, "baud rate")
	flag.BoolVar(&ram, "ram", false, "write RAM in cartridge")
	flag.BoolVar(&flash, "flash", false, "write Flash")
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

	// flash
	if flash {
		return
	}

	// normal cart
	if !ram {
		fileName = "test" + ".gba"
		ramName = "test" + ".sav"
	}

	// dump ROM
	w, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	defer w.Close()
	buf := make([]byte, PACKET_SIZE)
	romSize := uint32(4 * 1024 * 1024)
	for addr25 := uint32(0); addr25 < romSize; addr25 += PACKET_SIZE {
		fmt.Print(".")

		binary.LittleEndian.PutUint32(buf[0:4], uint32(REQ_READ16)) // Request
		binary.LittleEndian.PutUint32(buf[4:8], (addr25 >> 1))      // Value
		binary.LittleEndian.PutUint32(buf[8:12], PACKET_SIZE)       // Length
		binary.LittleEndian.PutUint32(buf[12:16], 0)                // reserved
		_, err := s.Write(buf[0:16])
		if err != nil {
			panic(err)
		}

		fmt.Print("+")

		n, err := io.ReadFull(s, buf[0:PACKET_SIZE])
		if err != nil {
			panic(err)
		}

		w.Write(buf[0:PACKET_SIZE])

		fmt.Printf("*: %d, ", n)
	}
	fmt.Printf("%s\n", fileName)

	if ram {
		// dump RAM
		w, err := os.Create(ramName)
		if err != nil {
			panic(err)
		}
		defer w.Close()
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s\n", ramName)

		// clear RAM
		/*
			{
				n, err := gb.ClearRAM(cartType, ramSize)
				if err != nil {
					panic(err)
				}
				fmt.Printf("clear RAM: %d\n", n)
			}
		*/
	}
}
