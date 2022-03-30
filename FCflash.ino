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

// PRG/CHR
constexpr uint8_t OUT_A01_07_CLR = 22; // PF1: All LOW:1
constexpr uint8_t OUT_A00 = 23;        // PF0: next:neg edge

constexpr uint8_t OUT_A08 = 17;        // PB0:
constexpr uint8_t OUT_A09 = 15;        // PB1:
constexpr uint8_t OUT_A10 = 16;        // PB2:
constexpr uint8_t OUT_A11 = 14;        // PB3:
constexpr uint8_t OUT_A12 = 8;         // PB4:
constexpr uint8_t OUT_CPU_A13 = 9;     // PB5:
constexpr uint8_t OUT_PPU_A13 = 4;     // PD4:
constexpr uint8_t OUT_CPU_A14 = 10;    // PB6:
constexpr uint8_t OUT_ROMSEL = 11;     // PB7:

constexpr uint8_t IO_D0 = 3;           // PD0:
constexpr uint8_t IO_D1 = 2;           // PD1:
constexpr uint8_t IO_D2 = 0;           // PD2:
constexpr uint8_t IO_D3 = 1;           // PD3:
constexpr uint8_t IO_D4 = 21;          // PF4:
constexpr uint8_t IO_D5 = 20;          // PF5:
constexpr uint8_t IO_D6 = 19;          // PF6:
constexpr uint8_t IO_D7 = 18;          // PF7:

// MMC/W-RAM
constexpr uint8_t OUT_PHI2 = 7;        // PE6:
constexpr uint8_t OUT_CPU_RW = 6;      // PD7: Read:1, Write:0

// CHR
// TODO: OUT_PPU_WR = 13;              // PC7: N.C.
constexpr uint8_t OUT_PPU_RD = 5;      // PC6:

#define ROMSEL(__b__) (PORTB = (PORTB&0b01111111)|((__b__)<<7))
#define PHI2(__b__) (PORTE = (PORTE&0b10111111)|((__b__)<<6))

// EEPROM
constexpr uint8_t EEP_OUT_PRG_CE = 13;    // PC7
constexpr uint8_t EEP_OPEN_DRAIN_WE = 12; // PD6: open-drain


/******************************************
  FC
*****************************************/
// Set Cartridge address
void clearA00A07()
{
    digitalWrite(OUT_A00, LOW);
    digitalWrite(OUT_A01_07_CLR, HIGH);
    __asm__(
        "nop\n\t"
        "nop\n\t"
        "nop\n\t"
        "nop\n\t"
    );
    digitalWrite(OUT_A01_07_CLR, LOW);
    __asm__(
        "nop\n\t"
        "nop\n\t"
        "nop\n\t"
        "nop\n\t"
    );
}
void nextA00A07(uint16_t lo_addr)
{
    PORTF = (PORTF & 0b11111110) | (lo_addr&1);
}
void setA08A14(uint16_t hi_addr)
{
    PORTB = (PORTB & 0x80) | ((hi_addr >> 8) & 0x7f);
    __asm__(
        "nop\n\t"
        "nop\n\t"
    );
}

// Read one byte out of the cartridge
uint8_t readByte(uint8_t OUT_OE) {
    // already disabled all chips (PRG, W-RAM & CHR)
    if (OUT_OE == OUT_ROMSEL) {
        // PRG
        ROMSEL(0); // select chip
        PHI2(1);   // enable read & set addr
    } else {
        // CHR
        digitalWrite(OUT_OE, LOW); // enable read
    }
    __asm__(
        "nop\n\t"
        "nop\n\t"
        "nop\n\t"
    );

    // read
    uint8_t temp = (PINF & 0xf0) | (PIND & 0x0f);

    if (OUT_OE == OUT_ROMSEL) {
        // PRG
        PHI2(0);
        ROMSEL(1);
    } else {
        // CHR
        digitalWrite(OUT_OE, HIGH);
    }
    __asm__(
        "nop\n\t"
    );

    return temp;
}
void writeByte(uint8_t OUT_CS, uint8_t OUT_WE, uint8_t data) {
    // already disabled both chips (MMC & W-RAM)
    digitalWrite(OUT_WE, LOW); // enable write

    // write
    DDRD |= 0x0f;
    DDRF |= 0xf0;
    PORTD = (PORTD & 0xf0) | (data & 0x0f);
    PORTF = (PORTF & 0x0f) | (data & 0xf0);

    ROMSEL(0); // select MMC:0 or W-RAM:1
    PHI2(1);   // enable that chip & set addr
    __asm__(
        "nop\n\t"
        "nop\n\t"
    );

    // latch
    PHI2(0);
    ROMSEL(1);

    DDRD &= ~0x0f;
    DDRF &= ~0xf0;

    digitalWrite(OUT_WE, HIGH);
}
void writeEEP(uint8_t OUT_CE, uint16_t addr, uint8_t buf[], uint16_t length) {
    // I/O pins: output
    DDRD |= 0x0f;
    DDRF |= 0xf0;

    digitalWrite(OUT_CE, LOW); // enable chip
    clearA00A07();
    for (uint16_t currByte = 0; currByte < length; currByte += 64) {
        for (uint16_t i = 0; i < 64; i++) {
            uint8_t data = buf[currByte + i];

            noInterrupts();
            nextA00A07(currByte + i);
            setA08A14(addr + currByte + i);

            pinMode(EEP_OPEN_DRAIN_WE, OUTPUT); // enable write

            // write
            PORTD = (PORTD & 0xf0) | (data & 0x0f);
            PORTF = (PORTF & 0x0f) | (data & 0xf0);
            __asm__(
                "nop\n\t"
                "nop\n\t"
            );

            pinMode(EEP_OPEN_DRAIN_WE, INPUT);
            __asm__(
                "nop\n\t"
                "nop\n\t"
            );

            interrupts();
        }
        delay(5); // [msec]
    }
    digitalWrite(OUT_CE, HIGH);

    // I/O pins: input/pull-up
    PORTD |= 0x0f;
    DDRD &= ~0x0f;
    PORTF |= 0xf0;
    DDRF &= ~0xf0;
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

#define REQ_CPU_WRITE_EEP  16
#define REQ_PPU_WRITE_EEP  17

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
    pinMode(OUT_A01_07_CLR, OUTPUT);
    pinMode(OUT_A00, OUTPUT);

    pinMode(OUT_A08, OUTPUT);
    pinMode(OUT_A09, OUTPUT);
    pinMode(OUT_A10, OUTPUT);
    pinMode(OUT_A11, OUTPUT);
    pinMode(OUT_A12, OUTPUT);
    pinMode(OUT_CPU_A13, OUTPUT);
    pinMode(OUT_PPU_A13, OUTPUT);
    pinMode(OUT_CPU_A14, OUTPUT);
    pinMode(OUT_ROMSEL, OUTPUT);

    pinMode(OUT_PHI2, OUTPUT);
    pinMode(OUT_CPU_RW, OUTPUT);
    //pinMode(OUT_PPU_WR, OUTPUT);
    pinMode(OUT_PPU_RD, OUTPUT);

    pinMode(EEP_OUT_PRG_CE, OUTPUT);
    digitalWrite(EEP_OUT_PRG_CE, HIGH);
    digitalWrite(EEP_OPEN_DRAIN_WE, LOW); // open-drain
    pinMode(EEP_OPEN_DRAIN_WE, INPUT);

    clearA00A07();
    setA08A14(0);

    digitalWrite(OUT_CPU_RW, HIGH);
    digitalWrite(OUT_ROMSEL, HIGH);
    digitalWrite(OUT_PHI2,   LOW);

    digitalWrite(OUT_PPU_A13, HIGH);
    //digitalWrite(OUT_PPU_WR,  HIGH);
    digitalWrite(OUT_PPU_RD,  HIGH);

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
    clearA00A07();
    for (uint16_t currByte = 0; currByte < length; currByte++) {
        noInterrupts();
        nextA00A07(currByte);
        setA08A14(addr + currByte);
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
        // addr: 0b1xxx_xxxx... 32KB full
        addr = PRG_BASE | addr;
        if (msg.length <= PACKET_SIZE) {
            readBytes(OUT_ROMSEL, addr, readbuf, msg.length);
            Serial.write(readbuf, msg.length);
        }
        return;
    }
    if (msg.request == REQ_CPU_WRITE_6502) {
        // addr: 0b100x_xxxx... 0x8000-0x9fff only
        addr = PRG_BASE | (addr & ~0x6000);
        uint8_t data = msg.length & 0xff;
        clearA00A07();
        noInterrupts();
        nextA00A07(addr&1);
        setA08A14(addr);
        writeByte(OUT_ROMSEL, OUT_CPU_RW, data);
        interrupts();
        return;
    }
    if (msg.request == REQ_PPU_READ) {
        // addr: 0b000x_xxxx... 8KB full
        addr &= 0x1fff;
        digitalWrite(OUT_PPU_A13, LOW);
        if (msg.length <= PACKET_SIZE) {
            readBytes(OUT_PPU_RD, addr, readbuf, msg.length);
            Serial.write(readbuf, msg.length);
        }
        digitalWrite(OUT_PPU_A13, HIGH);
        return;
    }

    // EEPROM
    if (msg.request == REQ_CPU_WRITE_EEP) {
        // addr: 0b1xxx_xxxx... 32KB full
        addr = PRG_BASE | addr;
        if (msg.length <= PACKET_SIZE) {
            Serial.readBytes(readbuf, msg.length);
            writeEEP(EEP_OUT_PRG_CE, addr, readbuf, msg.length);
        }
        return;
    }
    if (msg.request == REQ_PPU_WRITE_EEP) {
        // addr: 0b000x_xxxx... 8KB full
        addr &= 0x1fff;
        if (msg.length <= PACKET_SIZE) {
            Serial.readBytes(readbuf, msg.length);
            writeEEP(OUT_PPU_A13, addr, readbuf, msg.length);
        }
        return;
    }
}
