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
	REQ_READ16 Request = iota + 0
	REQ_WRITE16
	REQ_WRITE16_RND
	REQ_READ8_CS2
	REQ_WRITE8_CS2
	REQ_WRITE8_CS2_RND
)
const (
	REQ_CPU_READ Request = iota + 16
	REQ_CPU_WRITE
	REQ_PPU_READ
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
		fc8bit   bool
		fc7      bool
		fileName string
		ramName  string
	)
	flag.IntVar(&com, "com", 7, "com port")
	flag.IntVar(&baud, "baud", 500000, "baud rate")
	flag.BoolVar(&ram, "ram", false, "write RAM in cartridge")
	flag.BoolVar(&flash, "flash", false, "write Flash")
	flag.BoolVar(&fc8bit, "fc8bit", false, "dump FC 8BIT TxROM")
	flag.BoolVar(&fc7, "fc7", false, "dump FC mapper7 AxROM PRG")
	flag.Parse()
	if ram {
		args := flag.Args()
		if len(args) < 1 {
			panic(errors.New("no file name"))
		}
		ramName = args[0]
	}
	if fc8bit || fc7 {
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

	// flash
	if flash {
		return
	}

	// normal cart
	if fileName == "" && ramName == "" {
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

	// FC
	if fc8bit {
		err = dumpTxromPRG(w, s, 32, buf[:16*1024]) // 16KB * 32 = 512KB
		if err != nil {
			panic(err)
		}
		err = dumpTxromCHR(w, s, 32, buf[:2*1024]) // 8KB * 32 = 256KB
		if err != nil {
			panic(err)
		}
		return
	}
	if fc7 {
		err = dumpAxromPRG(w, s, 8, buf[:32*1024]) // 16KB * 8 = 128KB
		if err != nil {
			panic(err)
		}
		return
	}

	// GBA
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

func dumpTxromPRG(f io.Writer, s io.ReadWriter, prg int, buf []uint8) (err error) {
	packetSize := len(buf)

	banks := (prg * 16 * 1024) >> 13
	for bank := 0; bank < banks; bank += 2 {
		// MMC3: PRG ROM R6:$8000-$9FFF swappable
		bankSelect := uint32(0b00000110)
		binary.LittleEndian.PutUint32(buf[0:4], uint32(REQ_CPU_WRITE)) // Request
		binary.LittleEndian.PutUint32(buf[4:8], 0x8000)                // Value
		binary.LittleEndian.PutUint32(buf[8:12], bankSelect)           // Length
		binary.LittleEndian.PutUint32(buf[12:16], 0)                   // reserved
		_, err = s.Write(buf[0:16])
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(buf[0:4], uint32(REQ_CPU_WRITE)) // Request
		binary.LittleEndian.PutUint32(buf[4:8], 0x8001)                // Value
		binary.LittleEndian.PutUint32(buf[8:12], uint32(bank+0))       // Length
		binary.LittleEndian.PutUint32(buf[12:16], 0)                   // reserved
		_, err = s.Write(buf[0:16])
		if err != nil {
			return err
		}
		// MMC3: PRG ROM R7:$A000-$BFFF swappable
		bankSelect = uint32(0b00000111)
		binary.LittleEndian.PutUint32(buf[0:4], uint32(REQ_CPU_WRITE)) // Request
		binary.LittleEndian.PutUint32(buf[4:8], 0x8000)                // Value
		binary.LittleEndian.PutUint32(buf[8:12], bankSelect)           // Length
		binary.LittleEndian.PutUint32(buf[12:16], 0)                   // reserved
		_, err = s.Write(buf[0:16])
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(buf[0:4], uint32(REQ_CPU_WRITE)) // Request
		binary.LittleEndian.PutUint32(buf[4:8], 0x8001)                // Value
		binary.LittleEndian.PutUint32(buf[8:12], uint32(bank+1))       // Length
		binary.LittleEndian.PutUint32(buf[12:16], 0)                   // reserved
		_, err = s.Write(buf[0:16])
		if err != nil {
			return err
		}

		binary.LittleEndian.PutUint32(buf[0:4], uint32(REQ_CPU_READ)) // Request
		binary.LittleEndian.PutUint32(buf[4:8], 0x8000)               // Value
		binary.LittleEndian.PutUint32(buf[8:12], uint32(packetSize))  // Length
		binary.LittleEndian.PutUint32(buf[12:16], 0)                  // reserved
		_, err = s.Write(buf[0:16])
		if err != nil {
			return err
		}

		_, err = io.ReadFull(s, buf)
		if err != nil {
			return err
		}

		_, err = f.Write(buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func dumpTxromCHR(f io.Writer, s io.ReadWriter, chr int, buf []uint8) (err error) {
	packetSize := len(buf)

	banks := (chr * 8 * 1024) >> 10
	for bank := 0; bank < banks; bank += 2 {
		// MMC3: CHR ROM R0:$0000-$07FF swappable
		bankSelect := uint32(0b00000000)
		binary.LittleEndian.PutUint32(buf[0:4], uint32(REQ_CPU_WRITE)) // Request
		binary.LittleEndian.PutUint32(buf[4:8], 0x8000)                // Value
		binary.LittleEndian.PutUint32(buf[8:12], bankSelect)           // Length
		binary.LittleEndian.PutUint32(buf[12:16], 0)                   // reserved
		_, err = s.Write(buf[0:16])
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(buf[0:4], uint32(REQ_CPU_WRITE)) // Request
		binary.LittleEndian.PutUint32(buf[4:8], 0x8001)                // Value
		binary.LittleEndian.PutUint32(buf[8:12], uint32(bank))         // Length
		binary.LittleEndian.PutUint32(buf[12:16], 0)                   // reserved
		_, err = s.Write(buf[0:16])
		if err != nil {
			return err
		}

		binary.LittleEndian.PutUint32(buf[0:4], uint32(REQ_PPU_READ)) // Request
		binary.LittleEndian.PutUint32(buf[4:8], 0)                    // Value
		binary.LittleEndian.PutUint32(buf[8:12], uint32(packetSize))  // Length
		binary.LittleEndian.PutUint32(buf[12:16], 0)                  // reserved
		_, err = s.Write(buf[0:16])
		if err != nil {
			return err
		}

		_, err = io.ReadFull(s, buf)
		if err != nil {
			return err
		}

		_, err = f.Write(buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func dumpAxromPRG(f io.Writer, s io.ReadWriter, prg int, buf []uint8) (err error) {
	packetSize := len(buf)

	banks := (prg * 16 * 1024) >> 15
	for bank := 0; bank < banks; bank++ {
		binary.LittleEndian.PutUint32(buf[0:4], uint32(REQ_CPU_WRITE)) // Request
		binary.LittleEndian.PutUint32(buf[4:8], 0x8000)                // Value
		binary.LittleEndian.PutUint32(buf[8:12], uint32(bank))         // Length
		binary.LittleEndian.PutUint32(buf[12:16], 0)                   // reserved
		_, err = s.Write(buf[0:16])
		if err != nil {
			return err
		}

		binary.LittleEndian.PutUint32(buf[0:4], uint32(REQ_CPU_READ)) // Request
		binary.LittleEndian.PutUint32(buf[4:8], 0x8000)               // Value
		binary.LittleEndian.PutUint32(buf[8:12], uint32(packetSize))  // Length
		binary.LittleEndian.PutUint32(buf[12:16], 0)                  // reserved
		_, err = s.Write(buf[0:16])
		if err != nil {
			return err
		}

		_, err = io.ReadFull(s, buf)
		if err != nil {
			return err
		}

		_, err = f.Write(buf)
		if err != nil {
			return err
		}
	}
	return nil
}
