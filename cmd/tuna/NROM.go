package main

import (
	"encoding/binary"
	"io"
)

func dumpNromPRG(f io.Writer, s io.ReadWriter, prg int, buf []uint8) (err error) {
	for i := 0; i < prg*16*1024; i += PACKET_SIZE {
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
	return nil
}

func dumpNromCHR(f io.Writer, s io.ReadWriter, chr int, buf []uint8) (err error) {
	for i := 0; i < chr*8*1024; i += PACKET_SIZE {
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
	return nil
}
