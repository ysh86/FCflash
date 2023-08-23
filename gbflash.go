package FCflash

import (
	"encoding/binary"
	"fmt"
	"io"
)

var commands map[uint16][3]uint16

func init() {
	commands = map[uint16][3]uint16{
		0x01ad: {0x555, 0x2aa, 8},  // AMD Am29F016: 8-bit data bus
		0x01d2: {0xaaa, 0x555, 16}, // Micron M29F160FT: 16-bit data bus, 8-bit mode
	}
}

func (g *GB) readFlashReg(addr uint16) (byte, error) {
	g.Buf[0] = 0 // _reserverd
	g.Buf[1] = uint8(REQ_RAW_READ_LO)
	binary.LittleEndian.PutUint16(g.Buf[2:4], addr)                  // Value
	binary.LittleEndian.PutUint16(g.Buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(g.Buf[6:8], 1)                     // Length
	_, err := g.s.Write(g.Buf[0:8])
	if err != nil {
		return 0, err
	}

	_, err = io.ReadFull(g.s, g.Buf[8:(8+1)])
	if err != nil {
		return 0, err
	}

	return g.Buf[8], nil
}

func (g *GB) writeFlashReg(addr uint16, data byte) error {
	g.Buf[0] = 0 // _reserverd
	g.Buf[1] = uint8(REQ_RAW_WRITE_LO)
	binary.LittleEndian.PutUint16(g.Buf[2:4], addr)                  // Value
	binary.LittleEndian.PutUint16(g.Buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(g.Buf[6:8], 1)                     // Length
	g.Buf[8] = data
	_, err := g.s.Write(g.Buf[0:(8 + 1)])
	return err
}

func (g *GB) detectFlash8() (byte, byte, error) {
	// Reset
	g.writeFlashReg(0x555, 0xf0)

	// Autoselect Command
	g.writeFlashReg(0x555, 0xaa)
	g.writeFlashReg(0x2aa, 0x55)
	g.writeFlashReg(0x555, 0x90)

	manufacturerCode, err := g.readFlashReg(0x0000)
	if err != nil {
		return 0, 0, err
	}
	deviceCode, err := g.readFlashReg(0x0001)
	if err != nil {
		return 0, 0, err
	}

	// Reset
	g.writeFlashReg(0x555, 0xf0)

	return manufacturerCode, deviceCode, nil
}

func (g *GB) detectFlash16() (byte, byte, error) {
	// Reset
	g.writeFlashReg(0xaaa, 0xf0)

	// Autoselect Command
	g.writeFlashReg(0xaaa, 0xaa)
	g.writeFlashReg(0x555, 0x55)
	g.writeFlashReg(0xaaa, 0x90)

	manufacturerCode, err := g.readFlashReg(0x0000)
	if err != nil {
		return 0, 0, err
	}
	deviceCode, err := g.readFlashReg(0x0002)
	if err != nil {
		return 0, 0, err
	}

	// Reset
	g.writeFlashReg(0xaaa, 0xf0)

	return manufacturerCode, deviceCode, nil
}

func (g *GB) IsSupportedFlash() (device uint16, err error) {
	manufacturerCode, deviceCode, err := g.detectFlash8()
	if err != nil {
		return 0, err
	}
	// AMD Am29F016: 8-bit bus
	if manufacturerCode != 0x01 || deviceCode != 0xad {
		manufacturerCode, deviceCode, err = g.detectFlash16()
		if err != nil {
			return 0, err
		}
		// Micron M29F160FT: 16-bit bus, 8-bit mode
		if manufacturerCode != 0x01 || deviceCode != 0xd2 {
			return 0, fmt.Errorf("not supported device: %02x%02x", manufacturerCode, deviceCode)
		}
	}

	device = (uint16(manufacturerCode) << 8) | uint16(deviceCode)
	return device, err
}

func (g *GB) WriteFlash(device uint16, addr int, buf []byte) error {
	// Reset
	g.writeFlashReg(commands[device][0], 0xf0)

	// flash: 64KB sector
	//  8-bit flash: sa = # of sector
	// 16-bit flash: sa = phy addr of block
	if addr&0xffff == 0 {
		// Sector Erase
		sa := uint16(addr >> 16)
		if commands[device][2] == 16 && addr >= 0x10000 {
			sa = 0x4000 // MBC sets phy addr.
		}
		g.writeFlashReg(commands[device][0], 0xaa)
		g.writeFlashReg(commands[device][1], 0x55)
		g.writeFlashReg(commands[device][0], 0x80)
		g.writeFlashReg(commands[device][0], 0xaa)
		g.writeFlashReg(commands[device][1], 0x55)
		g.writeFlashReg(sa, 0x30)

		// wait
		for {
			status, err := g.readFlashReg(sa)
			if err != nil {
				return err
			}
			if status&0x80 != 0 {
				// done
				break
			}

			if status&0x20 != 0 {
				// retry
				status, err := g.readFlashReg(sa)
				if err != nil {
					return err
				}
				if status&0x80 != 0 {
					// done
					break
				} else {
					return fmt.Errorf("exceeded time limits: erase sector: %04x", sa)
				}
			}
		}
	}

	// write
	for i, d := range buf {
		if d == 0xff {
			continue
		}

		pa := addr + i
		va := uint16(pa & 0x0000_3fff)
		if pa >= 0x4000 {
			va |= 0x4000
		}

		g.writeFlashReg(commands[device][0], 0xaa)
		g.writeFlashReg(commands[device][1], 0x55)
		g.writeFlashReg(commands[device][0], 0xa0)
		g.writeFlashReg(va, d)

		// wait
		for {
			status, err := g.readFlashReg(va)
			if err != nil {
				return err
			}
			if status&0x80 == d&0x80 {
				// done
				break
			}

			if status&0x20 != 0 {
				// retry
				status, err := g.readFlashReg(va)
				if err != nil {
					return err
				}
				if status&0x80 == d&0x80 {
					// done
					break
				} else {
					return fmt.Errorf("exceeded time limits: write byte: pa=%06x, va=%04x", pa, va)
				}
			}
		}
	}

	// Reset
	g.writeFlashReg(commands[device][0], 0xf0)

	return nil
}
