package FCflash

import (
	"encoding/binary"
	"errors"
	"io"
	"time"
)

// GB Memory cart
// ROM: 29F008ATC-14    8 Mbits (1 Mbyte) flash
// RAM: UT621024SC-70LL 1 Mbits (128 KBytes) SRAM
// MBC: G-MMC1

const (
	GBM_MENU_TITLE      = "NP M-MENU  MENU"
	GBM_ENTIRE_MBC      = 0x1B // as MBC5 + RAM + battery
	GBM_ENTIRE_ROM_SIZE = 5    // 2<<5 = 64banks -> 64*16 = 1024KB
	GBM_ENTIRE_RAM_SIZE = 3    //         4banks ->  4* 8 =   32KB
)

const (
	CMD_BANK = 0x99

	CMD_01h_UNKNOWN = 0x01 // ???

	CMD_0Ah_WRITE_ENABLE_STEP_1 = 0x0A
	CMD_02h_WRITE_ENABLE_STEP_2 = 0x02
	CMD_03h_UNDO_WRITE_STEP_2   = 0x03

	CMD_04h_MAP_ENTIRE_ROM = 0x04 // MBC4 mode
	CMD_05h_MAP_MENU       = 0x05 // MBC5 mode

	CMD_09h_WAKEUP_AND_RE_ENABLE_FLASH_REGS = 0x09 // regs at 0120h..013Fh
	CMD_08h_DISABLE_FLASH_REGS              = 0x08

	CMD_11h_RE_ENABLE_MBC_REGS = 0x11 // regs at 2100h
	CMD_10h_DISABLE_MBC_REGS   = 0x10

	CMD_0Fh_WRITE_TO_FLASH = 0x0f

	CMD_C0h_MAP_SELECTED_GAME_WITHOUT_RESET = 0xc0
)

func (g *GB) Read256(offset uint32) error {
	g.Buf[0] = 0 // _reserverd
	g.Buf[1] = uint8(REQ_RAW_READ_WO_CS)
	binary.LittleEndian.PutUint16(g.Buf[2:4], uint16(offset>>8))     // Value
	binary.LittleEndian.PutUint16(g.Buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(g.Buf[6:8], 256)                   // Length
	_, err := g.s.Write(g.Buf[0:8])
	if err != nil {
		return err
	}

	_, err = io.ReadFull(g.s, g.Buf[0:256])
	if err != nil {
		return err
	}

	return nil
}

func (g *GB) writeGBMReg(addr uint16, data byte) error {
	g.Buf[0] = 0 // _reserverd
	g.Buf[1] = uint8(REQ_RAW_WRITE_LO_WO_CS)
	binary.LittleEndian.PutUint16(g.Buf[2:4], addr)                  // Value
	binary.LittleEndian.PutUint16(g.Buf[4:6], uint16(INDEX_IMPLIED)) // index
	binary.LittleEndian.PutUint16(g.Buf[6:8], 1)                     // Length
	g.Buf[8] = data
	_, err := g.s.Write(g.Buf[0:(8 + 1)])
	return err
}

func (g *GB) commandGBM(command byte, addr uint16, data byte) {
	if command == CMD_BANK {
		g.writeGBMReg(addr, data)
		return
	}

	if command != CMD_C0h_MAP_SELECTED_GAME_WITHOUT_RESET {
		g.writeGBMReg(0x0120, command)
	} else {
		g.writeGBMReg(0x0120, command|data) // select game
	}

	if command == CMD_09h_WAKEUP_AND_RE_ENABLE_FLASH_REGS {
		g.writeGBMReg(0x0121, 0xAA)
		g.writeGBMReg(0x0122, 0x55)
	}
	if command == CMD_0Ah_WRITE_ENABLE_STEP_1 {
		g.writeGBMReg(0x0125, 0x62)
		g.writeGBMReg(0x0126, 0x04)
	}
	if command == CMD_0Fh_WRITE_TO_FLASH {
		g.writeGBMReg(0x0125, byte((addr>>8)&0xff)) // high
		g.writeGBMReg(0x0126, byte(addr&0xff))      // low
		g.writeGBMReg(0x0127, data)
	}

	g.writeGBMReg(0x013F, 0xA5)
}

func (g *GB) resetGBMFlash() {
	// Enable ports 0x0120
	g.commandGBM(CMD_09h_WAKEUP_AND_RE_ENABLE_FLASH_REGS, 0, 0)

	// Send reset command
	g.commandGBM(CMD_BANK, 0x2100, 0x01)
	g.commandGBM(CMD_0Fh_WRITE_TO_FLASH, 0x5555, 0xAA)
	g.commandGBM(CMD_0Fh_WRITE_TO_FLASH, 0x2AAA, 0x55)
	g.commandGBM(CMD_0Fh_WRITE_TO_FLASH, 0x5555, 0xF0)

	time.Sleep(100 * time.Millisecond)

	// Disable ports 0x0120h
	g.commandGBM(CMD_08h_DISABLE_FLASH_REGS, 0, 0)
}

func (g *GB) DetectGBM() error {
	// Check for Nintendo Power GB Memory cart
	// First byte of NP register is always 0x21
	i := 0
	for ; i < 1000; i++ {
		reg0120h, err := g.readFlashReg(0x0120)
		if err != nil {
			return err
		}
		if reg0120h == 0x21 {
			break
		}

		// Enable access to ports 0120h
		g.commandGBM(CMD_09h_WAKEUP_AND_RE_ENABLE_FLASH_REGS, 0, 0)
		time.Sleep(1 * time.Millisecond)
	}
	if i == 1000 {
		return errors.New("time out")
	}

	// Disable ports 0x0120h
	g.commandGBM(CMD_08h_DISABLE_FLASH_REGS, 0, 0)

	return nil
}

func (g *GB) ReadMappingGBM(w io.Writer) error {
	// Enable ports 0x0120
	g.commandGBM(CMD_09h_WAKEUP_AND_RE_ENABLE_FLASH_REGS, 0, 0)

	// Set WE and WP
	g.commandGBM(CMD_0Ah_WRITE_ENABLE_STEP_1, 0, 0)
	g.commandGBM(CMD_02h_WRITE_ENABLE_STEP_2, 0, 0)

	// Enable hidden mapping area
	g.commandGBM(CMD_BANK, 0x2100, 0x01)
	g.commandGBM(CMD_0Fh_WRITE_TO_FLASH, 0x5555, 0xAA)
	g.commandGBM(CMD_0Fh_WRITE_TO_FLASH, 0x2AAA, 0x55)
	g.commandGBM(CMD_0Fh_WRITE_TO_FLASH, 0x5555, 0x77)
	g.commandGBM(CMD_0Fh_WRITE_TO_FLASH, 0x5555, 0xAA)
	g.commandGBM(CMD_0Fh_WRITE_TO_FLASH, 0x2AAA, 0x55)
	g.commandGBM(CMD_0Fh_WRITE_TO_FLASH, 0x5555, 0x77)

	// Read mapping
	err := g.Read256(0)
	if err != nil {
		return err
	}

	// Reset flash to leave hidden mapping area and disable port
	g.resetGBMFlash()

	_, err = w.Write(g.Buf[0:128])
	return err
}

func (g *GB) MapEntireROM() error {
	// Enable access to ports 0120h
	g.commandGBM(CMD_09h_WAKEUP_AND_RE_ENABLE_FLASH_REGS, 0, 0)
	// Map entire flashrom
	g.commandGBM(CMD_04h_MAP_ENTIRE_ROM, 0, 0)
	// Disable ports 0x0120h
	g.commandGBM(CMD_08h_DISABLE_FLASH_REGS, 0, 0)

	return nil
}
