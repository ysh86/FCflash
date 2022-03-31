package main

import (
	"encoding/binary"
	"io"
)

func dumpRAW(f io.Writer, s io.ReadWriter, size int, buf []uint8) (err error) {
	for i := 0; i < size; i += PACKET_SIZE {
		buf[0] = 0 // _reserverd
		buf[1] = uint8(REQ_RAW_READ)
		binary.LittleEndian.PutUint16(buf[2:4], uint16(i>>8))          // Value
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
