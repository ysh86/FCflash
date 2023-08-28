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
		com      int
		baud     int
		ram      bool
		flash    bool
		all      bool
		fileName string
		ramName  string
	)
	flag.IntVar(&com, "com", 5, "com port")
	flag.IntVar(&baud, "baud", 115200, "baud rate")
	flag.BoolVar(&ram, "ram", false, "write RAM in cartridge")
	flag.BoolVar(&flash, "flash", false, "write Flash")
	flag.BoolVar(&all, "a", false, "dump both ROM & RAM")
	flag.Parse()
	if ram {
		args := flag.Args()
		if len(args) < 1 {
			panic(errors.New("no file name"))
		}
		ramName = args[0]
	}
	if flash {
		args := flag.Args()
		if len(args) < 1 {
			panic(errors.New("no file name"))
		}
		fileName = args[0]
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

	// ram
	if ram {
		panic("not implemented")
	}
	// flash
	if flash {
		err := writeFlash(gb, fileName)
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
	title, cgb, cartType, romSize, ramSize, ok := FCflash.ParseHeader(gb.Buf)
	if !ok {
		panic("invalid header")
	}

	// GBM cart
	if title == FCflash.GBM_MENU_TITLE {
		err := gbm(gb)
		if err != nil {
			panic(err)
		}
		return
	}

	// normal cart
	if cgb == 0xc0 {
		fileName = title + ".gbc"
	} else {
		fileName = title + ".gb"
	}
	ramName = title + ".sav"

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
	if all && (ramSize != 0 || cartType == 6) {
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

func writeFlash(gb *FCflash.GB, fileName string) error {
	device, err := gb.IsSupportedFlash()
	if err != nil {
		return err
	}
	fmt.Printf("flash: manufacturerCode=%02x, deviceCode=%02x\n", device>>8, device&0xff)

	f, err := os.Open(fileName)
	if err != nil {
		return err
	}

	fmt.Printf("Bank:")
	buf := make([]byte, 16*1024) // = bank size
	bank := 0
	addr := 0
loop:
	for {
		_, err = io.ReadFull(f, buf)
		if err != nil {
			break
		}

		fmt.Printf(" %02x", bank)
		gb.WriteRegByte(0x2100, bank)
		for i := 0; i < len(buf); i += FCflash.PACKET_SIZE {
			err = gb.WriteFlash(device, addr, buf[i:i+FCflash.PACKET_SIZE])
			if err != nil {
				break loop
			}
			addr += FCflash.PACKET_SIZE
		}
		bank += 1
	}
	fmt.Println("")
	if err == io.EOF {
		return nil
	}

	return err
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
	mapping, err := gbm.ReadMappingGBM(fmap)
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
	fmt.Printf("mapping:\n")
	for i, m := range mapping {
		if i&15 == 0 {
			fmt.Printf("%08x", i)
		}
		fmt.Printf(" %02x", m)
		if i&15 == 15 {
			fmt.Print("\n")
		}
	}

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
