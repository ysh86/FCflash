package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

func dumpSxromPRG(f io.Writer, s io.ReadWriter, prg int, buf []uint8) (err error) {
	// MMC1: reset
	resetValue := uint16(0xff)
	buf[0] = 0 // _reserverd
	buf[1] = uint8(REQ_CPU_WRITE_6502)
	binary.LittleEndian.PutUint16(buf[2:4], 0x8000)                // Value
	binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(buf[6:8], resetValue)            // Length
	_, err = s.Write(buf[0:8])
	if err != nil {
		return err
	}
	// MMC1: (from LSB)
	// Mirroring 3: horizontal
	// PRG ROM bank mode 3:$8000-$BFFF 16KB swappable
	// CHR ROM bank mode 0:8KB single
	bankControl := uint16(0b01111)
	buf[0] = 0 // _reserverd
	buf[1] = uint8(REQ_CPU_WRITE_5BITS_6502)
	binary.LittleEndian.PutUint16(buf[2:4], 0x8000)                // Value
	binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(buf[6:8], bankControl)           // Length
	_, err = s.Write(buf[0:8])
	if err != nil {
		return err
	}

	// for SUROM
	banks1st := 0
	banks2nd := 0
	if prg <= 16 {
		banks1st = prg
	} else {
		banks1st = 16
		banks2nd = prg - 16
	}

	// 1st
	//
	// MMC1: (from LSB)
	// CHR RAM bank 0:(ignored in 8KB mode)
	// xxx
	// PRG 256KB bank 0:1st (PRG RAM disable 0:enable)
	chrBank := uint16(0b00000)
	buf[0] = 0 // _reserverd
	buf[1] = uint8(REQ_CPU_WRITE_5BITS_6502)
	binary.LittleEndian.PutUint16(buf[2:4], 0xA000)                // Value
	binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(buf[6:8], chrBank)               // Length
	_, err = s.Write(buf[0:8])
	if err != nil {
		return err
	}
	for bank := 0; bank < banks1st; bank++ {
		buf[0] = 0 // _reserverd
		buf[1] = uint8(REQ_CPU_WRITE_5BITS_6502)
		binary.LittleEndian.PutUint16(buf[2:4], 0xE000)                // Value
		binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
		binary.LittleEndian.PutUint16(buf[6:8], uint16(bank))          // Length
		_, err = s.Write(buf[0:8])
		if err != nil {
			return err
		}

		for i := 0; i < 0x4000; i += PACKET_SIZE {
			buf[0] = 0 // _reserverd
			buf[1] = uint8(REQ_CPU_READ)
			binary.LittleEndian.PutUint16(buf[2:4], 0x8000|uint16(i))      // Value
			binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
			binary.LittleEndian.PutUint16(buf[6:8], PACKET_SIZE)           // Length
			_, err = s.Write(buf[0:8])
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
	}
	if banks2nd == 0 {
		return nil
	}

	// 2nd
	//
	// MMC1: (from LSB)
	// CHR RAM bank 0:(ignored in 8KB mode)
	// xxx
	// PRG 256KB bank 1:2nd (PRG RAM disable 1:open bus)
	chrBank = uint16(0b10000)
	buf[0] = 0 // _reserverd
	buf[1] = uint8(REQ_CPU_WRITE_5BITS_6502)
	binary.LittleEndian.PutUint16(buf[2:4], 0xA000)                // Value
	binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(buf[6:8], chrBank)               // Length
	_, err = s.Write(buf[0:8])
	if err != nil {
		return err
	}
	for bank := 0; bank < banks2nd; bank++ {
		buf[0] = 0 // _reserverd
		buf[1] = uint8(REQ_CPU_WRITE_5BITS_6502)
		binary.LittleEndian.PutUint16(buf[2:4], 0xE000)                // Value
		binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
		binary.LittleEndian.PutUint16(buf[6:8], uint16(bank))          // Length
		_, err = s.Write(buf[0:8])
		if err != nil {
			return err
		}

		for i := 0; i < 0x4000; i += PACKET_SIZE {
			buf[0] = 0 // _reserverd
			buf[1] = uint8(REQ_CPU_READ)
			binary.LittleEndian.PutUint16(buf[2:4], 0x8000|uint16(i))      // Value
			binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
			binary.LittleEndian.PutUint16(buf[6:8], PACKET_SIZE)           // Length
			_, err = s.Write(buf[0:8])
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
	}
	return nil
}

func writeSxromPRG(s io.Writer, fileName string, prg int, buf []uint8) (err error) {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	// MMC1: reset
	resetValue := uint16(0xff)
	buf[0] = 0 // _reserverd
	buf[1] = uint8(REQ_CPU_WRITE_6502)
	binary.LittleEndian.PutUint16(buf[2:4], 0x8000)                // Value
	binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(buf[6:8], resetValue)            // Length
	_, err = s.Write(buf[0:8])
	if err != nil {
		return err
	}

	// for SUROM
	banks1st := 0
	banks2nd := 0
	if prg <= 16 {
		banks1st = prg
	} else {
		banks1st = 16
		banks2nd = prg - 16
	}

	// ------------------------
	// for even banks
	// ------------------------

	// MMC1: (from LSB)
	// Mirroring 3: horizontal
	// PRG ROM bank mode 3:$8000-$BFFF 16KB swappable
	// CHR ROM bank mode 0:8KB single
	bankControl := uint16(0b01111)
	buf[0] = 0 // _reserverd
	buf[1] = uint8(REQ_CPU_WRITE_5BITS_6502)
	binary.LittleEndian.PutUint16(buf[2:4], 0x8000)                // Value
	binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(buf[6:8], bankControl)           // Length
	_, err = s.Write(buf[0:8])
	if err != nil {
		return err
	}

	// 1st
	//
	// MMC1: (from LSB)
	// CHR RAM bank 0:(ignored in 8KB mode)
	// xxx
	// PRG 256KB bank 0:1st (PRG RAM disable 0:enable)
	chrBank := uint16(0b00000)
	buf[0] = 0 // _reserverd
	buf[1] = uint8(REQ_CPU_WRITE_5BITS_6502)
	binary.LittleEndian.PutUint16(buf[2:4], 0xA000)                // Value
	binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(buf[6:8], chrBank)               // Length
	_, err = s.Write(buf[0:8])
	if err != nil {
		return err
	}
	fmt.Printf("even 1st: ")
	for bank := 0; bank < banks1st; bank += 2 {
		buf[0] = 0 // _reserverd
		buf[1] = uint8(REQ_CPU_WRITE_5BITS_6502)
		binary.LittleEndian.PutUint16(buf[2:4], 0xE000)                // Value
		binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
		binary.LittleEndian.PutUint16(buf[6:8], uint16(bank))          // Length
		_, err = s.Write(buf[0:8])
		if err != nil {
			return err
		}

		// seek
		_, err = f.Seek(16+16*1024*int64(bank), io.SeekStart)
		if err != nil {
			return err
		}

		for i := 0; i < 0x4000; i += PACKET_SIZE {
			fmt.Printf(".")

			buf[0] = 0 // _reserverd
			buf[1] = uint8(REQ_CPU_WRITE_FLASH)
			binary.LittleEndian.PutUint16(buf[2:4], uint16(i))             // Value
			binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
			binary.LittleEndian.PutUint16(buf[6:8], PACKET_SIZE)           // Length
			_, err = s.Write(buf[0:8])
			if err != nil {
				return err
			}

			_, err = io.ReadFull(f, buf)
			if err != nil {
				return err
			}

			_, err = s.Write(buf)
			if err != nil {
				return err
			}
		}
	}
	fmt.Println("")

	// 2nd
	//
	// MMC1: (from LSB)
	// CHR RAM bank 0:(ignored in 8KB mode)
	// xxx
	// PRG 256KB bank 1:2nd (PRG RAM disable 1:open bus)
	chrBank = uint16(0b10000)
	buf[0] = 0 // _reserverd
	buf[1] = uint8(REQ_CPU_WRITE_5BITS_6502)
	binary.LittleEndian.PutUint16(buf[2:4], 0xA000)                // Value
	binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(buf[6:8], chrBank)               // Length
	_, err = s.Write(buf[0:8])
	if err != nil {
		return err
	}
	fmt.Printf("even 2nd: ")
	for bank := 0; bank < banks2nd; bank += 2 {
		buf[0] = 0 // _reserverd
		buf[1] = uint8(REQ_CPU_WRITE_5BITS_6502)
		binary.LittleEndian.PutUint16(buf[2:4], 0xE000)                // Value
		binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
		binary.LittleEndian.PutUint16(buf[6:8], uint16(bank))          // Length
		_, err = s.Write(buf[0:8])
		if err != nil {
			return err
		}

		// seek
		_, err = f.Seek(16+256*1024+16*1024*int64(bank), io.SeekStart)
		if err != nil {
			return err
		}

		for i := 0; i < 0x4000; i += PACKET_SIZE {
			fmt.Printf(".")

			buf[0] = 0 // _reserverd
			buf[1] = uint8(REQ_CPU_WRITE_FLASH)
			binary.LittleEndian.PutUint16(buf[2:4], uint16(i))             // Value
			binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
			binary.LittleEndian.PutUint16(buf[6:8], PACKET_SIZE)           // Length
			_, err = s.Write(buf[0:8])
			if err != nil {
				return err
			}

			_, err = io.ReadFull(f, buf)
			if err != nil {
				return err
			}

			_, err = s.Write(buf)
			if err != nil {
				return err
			}
		}
	}
	fmt.Println("")

	// ------------------------
	// for odd banks
	// ------------------------

	// MMC1: (from LSB)
	// Mirroring 3: horizontal
	// PRG ROM bank mode 2:$C000-$FFFF 16KB swappable
	// CHR ROM bank mode 0:8KB single
	bankControl = uint16(0b01011)
	buf[0] = 0 // _reserverd
	buf[1] = uint8(REQ_CPU_WRITE_5BITS_6502)
	binary.LittleEndian.PutUint16(buf[2:4], 0x8000)                // Value
	binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(buf[6:8], bankControl)           // Length
	_, err = s.Write(buf[0:8])
	if err != nil {
		return err
	}

	// 1st
	//
	// MMC1: (from LSB)
	// CHR RAM bank 0:(ignored in 8KB mode)
	// xxx
	// PRG 256KB bank 0:1st (PRG RAM disable 0:enable)
	chrBank = uint16(0b00000)
	buf[0] = 0 // _reserverd
	buf[1] = uint8(REQ_CPU_WRITE_5BITS_6502)
	binary.LittleEndian.PutUint16(buf[2:4], 0xA000)                // Value
	binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(buf[6:8], chrBank)               // Length
	_, err = s.Write(buf[0:8])
	if err != nil {
		return err
	}
	fmt.Printf("odd  1st: ")
	for bank := 1; bank < banks1st; bank += 2 {
		buf[0] = 0 // _reserverd
		buf[1] = uint8(REQ_CPU_WRITE_5BITS_6502)
		binary.LittleEndian.PutUint16(buf[2:4], 0xE000)                // Value
		binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
		binary.LittleEndian.PutUint16(buf[6:8], uint16(bank))          // Length
		_, err = s.Write(buf[0:8])
		if err != nil {
			return err
		}

		// seek
		_, err = f.Seek(16+16*1024*int64(bank), io.SeekStart)
		if err != nil {
			return err
		}

		for i := 0; i < 0x4000; i += PACKET_SIZE {
			fmt.Printf(".")

			buf[0] = 0 // _reserverd
			buf[1] = uint8(REQ_CPU_WRITE_FLASH)
			binary.LittleEndian.PutUint16(buf[2:4], 0x4000|uint16(i))      // Value
			binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
			binary.LittleEndian.PutUint16(buf[6:8], PACKET_SIZE)           // Length
			_, err = s.Write(buf[0:8])
			if err != nil {
				return err
			}

			_, err = io.ReadFull(f, buf)
			if err != nil {
				return err
			}

			_, err = s.Write(buf)
			if err != nil {
				return err
			}
		}
	}
	fmt.Println("")

	// 2nd
	//
	// MMC1: (from LSB)
	// CHR RAM bank 0:(ignored in 8KB mode)
	// xxx
	// PRG 256KB bank 1:2nd (PRG RAM disable 1:open bus)
	chrBank = uint16(0b10000)
	buf[0] = 0 // _reserverd
	buf[1] = uint8(REQ_CPU_WRITE_5BITS_6502)
	binary.LittleEndian.PutUint16(buf[2:4], 0xA000)                // Value
	binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(buf[6:8], chrBank)               // Length
	_, err = s.Write(buf[0:8])
	if err != nil {
		return err
	}
	fmt.Printf("odd  2nd: ")
	for bank := 1; bank < banks2nd; bank += 2 {
		buf[0] = 0 // _reserverd
		buf[1] = uint8(REQ_CPU_WRITE_5BITS_6502)
		binary.LittleEndian.PutUint16(buf[2:4], 0xE000)                // Value
		binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
		binary.LittleEndian.PutUint16(buf[6:8], uint16(bank))          // Length
		_, err = s.Write(buf[0:8])
		if err != nil {
			return err
		}

		// seek
		_, err = f.Seek(16+256*1024+16*1024*int64(bank), io.SeekStart)
		if err != nil {
			return err
		}

		for i := 0; i < 0x4000; i += PACKET_SIZE {
			fmt.Printf(".")

			buf[0] = 0 // _reserverd
			buf[1] = uint8(REQ_CPU_WRITE_FLASH)
			binary.LittleEndian.PutUint16(buf[2:4], 0x4000|uint16(i))      // Value
			binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
			binary.LittleEndian.PutUint16(buf[6:8], PACKET_SIZE)           // Length
			_, err = s.Write(buf[0:8])
			if err != nil {
				return err
			}

			_, err = io.ReadFull(f, buf)
			if err != nil {
				return err
			}

			_, err = s.Write(buf)
			if err != nil {
				return err
			}
		}
	}
	fmt.Println("")

	return nil
}
