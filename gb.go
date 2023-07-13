package FCflash

import (
	"encoding/binary"
	"fmt"
	"io"
)

type GB struct {
	Buf []uint8
	s   io.ReadWriter
}

func NewGB(s io.ReadWriter) *GB {
	buf := make([]uint8, PACKET_SIZE*2)
	return &GB{Buf: buf, s: s}
}

func (g *GB) ReadFull(offset uint32) error {
	g.Buf[0] = 0 // _reserverd
	g.Buf[1] = uint8(REQ_RAW_READ)
	binary.LittleEndian.PutUint16(g.Buf[2:4], uint16(offset>>8))     // Value
	binary.LittleEndian.PutUint16(g.Buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(g.Buf[6:8], uint16(PACKET_SIZE))   // Length
	_, err := g.s.Write(g.Buf[0:8])
	if err != nil {
		return err
	}

	_, err = io.ReadFull(g.s, g.Buf[0:PACKET_SIZE])
	if err != nil {
		return err
	}

	return nil
}

func (g *GB) writeRegByte(addr uint32, data int) error {
	g.Buf[0] = 0 // _reserverd
	g.Buf[1] = uint8(REQ_RAW_WRITE_WO_CS)
	binary.LittleEndian.PutUint16(g.Buf[2:4], uint16(addr>>8))       // Value
	binary.LittleEndian.PutUint16(g.Buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(g.Buf[6:8], 1)                     // Length
	g.Buf[8] = uint8(data & 0xff)
	_, err := g.s.Write(g.Buf[0:(8 + 1)])
	return err
}

func (g *GB) setMemory(addr uint32, value byte, limit uint32) error {
	if limit > PACKET_SIZE {
		return fmt.Errorf("too long: %d", limit)
	}
	n := uint16(limit)

	g.Buf[0] = 0 // _reserverd
	g.Buf[1] = uint8(REQ_RAW_WRITE)
	binary.LittleEndian.PutUint16(g.Buf[2:4], uint16(addr>>8))       // Value
	binary.LittleEndian.PutUint16(g.Buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(g.Buf[6:8], n)                     // Length
	for i := range g.Buf[8:(8 + n)] {
		g.Buf[8+i] = value
	}
	_, err := g.s.Write(g.Buf[0:(8 + n)])
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
			g.writeRegByte(0x6000, 0)             // Set ROM Mode (0: ROM 16Mbit/RAM 8KB mode, 1: ROM 4Mbit/RAM 32KB mode)
			g.writeRegByte(0x4000, currBank>>5)   // Set bits 5 & 6 (01100000) of ROM bank
			g.writeRegByte(0x2000, currBank&0x1F) // Set bits 0 & 4 (00011111) of ROM bank
		} else if cartType == 5 || cartType == 6 {
			// MBC2?
			g.writeRegByte(0x2100, currBank)
		} else if cartType == 0x20 {
			// MBC6
			b := currBank << 1
			g.writeRegByte(0x2000, b)
			g.writeRegByte(0x2800, 0)
			g.writeRegByte(0x3000, b+1)
			g.writeRegByte(0x3800, 0)
		} else {
			// MBC5 or ???
			//g.writeRegByte(0x3000, currBank >> 8); // TODO: Are there 32Mbit ROMs?
			//g.writeRegByte(0x2000, currBank & 0xFF);
			g.writeRegByte(0x2100, currBank&0xFF)
		}

		// Switch bank start address
		if currBank > 1 {
			currAddr = 0x4000
		}
		for ; currAddr < 0x7FFF; currAddr += PACKET_SIZE {
			err = g.ReadFull(currAddr)
			if err != nil {
				return checkSum, err
			}
			w.Write(g.Buf[0:PACKET_SIZE])

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

func calcRamSize(cartType, ramSize byte) (size uint32, numBanks int, err error) {
	switch ramSize {
	case 0:
		// MBC2 includes a built-in RAM
		if cartType == 6 {
			size = 512 // nibbles
			numBanks = 1
		} else {
			return size, numBanks, fmt.Errorf("invalid ramSize: %d", ramSize)
		}
	case 1:
		size = 2 * 1024
		numBanks = 1
	case 2:
		size = 8 * 1024
		numBanks = 1
	case 3:
		size = 32 * 1024
		numBanks = 4
	case 4:
		size = 128 * 1024
		numBanks = 16
	case 5:
		size = 64 * 1024
		numBanks = 8
	default:
		return size, numBanks, fmt.Errorf("invalid ramSize: %d", ramSize)
	}

	return size, numBanks, nil
}

func (g *GB) DumpRAM(w io.Writer, cartType, ramSize byte) (size uint32, err error) {
	size, numBanks, err := calcRamSize(cartType, ramSize)
	if err != nil {
		return size, err
	}

	// MBC1
	if cartType < 5 {
		g.writeRegByte(0x6000, 1) // Set RAM Mode
	}

	// enable RAM
	g.writeRegByte(0x0000, 0x0a)

	// Switch RAM banks: 8[KB/bank] @ a000-end
	for currBank := 0; currBank < numBanks; currBank++ {
		g.writeRegByte(0x4000, currBank)

		for addr := uint32(0); addr < 8192; addr += PACKET_SIZE {
			err = g.ReadFull(0xa000 + addr)
			if err != nil {
				return size, err
			}
			if size < PACKET_SIZE {
				w.Write(g.Buf[0:size])
				break
			}
			w.Write(g.Buf[0:PACKET_SIZE])
		}
	}

	// disable RAM
	g.writeRegByte(0x0000, 0x00)

	// MBC1
	if cartType < 5 {
		g.writeRegByte(0x6000, 0)
	}

	return size, nil
}

func (g *GB) ClearRAM(cartType, ramSize byte) (size uint32, err error) {
	size, numBanks, err := calcRamSize(cartType, ramSize)
	if err != nil {
		return size, err
	}

	// MBC1
	if cartType < 5 {
		g.writeRegByte(0x6000, 1) // Set RAM Mode
	}

	// enable RAM
	g.writeRegByte(0x0000, 0x0a)

	// Switch RAM banks: 8[KB/bank] @ a000-end
	limit := uint32(PACKET_SIZE)
	if size < limit {
		limit = size
	}
	for currBank := 0; currBank < numBanks; currBank++ {
		g.writeRegByte(0x4000, currBank)

		for addr := uint32(0); addr < 8192; addr += PACKET_SIZE {
			// zero
			err = g.setMemory(0xa000+addr, 0xfd, limit)
			if err != nil {
				return size, err
			}
			if size < PACKET_SIZE {
				break
			}
		}
	}

	// disable RAM
	g.writeRegByte(0x0000, 0x00)

	// MBC1
	if cartType < 5 {
		g.writeRegByte(0x6000, 0)
	}

	return size, nil
}
