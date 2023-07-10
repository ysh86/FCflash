package FCflash

import (
	"encoding/binary"
	"fmt"
	"io"
)

const PACKET_SIZE = 0x400

type Request uint8

const (
	REQ_ECHO Request = iota
	REQ_PHI2_INIT

	REQ_CPU_READ_6502
	REQ_CPU_READ
	REQ_CPU_WRITE_6502
	REQ_CPU_WRITE_FLASH

	REQ_PPU_READ
	REQ_PPU_WRITE

	REQ_CPU_WRITE_EEP = 16
	REQ_PPU_WRITE_EEP = 17

	REQ_RAW_READ        = 32
	REQ_RAW_ERASE_FLASH = 33
	REQ_RAW_WRITE_FLASH = 34

	REQ_CPU_WRITE_5BITS_6502 = 35

	REQ_RAW_WRITE = 64
)

type Index uint16

const (
	INDEX_IMPLIED Index = iota
	INDEX_CPU
	INDEX_PPU
	INDEX_BOTH
)

type Message struct {
	_reserverd uint8
	Request    Request
	Value      uint16
	index      Index
	Length     uint16
}

type GB struct {
	Buf []uint8
	s   io.ReadWriter
}

func NewGB(s io.ReadWriter) *GB {
	buf := make([]uint8, PACKET_SIZE)
	return &GB{Buf: buf, s: s}
}

func (g *GB) ReadFull(offset uint32) error {
	n := len(g.Buf)
	if n > PACKET_SIZE {
		return io.ErrShortWrite
	}

	g.Buf[0] = 0 // _reserverd
	g.Buf[1] = uint8(REQ_RAW_READ)
	binary.LittleEndian.PutUint16(g.Buf[2:4], uint16(offset>>8))     // Value
	binary.LittleEndian.PutUint16(g.Buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(g.Buf[6:8], uint16(n))             // Length
	_, err := g.s.Write(g.Buf[0:8])
	if err != nil {
		return err
	}

	_, err = io.ReadFull(g.s, g.Buf)
	if err != nil {
		return err
	}

	return nil
}

func (g *GB) writeByte(addr uint32, data int) error {
	g.Buf[0] = 0 // _reserverd
	g.Buf[1] = uint8(REQ_RAW_WRITE)
	binary.LittleEndian.PutUint16(g.Buf[2:4], uint16(addr>>8))       // Value
	binary.LittleEndian.PutUint16(g.Buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(g.Buf[6:8], 1)                     // Length
	g.Buf[8] = uint8(data & 0xff)
	_, err := g.s.Write(g.Buf[0:9])
	return err
}

func (g *GB) DumpROM(w io.Writer, cartType, romSize byte) (checkSum uint32, err error) {
	numBanks := 2 << int(romSize) // 16[KB/bank]
	currAddr := uint32(0)

	fmt.Printf("Bank: 00")
	for currBank := 1; currBank < numBanks; currBank++ {
		// Set ROM bank
		if cartType == 0 {
			// MBC0
			// nothing to do
		} else if cartType < 5 {
			// MBC1
			g.writeByte(0x6000, 0)             // Set ROM Mode (0: ROM 16Mbit/RAM 8KB mode, 1: ROM 4Mbit/RAM 32KB mode)
			g.writeByte(0x4000, currBank>>5)   // Set bits 5 & 6 (01100000) of ROM bank
			g.writeByte(0x2000, currBank&0x1F) // Set bits 0 & 4 (00011111) of ROM bank
		} else if cartType == 5 || cartType == 6 {
			// MBC2?
			g.writeByte(0x2100, currBank)
		} else if cartType == 0x20 {
			// MBC6
			b := currBank << 1
			g.writeByte(0x2000, b)
			g.writeByte(0x2800, 0)
			g.writeByte(0x3000, b+1)
			g.writeByte(0x3800, 0)
		} else {
			// MBC5 or ???
			//g.writeByte(0x3000, currBank >> 8); // TODO: Are there 32Mbit ROMs?
			//g.writeByte(0x2000, currBank & 0xFF);
			g.writeByte(0x2100, currBank&0xFF)
		}

		// Switch bank start address
		if currBank > 1 {
			currAddr = 0x4000
		}
		for ; currAddr < 0x7FFF; currAddr += PACKET_SIZE {
			err = g.ReadFull(currAddr)
			if err != nil {
				return 0, err
			}
			w.Write(g.Buf)

			// checksum
			for currByte := uint32(0); currByte < PACKET_SIZE; currByte++ {
				if currAddr+currByte != 0x014e && currAddr+currByte != 0x014f {
					checkSum += uint32(g.Buf[currByte])
				}
			}
		}

		fmt.Printf(" %02x", currBank)
	}
	fmt.Printf("\n")

	return checkSum, nil
}
