package main

import (
	"encoding/binary"
	"io"
)

func dumpTxromPRG(f io.Writer, s io.ReadWriter, prg int, buf []uint8) (err error) {
	// MMC3: PRG ROM R6:$8000-$9FFF swappable
	bankSelect := uint16(0b00000110)
	buf[0] = 0 // _reserverd
	buf[1] = uint8(REQ_CPU_WRITE_6502)
	binary.LittleEndian.PutUint16(buf[2:4], 0x8000)                // Value
	binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(buf[6:8], bankSelect)            // Length
	_, err = s.Write(buf[0:8])
	if err != nil {
		return err
	}

	banks := (prg * 16 * 1024) >> 13
	for bank := 0; bank < banks; bank++ {
		buf[0] = 0 // _reserverd
		buf[1] = uint8(REQ_CPU_WRITE_6502)
		binary.LittleEndian.PutUint16(buf[2:4], 0x8001)                // Value
		binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
		binary.LittleEndian.PutUint16(buf[6:8], uint16(bank))          // Length
		_, err = s.Write(buf[0:8])
		if err != nil {
			return err
		}

		for i := 0; i < 0x2000; i += PACKET_SIZE {
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

func dumpTxromCHR(f io.Writer, s io.ReadWriter, chr int, buf []uint8) (err error) {
	// MMC3: CHR ROM R0:$0000-$07FF swappable
	bankSelect := uint16(0b00000000)
	buf[0] = 0 // _reserverd
	buf[1] = uint8(REQ_CPU_WRITE_6502)
	binary.LittleEndian.PutUint16(buf[2:4], 0x8000)                // Value
	binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(buf[6:8], bankSelect)            // Length
	_, err = s.Write(buf[0:8])
	if err != nil {
		return err
	}

	banks := (chr * 8 * 1024) >> 10
	for bank := 0; bank < banks; bank += 2 {
		buf[0] = 0 // _reserverd
		buf[1] = uint8(REQ_CPU_WRITE_6502)
		binary.LittleEndian.PutUint16(buf[2:4], 0x8001)                // Value
		binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
		binary.LittleEndian.PutUint16(buf[6:8], uint16(bank))          // Length
		_, err = s.Write(buf[0:8])
		if err != nil {
			return err
		}

		for i := 0; i < 0x800; i += PACKET_SIZE {
			buf[0] = 0 // _reserverd
			buf[1] = uint8(REQ_PPU_READ)
			binary.LittleEndian.PutUint16(buf[2:4], uint16(i))             // Value
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
