package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

func writeFlash(s io.Writer, fileName string, size int, buf []uint8) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	// skip header
	_, err = f.Seek(16, io.SeekStart)
	if err != nil {
		return err
	}

	// erase automatically
	/*
		buf[0] = 0 // _reserverd
		buf[1] = uint8(REQ_RAW_ERASE_FLASH)
		binary.LittleEndian.PutUint16(buf[2:4], 0)                     // Value
		binary.LittleEndian.PutUint16(buf[4:6], uint16(INDEX_IMPLIED)) // index
		binary.LittleEndian.PutUint16(buf[6:8], 0)                     // Length
		_, err = s.Write(buf[0:8])
		if err != nil {
			return err
		}
		fmt.Println("erased")
	*/

	for i := 0; i < size; i += PACKET_SIZE {
		fmt.Printf(".")

		buf[0] = 0 // _reserverd
		buf[1] = uint8(REQ_RAW_WRITE_FLASH)
		binary.LittleEndian.PutUint16(buf[2:4], uint16(i>>8))          // Value
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
	fmt.Println("")

	return nil
}
