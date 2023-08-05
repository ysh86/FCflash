package FCflash

const PACKET_SIZE = 0x400

type Request uint8

const (
	REQ_ECHO Request = iota + 0
	REQ_PHI2_INIT
	REQ_CPU_READ_6502
	REQ_CPU_READ
	REQ_CPU_WRITE_6502
	REQ_CPU_WRITE_6502_5BITS
	REQ_PPU_READ
	REQ_PPU_WRITE
)
const (
	REQ_CPU_WRITE_EEP Request = iota + 16
	REQ_PPU_WRITE_EEP
	REQ_CPU_WRITE_FLASH
)
const (
	REQ_RAW_READ Request = iota + 32
	REQ_RAW_READ_LO
	REQ_RAW_WRITE
	REQ_RAW_WRITE_LO
	REQ_RAW_WRITE_WO_CS
)
const (
	REQ_RAW_ERASE_FLASH Request = iota + 64
	REQ_RAW_WRITE_FLASH
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
