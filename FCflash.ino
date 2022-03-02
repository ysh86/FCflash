// build:
// $ arduino-cli compile -b arduino:avr:micro --build-cache-path ./build --build-path ./build ../FCflash
//
// program:
// $ avrdude.exe -p m32u4 -c avr109 -P COM5 -U flash:w:./build/FCflash.ino.hex:i

#include <stdbool.h>
#include <stdint.h>

/******************************************
  Board
*****************************************/

// PRG
constexpr uint8_t OUT_A00 = 15;
constexpr uint8_t OUT_A01 = 14;
constexpr uint8_t OUT_A02 = 23;
constexpr uint8_t OUT_A03 = 22;
constexpr uint8_t OUT_A04 = 21;
constexpr uint8_t OUT_A05 = 20;
constexpr uint8_t OUT_A06 = 19;
constexpr uint8_t OUT_A07 = 18;

constexpr uint8_t OUT_A08 = 13;
constexpr uint8_t OUT_A09 = 17;
constexpr uint8_t OUT_A10 = 1;
constexpr uint8_t OUT_A11 = 0;
constexpr uint8_t OUT_A12 = 2;
constexpr uint8_t OUT_A13 = 3;
constexpr uint8_t OUT_A14 = 4;
constexpr uint8_t OUT_ROMSEL = 16;

constexpr uint8_t IO_D0 = 5;
constexpr uint8_t IO_D1 = 6;
constexpr uint8_t IO_D2 = 7;
constexpr uint8_t IO_D3 = 8;
constexpr uint8_t IO_D4 = 9;
constexpr uint8_t IO_D5 = 10;
constexpr uint8_t IO_D6 = 11;
constexpr uint8_t IO_D7 = 12;

// MMC/WRAM
constexpr uint8_t OUT_PHI2 = OUT_ROMSEL; // shared
constexpr uint8_t OUT_RW = OUT_A14;      // shared (Read:1, Write:0)
constexpr uint8_t OUT_RAMSEL = OUT_A13;  // shared (Enable:1, disable:0)

// CHR
// TODO: OUT_PPU_A13 -> /CS HIGH
constexpr uint8_t OUT_RD_NROM = OUT_A13; // shared
constexpr uint8_t OUT_RD      = OUT_A12; // shared
// TODO: OUT_WR = OUT_A14; // shared N.C.


/******************************************
  FC
*****************************************/
// Set Cartridge address
void setA00A14(uint16_t addr)
{
    digitalWrite(OUT_A13, (addr >> 13)&1); // shared with RAMSEL
    digitalWrite(OUT_A14, (addr >> 14)&1); // shared with CPU R/W

    __asm__(
        "nop\n\t"
        "nop\n\t"
    );

    digitalWrite(OUT_A00, addr & 1);
    digitalWrite(OUT_A01, (addr >> 1)&1);
    digitalWrite(OUT_A02, (addr >> 2)&1);
    digitalWrite(OUT_A03, (addr >> 3)&1);
    digitalWrite(OUT_A04, (addr >> 4)&1);
    digitalWrite(OUT_A05, (addr >> 5)&1);
    digitalWrite(OUT_A06, (addr >> 6)&1);
    digitalWrite(OUT_A07, (addr >> 7)&1);

    digitalWrite(OUT_A08, (addr >> 8 )&1);
    digitalWrite(OUT_A09, (addr >> 9 )&1);
    digitalWrite(OUT_A10, (addr >> 10)&1);
    digitalWrite(OUT_A11, (addr >> 11)&1);
    digitalWrite(OUT_A12, (addr >> 12)&1);

    __asm__(
        "nop\n\t"
        "nop\n\t"
    );
}

// Read one byte out of the cartridge
uint8_t readByte(uint8_t OUT_OE) {
    // Pull read low
    digitalWrite(OUT_OE, LOW);
    __asm__(
        "nop\n\t"
        "nop\n\t"
    );

    // read
    uint8_t temp = (
        digitalRead(IO_D0)      |
        digitalRead(IO_D1) << 1 |
        digitalRead(IO_D2) << 2 |
        digitalRead(IO_D3) << 3 |
        digitalRead(IO_D4) << 4 |
        digitalRead(IO_D5) << 5 |
        digitalRead(IO_D6) << 6 |
        digitalRead(IO_D7) << 7
    );


    // Pull read high
    digitalWrite(OUT_OE, HIGH);
    __asm__(
        "nop\n\t"
        "nop\n\t"
    );

    return temp;
}
void writeByte(uint8_t OUT_CE, uint8_t OUT_WE, uint8_t data) {
    digitalWrite(OUT_WE, LOW);
    __asm__(
        "nop\n\t"
        "nop\n\t"
    );

    pinMode(IO_D0, OUTPUT);
    pinMode(IO_D1, OUTPUT);
    pinMode(IO_D2, OUTPUT);
    pinMode(IO_D3, OUTPUT);
    pinMode(IO_D4, OUTPUT);
    pinMode(IO_D5, OUTPUT);
    pinMode(IO_D6, OUTPUT);
    pinMode(IO_D7, OUTPUT);

    // write
    digitalWrite(IO_D0, data & 1);
    digitalWrite(IO_D1, (data >> 1)&1);
    digitalWrite(IO_D2, (data >> 2)&1);
    digitalWrite(IO_D3, (data >> 3)&1);
    digitalWrite(IO_D4, (data >> 4)&1);
    digitalWrite(IO_D5, (data >> 5)&1);
    digitalWrite(IO_D6, (data >> 6)&1);
    digitalWrite(IO_D7, (data >> 7)&1);
    __asm__(
        "nop\n\t"
        "nop\n\t"
    );

    // Pull write low
    digitalWrite(OUT_CE, LOW);
    __asm__(
        "nop\n\t"
        "nop\n\t"
    );
    // Pull write high
    digitalWrite(OUT_CE, HIGH);
    __asm__(
        "nop\n\t"
        "nop\n\t"
    );

    digitalWrite(OUT_WE, HIGH);

    pinMode(IO_D0, INPUT_PULLUP);
    pinMode(IO_D1, INPUT_PULLUP);
    pinMode(IO_D2, INPUT_PULLUP);
    pinMode(IO_D3, INPUT_PULLUP);
    pinMode(IO_D4, INPUT_PULLUP);
    pinMode(IO_D5, INPUT_PULLUP);
    pinMode(IO_D6, INPUT_PULLUP);
    pinMode(IO_D7, INPUT_PULLUP);
}


/******************************************
  Host
*****************************************/
#define PACKET_SIZE (0x400)

// request
#define REQ_ECHO            0
#define REQ_PHI2_INIT       1
#define REQ_CPU_READ_6502   2
#define REQ_CPU_READ        3
#define REQ_CPU_WRITE_6502  4
#define REQ_CPU_WRITE_FLASH 5
#define REQ_PPU_READ        6
#define REQ_PPU_WRITE       7

// index
#define INDEX_IMPLIED 0
#define INDEX_CPU     1
#define INDEX_PPU     2
#define INDEX_BOTH    3

typedef struct Message {
    uint8_t  _reserved;
    uint8_t  request;
    uint16_t value;
    uint16_t index;
    uint16_t length;
} Message_t;


/******************************************
  Arduino
*****************************************/
void setup() {
    Serial.begin(115200);

    // init all pins to input/pull-up
    PORTB |= 0b11111111; // pull-up: B0-7
    DDRB &= ~0b11111111; // input: B0-7
    PORTC |= 0b11000000; // pull-up: C6,7
    DDRC &= ~0b11000000; // input: C6,7
    PORTD |= 0b11011111; // pull-up: D0-4,6,7
    DDRD &= ~0b11011111; // input: D0-4,6,7
    PORTE |= 0b01000000; // pull-up: E6
    DDRE &= ~0b01000000; // input: E6
    PORTF |= 0b11110011; // pull-up: F0,1,4-7
    DDRF &= ~0b11110011; // input: F0,1,4-7

    // Pins
    pinMode(OUT_A00, OUTPUT);
    pinMode(OUT_A01, OUTPUT);
    pinMode(OUT_A02, OUTPUT);
    pinMode(OUT_A03, OUTPUT);
    pinMode(OUT_A04, OUTPUT);
    pinMode(OUT_A05, OUTPUT);
    pinMode(OUT_A06, OUTPUT);
    pinMode(OUT_A07, OUTPUT);

    pinMode(OUT_A08, OUTPUT);
    pinMode(OUT_A09, OUTPUT);
    pinMode(OUT_A10, OUTPUT);
    pinMode(OUT_A11, OUTPUT);
    pinMode(OUT_A12, OUTPUT);
    pinMode(OUT_A13, OUTPUT);
    pinMode(OUT_A14, OUTPUT);
    pinMode(OUT_ROMSEL, OUTPUT);

    digitalWrite(OUT_RAMSEL, LOW);
    digitalWrite(OUT_ROMSEL, HIGH);
    digitalWrite(OUT_PHI2,   HIGH);
    digitalWrite(OUT_RW,     HIGH);

    //digitalWrite(OUT_PPU_A13, HIGH);
    //digitalWrite(OUT_RD_NROM, HIGH);
    digitalWrite(OUT_RD,      HIGH);
    //digitalWrite(OUT_WR,      HIGH);

    pinMode(IO_D0, INPUT_PULLUP);
    pinMode(IO_D1, INPUT_PULLUP);
    pinMode(IO_D2, INPUT_PULLUP);
    pinMode(IO_D3, INPUT_PULLUP);
    pinMode(IO_D4, INPUT_PULLUP);
    pinMode(IO_D5, INPUT_PULLUP);
    pinMode(IO_D6, INPUT_PULLUP);
    pinMode(IO_D7, INPUT_PULLUP);
}

#define PRG_BASE 0x8000
static uint8_t readbuf[PACKET_SIZE];


/******************************************
  main
*****************************************/

void readBytes(uint8_t OUT_OE, uint16_t addr, uint8_t buf[], uint16_t length) {
    for (uint16_t currByte = 0; currByte < length; currByte++) {
        noInterrupts();
        setA00A14(addr + currByte);
        buf[currByte] = readByte(OUT_OE);
        interrupts();
    }
}

void loop() {
    if (Serial.available() < 8) {
        return;
    }

    Message_t msg;
    Serial.readBytes((uint8_t *)&msg, sizeof(msg));

    uint16_t addr = msg.value;
    if (msg.request == REQ_CPU_READ) {
        // addr
        // NROM:  0b1xxx_xxxx... 32KB full
        // TxROM: 0b110x_xxxx... 8KB  0xc000-0xdfff
        addr = PRG_BASE | addr;
        if (msg.length <= PACKET_SIZE) {
            readBytes(OUT_ROMSEL, addr, readbuf, msg.length);
            Serial.write(readbuf, msg.length);
        }
        return;
    }
    if (msg.request == REQ_CPU_WRITE_6502) {
        // addr
        // 0b1001_xxxx... 0x9000-0x9fff only
        addr = PRG_BASE | (addr & ~0x6000) | 0x1000;
        uint8_t data = msg.length & 0xff;
        noInterrupts();
        setA00A14(addr);
        writeByte(OUT_ROMSEL, OUT_RW, data);
        interrupts();
        return;
    }
    if (msg.request == REQ_PPU_READ) {
        // addr
        // NROM:  0b0x1x_xxxx... 8KB full
        // TxROM: 0b0x00_xxxx... 4KB first half only
        addr = (addr & 0x2000) ? (0x4000 | addr) : (0x5000 | addr);
        uint8_t OUT_OE = (addr & 0x2000) ? OUT_RD_NROM : OUT_RD;
        if (msg.length <= PACKET_SIZE) {
            readBytes(OUT_OE, addr, readbuf, msg.length);
            Serial.write(readbuf, msg.length);
        }
        return;
    }
}
