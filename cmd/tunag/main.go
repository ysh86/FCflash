package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/tarm/serial"
	"github.com/ysh86/FCflash"
)

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
	gb := FCflash.NewGB(s)

	// header
	err = gb.ReadFull(0)
	if err != nil {
		panic(err)
	}
	title, cgb, cartType, romSize, ramSize := parseHeader(gb.Buf)
	if !ram {
		if cgb == 0xc0 {
			fileName = title + ".gbc"
		} else {
			fileName = title + ".gb"
		}
		ramName = title + ".sav"
	}

	// dump ROM
	w, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	defer w.Close()
	checkSum, err := gb.DumpROM(w, cartType, romSize)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s: %04x\n", fileName, checkSum&0xffff)

	// dump RAM
	fmt.Printf("%s: %d\n", ramName, ramSize)
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
