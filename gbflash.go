package FCflash

import (
	"encoding/binary"
	"errors"
	"io"
)

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

func (g *GB) DetectFlash() (byte, byte, error) {
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

func (g *GB) WriteFlash(addr int, buf []byte) error {
	// Reset
	g.writeFlashReg(0x555, 0xf0)

	// flash: 64KB sector
	if addr&0xffff == 0 {
		// Sector Erase
		sa := uint16(addr >> 16)
		g.writeFlashReg(0x555, 0xaa)
		g.writeFlashReg(0x2aa, 0x55)
		g.writeFlashReg(0x555, 0x80)
		g.writeFlashReg(0x555, 0xaa)
		g.writeFlashReg(0x2aa, 0x55)
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
					return errors.New("exceeded time limits")
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

		g.writeFlashReg(0x555, 0xaa)
		g.writeFlashReg(0x2aa, 0x55)
		g.writeFlashReg(0x555, 0xa0)
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
					return errors.New("exceeded time limits")
				}
			}
		}
	}

	// Reset
	g.writeFlashReg(0x555, 0xf0)

	return nil
}
