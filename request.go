package FCflash

const PACKET_SIZE = 0x400

type Request uint8

const (
	REQ_ECHO Request = iota
	REQ_PHI2_INIT
	REQ_CPU_READ_6502
	REQ_CPU_READ
	REQ_CPU_WRITE_6502
	REQ_CPU_WRITE_6502_5BITS
	REQ_PPU_READ
	REQ_PPU_WRITE

	REQ_CPU_WRITE_EEP   = 16
	REQ_PPU_WRITE_EEP   = 17
	REQ_CPU_WRITE_FLASH = 18

	REQ_RAW_READ        = 32
	REQ_RAW_READ_LO     = 33
	REQ_RAW_WRITE       = 34
	REQ_RAW_WRITE_LO    = 35
	REQ_RAW_WRITE_WO_CS = 36

	REQ_RAW_ERASE_FLASH = 64
	REQ_RAW_WRITE_FLASH = 65
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
