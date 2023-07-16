package main

import (
	"bytes"
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
		flash    bool
		fileName string
		ramName  string
	)
	flag.IntVar(&com, "com", 5, "com port")
	flag.IntVar(&baud, "baud", 115200, "baud rate")
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

	// start
	gb := FCflash.NewGB(s)

	if flash {
		err := writeFlash(gb)
		if err != nil {
			panic(err)
		}
		return
	}

	// header
	err = gb.ReadFull(0)
	if err != nil {
		panic(err)
	}
	title, cgb, cartType, romSize, ramSize := parseHeader(gb.Buf)

	// GBM cart
	if title == FCflash.GBM_MENU_TITLE {
		err := gbm(gb)
		if err != nil {
			panic(err)
		}
		return
	}

	// normal cart
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

	if ramSize != 0 || cartType == 6 {
		// dump RAM
		w, err := os.Create(ramName)
		if err != nil {
			panic(err)
		}
		defer w.Close()
		n, err := gb.DumpRAM(w, cartType, ramSize)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s: %d\n", ramName, n)

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

func parseHeader(buf []byte) (title string, cgb, cartType, romSize, ramSize byte) {
	begin := 0x0134
	header := buf[begin:0x0150]

	t := header[0:(0x0143 - begin)]
	i := bytes.IndexByte(t, 0x00)
	if i != -1 {
		t = t[:i]
	}

	title = string(bytes.TrimSpace(t))
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

func writeFlash(gb *FCflash.GB) error {
	manufacturerCode, deviceCode, err := gb.DetectFlash()
	if err != nil {
		return err
	}
	fmt.Printf("flash: manufacturerCode=%02x, deviceCode=%02x\n", manufacturerCode, deviceCode)

	return nil
}

func gbm(gbm *FCflash.GB) error {
	err := gbm.DetectGBM()
	if err != nil {
		return err
	}

	// Map info
	fmap, err := os.Create("GBM.map")
	if err != nil {
		return err
	}
	defer fmap.Close()
	err = gbm.ReadMappingGBM(fmap)
	if err != nil {
		return err
	}

	// TODO: print Map info
	// https://github.com/lesserkuma/FlashGBX/blob/master/FlashGBX/GBMemory.py
	// https://forums.nesdev.org/viewtopic.php?f=12&t=11453&start=135#p161062
	//
	// keys = ["mapper_params",
	//         "f_size", "b_size",
	//         "game_code", "title", "timestamp", "kiosk_id",
	//         "write_count",
	//         "cart_id",
	//         "padding", "unknown"]
	// 24: 3byte * 8本
	//  H: FFFF uint16
	//  H: FFFF uint16
	// 12: FF
	// 44: FF
	// 18: FF
	//  8: FF
	//  H: write count uint16
	//  8: cart ID
	//  6: FF
	//  H: 0000 uint16
	//
	// 00 23 3f: 000 000 00 0 23 3f
	// 00 00 01:
	// a5 00 01: 101 010 01 0 00 01
	//
	// https://forums.nesdev.org/viewtopic.php?p=163123&sid=ac1abb99aa8b0c98ad50e945e26f33b9#p163123
	// 1C000h + 200h * 8 本
	// ...
	//
	mapping := make([]byte, 128)
	copy(mapping, gbm.Buf[0:128])
	fmt.Printf("mapping: %+v\n", mapping)

	// Map entire ROM
	err = gbm.MapEntireROM()
	if err != nil {
		return err
	}

	// dump ROM
	fileName := "GBMMBC4.gbc"
	w, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer w.Close()
	checkSum, err := gbm.DumpROM(w, FCflash.GBM_ENTIRE_MBC, FCflash.GBM_ENTIRE_ROM_SIZE)
	if err != nil {
		return err
	}
	fmt.Printf("%s: %04x\n", fileName, checkSum&0xffff)

	// dump RAM
	{
		ramName := "GBMMBC4.sav"
		w, err := os.Create(ramName)
		if err != nil {
			return err
		}
		defer w.Close()
		n, err := gbm.DumpRAM(w, FCflash.GBM_ENTIRE_MBC, FCflash.GBM_ENTIRE_RAM_SIZE)
		if err != nil {
			return err
		}
		fmt.Printf("%s: %d\n", ramName, n)
	}

	return nil
}
