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

// Raw
constexpr uint8_t RAW_OUT_OE = 12; // PD6: open-drain
//constexpr uint8_t RAW_OUT_OE = 5;  // Mask ROM
constexpr uint8_t RAW_OUT_WE = 13; // PC7
constexpr uint8_t RAW_OUT_CE = 11; // shared with ROMSEL
constexpr uint8_t RAW_OUT_A15 = 4; // shared with PPU_A13
constexpr uint8_t RAW_OUT_A16 = 5; // shared with PPU_RD
//constexpr uint8_t RAW_OUT_A16 = 13;// Mask ROM
constexpr uint8_t RAW_OUT_A17 = 6; // shared with CPU_RW (swapped addr)
constexpr uint8_t RAW_OUT_A18 = 7; // shared with PHI2   (swapped addr)


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
void setA15A18(uint32_t addr)
{
    uint8_t nibb = (addr >> 15) & 0x0f;
    digitalWrite(RAW_OUT_A15, nibb&1);
    digitalWrite(RAW_OUT_A16, (nibb>>1)&1);
    digitalWrite(RAW_OUT_A17, (nibb>>2)&1);
    digitalWrite(RAW_OUT_A18, (nibb>>3)&1);
}

// Read one byte out of the cartridge
uint8_t readByte(uint8_t OUT_OE) {
    // already disabled all chips (PRG, W-RAM & CHR)
    if (OUT_OE == OUT_ROMSEL) {
        // PRG
        ROMSEL(0); // select chip
        PHI2(1);   // enable read & set addr
    } else {
        // CHR, RAW
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
        // CHR, RAW
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
        delay(7); // [msec]
    }
    digitalWrite(OUT_CE, HIGH);

    // I/O pins: input/pull-up
    PORTD |= 0x0f;
    DDRD &= ~0x0f;
    PORTF |= 0xf0;
    DDRF &= ~0xf0;
}

// Raw
void writeRawEEP(uint8_t OUT_CE, uint16_t addr, uint8_t buf[], uint16_t length) {
    writeRawEEP0(OUT_CE, addr, buf, length);
    //writeRawEEP1(OUT_CE, addr, buf, length);
}
void writeRawEEP0(uint8_t OUT_CE, uint16_t addr, uint8_t buf[], uint16_t length) {
    // I/O pins: output
    DDRD |= 0x0f;
    DDRF |= 0xf0;

#if 0
    // NROM cart
    digitalWrite(OUT_CE, LOW); // enable chip
#else
    // Raw
    digitalWrite(RAW_OUT_CE, LOW); // enable chip
#endif
    clearA00A07();
    for (uint16_t currByte = 0; currByte < length; currByte += 64) {
        for (uint16_t i = 0; i < 64; i++) {
            uint8_t data = buf[currByte + i];

            noInterrupts();
            nextA00A07(currByte + i);
            setA08A14(addr + currByte + i);

#if 0
            // NROM cart
            pinMode(EEP_OPEN_DRAIN_WE, OUTPUT); // enable write
#else
            // Raw
            digitalWrite(EEP_OPEN_DRAIN_WE, LOW); // enable write
            pinMode(EEP_OPEN_DRAIN_WE, OUTPUT);
#endif

            // write
            PORTD = (PORTD & 0xf0) | (data & 0x0f);
            PORTF = (PORTF & 0x0f) | (data & 0xf0);
            __asm__(
                "nop\n\t"
                "nop\n\t"
            );

#if 0
            // NROM cart
            pinMode(EEP_OPEN_DRAIN_WE, INPUT);
#else
            // Raw
            digitalWrite(EEP_OPEN_DRAIN_WE, HIGH);
#endif
            __asm__(
                "nop\n\t"
                "nop\n\t"
            );

            interrupts();
        }
        delay(7); // [msec]
    }
#if 0
    // NROM cart
    digitalWrite(OUT_CE, HIGH);
#else
    // Raw
    digitalWrite(RAW_OUT_CE, HIGH);
#endif

    // I/O pins: input/pull-up
    PORTD |= 0x0f;
    DDRD &= ~0x0f;
    PORTF |= 0xf0;
    DDRF &= ~0xf0;
}
void writeRawEEP1(uint8_t OUT_CE, uint16_t addr, uint8_t buf[], uint16_t length) {
    // I/O pins: output
    DDRD |= 0x0f;
    DDRF |= 0xf0;

#if 0
    // NROM cart
    digitalWrite(OUT_CE, LOW); // enable chip
#else
    // Raw
    digitalWrite(RAW_OUT_CE, LOW); // enable chip
#endif
    clearA00A07();
    for (uint16_t a = 1; a <= 64; a++) {
        nextA00A07(a);
    }
    for (uint16_t currByte = 64; currByte < length; currByte += 64) {
        for (uint16_t i = 0; i < 64; i++) {
            uint8_t data = buf[currByte + i];

            noInterrupts();
            nextA00A07(currByte + i);
            setA08A14(addr + currByte + i);

#if 0
            // NROM cart
            pinMode(EEP_OPEN_DRAIN_WE, OUTPUT); // enable write
#else
            // Raw
            digitalWrite(EEP_OPEN_DRAIN_WE, LOW); // enable write
            pinMode(EEP_OPEN_DRAIN_WE, OUTPUT);
#endif

            // write
            PORTD = (PORTD & 0xf0) | (data & 0x0f);
            PORTF = (PORTF & 0x0f) | (data & 0xf0);
            __asm__(
                "nop\n\t"
                "nop\n\t"
            );

#if 0
            // NROM cart
            pinMode(EEP_OPEN_DRAIN_WE, INPUT);
#else
            // Raw
            digitalWrite(EEP_OPEN_DRAIN_WE, HIGH);
#endif
            __asm__(
                "nop\n\t"
                "nop\n\t"
            );

            interrupts();
        }
        delay(7); // [msec]
    }
#if 0
    // NROM cart
    digitalWrite(OUT_CE, HIGH);
#else
    // Raw
    digitalWrite(RAW_OUT_CE, HIGH);
#endif

    // I/O pins: input/pull-up
    PORTD |= 0x0f;
    DDRD &= ~0x0f;
    PORTF |= 0xf0;
    DDRF &= ~0xf0;
}

// addr7: A18-12:7bits => 4[KB/sector] * 128 = max 512KB(4Mbits)
void eraseFlash(uint8_t OUT_CE, uint8_t addr7) {
    // I/O pins: output
    DDRD |= 0x0f;
    DDRF |= 0xf0;

    // commands
    {
        uint8_t data;
        noInterrupts();

        // 5555H, AAH
        clearA00A07();
        for (uint16_t a = 1; a <= 0x55; a++) {
            nextA00A07(a);
        }
        setA08A14(0x5500);
        data = 0xaa;
        digitalWrite(OUT_CE, LOW);
        PORTD = (PORTD & 0xf0) | (data & 0x0f);
        PORTF = (PORTF & 0x0f) | (data & 0xf0);
        __asm__(
            "nop\n\t"
            //"nop\n\t" // EEPROM
        );
        digitalWrite(OUT_CE, HIGH);
        __asm__(
            "nop\n\t"
            //"nop\n\t" // EEPROM
        );

        // 2AAAH, 55H
        clearA00A07();
        for (uint16_t a = 1; a <= 0xaa; a++) {
            nextA00A07(a);
        }
        setA08A14(0x2a00);
        data = 0x55;
        digitalWrite(OUT_CE, LOW);
        PORTD = (PORTD & 0xf0) | (data & 0x0f);
        PORTF = (PORTF & 0x0f) | (data & 0xf0);
        __asm__(
            "nop\n\t"
            //"nop\n\t" // EEPROM
        );
        digitalWrite(OUT_CE, HIGH);
        __asm__(
            "nop\n\t"
            //"nop\n\t" // EEPROM
        );

        // 5555H, 80H
        clearA00A07();
        for (uint16_t a = 1; a <= 0x55; a++) {
            nextA00A07(a);
        }
        setA08A14(0x5500);
        data = 0x80;
        digitalWrite(OUT_CE, LOW);
        PORTD = (PORTD & 0xf0) | (data & 0x0f);
        PORTF = (PORTF & 0x0f) | (data & 0xf0);
        __asm__(
            "nop\n\t"
            //"nop\n\t" // EEPROM
        );
        digitalWrite(OUT_CE, HIGH);
        __asm__(
            "nop\n\t"
            //"nop\n\t" // EEPROM
        );

        // 5555H, AAH
        clearA00A07();
        for (uint16_t a = 1; a <= 0x55; a++) {
            nextA00A07(a);
        }
        setA08A14(0x5500);
        data = 0xaa;
        digitalWrite(OUT_CE, LOW);
        PORTD = (PORTD & 0xf0) | (data & 0x0f);
        PORTF = (PORTF & 0x0f) | (data & 0xf0);
        __asm__(
            "nop\n\t"
            //"nop\n\t" // EEPROM
        );
        digitalWrite(OUT_CE, HIGH);
        __asm__(
            "nop\n\t"
            //"nop\n\t" // EEPROM
        );

        // 2AAAH, 55H
        clearA00A07();
        for (uint16_t a = 1; a <= 0xaa; a++) {
            nextA00A07(a);
        }
        setA08A14(0x2a00);
        data = 0x55;
        digitalWrite(OUT_CE, LOW);
        PORTD = (PORTD & 0xf0) | (data & 0x0f);
        PORTF = (PORTF & 0x0f) | (data & 0xf0);
        __asm__(
            "nop\n\t"
            //"nop\n\t" // EEPROM
        );
        digitalWrite(OUT_CE, HIGH);
        __asm__(
            "nop\n\t"
            //"nop\n\t" // EEPROM
        );

        if (addr7 == 0xff) {
            // Chip-Erase: 5555H, 10H
            clearA00A07();
            for (uint16_t a = 1; a <= 0x55; a++) {
                nextA00A07(a);
            }
            setA08A14(0x5500);
            data = 0x10; // Flash
            //data = 0x20; // EEPROM
        } else {
            // Sector-Erase: sector(A18-12), 30H
            uint32_t sector = (uint32_t)addr7 << 12;
            clearA00A07();
            setA08A14(sector&0x7fff);
            data = 0x30;
        }
        digitalWrite(OUT_CE, LOW);
        PORTD = (PORTD & 0xf0) | (data & 0x0f);
        PORTF = (PORTF & 0x0f) | (data & 0xf0);
        __asm__(
            "nop\n\t"
            //"nop\n\t" // EEPROM
        );
        digitalWrite(OUT_CE, HIGH);
        __asm__(
            "nop\n\t"
            //"nop\n\t" // EEPROM
        );

        interrupts();
    }

    if (addr7 == 0xff) {
        // Chip-Erase
        delay(100); // [msec]
    } else {
        // Sector-Erase
        delay(25); // [msec]
    }

    // I/O pins: input/pull-up
    PORTD |= 0x0f;
    DDRD &= ~0x0f;
    PORTF |= 0xf0;
    DDRF &= ~0xf0;
}
void writeFlash(uint8_t OUT_CE, uint16_t addr15, uint8_t buf[], uint16_t length) {
    // erase @ every 4KB
    if ((addr15 & 0x0fff) == 0) {
        eraseFlash(OUT_CE, addr15>>12);
    }

    // I/O pins: output
    DDRD |= 0x0f;
    DDRF |= 0xf0;

    for (uint16_t currByte = 0; currByte < length; currByte++) {
        uint8_t data;
        noInterrupts();

        // 5555H, AAH
        clearA00A07();
        for (uint16_t a = 1; a <= 0x55; a++) {
            nextA00A07(a);
        }
        setA08A14(0x5500);
        data = 0xaa;
        digitalWrite(OUT_CE, LOW);
        PORTD = (PORTD & 0xf0) | (data & 0x0f);
        PORTF = (PORTF & 0x0f) | (data & 0xf0);
        __asm__(
            "nop\n\t"
        );
        digitalWrite(OUT_CE, HIGH);
        __asm__(
            "nop\n\t"
        );

        // 2AAAH, 55H
        clearA00A07();
        for (uint16_t a = 1; a <= 0xaa; a++) {
            nextA00A07(a);
        }
        setA08A14(0x2a00);
        data = 0x55;
        digitalWrite(OUT_CE, LOW);
        PORTD = (PORTD & 0xf0) | (data & 0x0f);
        PORTF = (PORTF & 0x0f) | (data & 0xf0);
        __asm__(
            "nop\n\t"
        );
        digitalWrite(OUT_CE, HIGH);
        __asm__(
            "nop\n\t"
        );

        // 5555H, A0H
        clearA00A07();
        for (uint16_t a = 1; a <= 0x55; a++) {
            nextA00A07(a);
        }
        setA08A14(0x5500);
        data = 0xA0;
        digitalWrite(OUT_CE, LOW);
        PORTD = (PORTD & 0xf0) | (data & 0x0f);
        PORTF = (PORTF & 0x0f) | (data & 0xf0);
        __asm__(
            "nop\n\t"
        );
        digitalWrite(OUT_CE, HIGH);
        __asm__(
            "nop\n\t"
        );

        // addr15, Data
        uint16_t addr = addr15 + currByte;
        clearA00A07();
        for (uint16_t a = 1; a <= (addr&0xff); a++) {
            nextA00A07(a);
        }
        setA08A14(addr&0x7fff);
        data = buf[currByte];
        digitalWrite(OUT_CE, LOW);
        PORTD = (PORTD & 0xf0) | (data & 0x0f);
        PORTF = (PORTF & 0x0f) | (data & 0xf0);
        __asm__(
            "nop\n\t"
        );
        digitalWrite(OUT_CE, HIGH);
        __asm__(
            "nop\n\t"
        );

        interrupts();

        delayMicroseconds(20);
    }

    // I/O pins: input/pull-up
    PORTD |= 0x0f;
    DDRD &= ~0x0f;
    PORTF |= 0xf0;
    DDRF &= ~0xf0;
}
void writeRaw(uint8_t OUT_WE, uint32_t addr24, uint8_t buf[], uint16_t length) {
    uint16_t lo_addr = addr24 & 0xff;

    // I/O pins: output
    DDRD |= 0x0f;
    DDRF |= 0xf0;

    clearA00A07();
    for (uint16_t a = 1; a <= lo_addr; a++) {
        nextA00A07(a);
    }
    for (uint16_t currByte = 0; currByte < length; currByte++) {
        uint8_t data = buf[currByte];

        noInterrupts();
        nextA00A07(lo_addr + currByte);
        setA08A14(addr24 + currByte);

        digitalWrite(OUT_WE, LOW);
        PORTD = (PORTD & 0xf0) | (data & 0x0f);
        PORTF = (PORTF & 0x0f) | (data & 0xf0);
        __asm__(
            "nop\n\t"
            "nop\n\t"
        );
        digitalWrite(OUT_WE, HIGH);
        __asm__(
            "nop\n\t"
            "nop\n\t"
        );

        interrupts();
    }

    // I/O pins: input/pull-up
    PORTD |= 0x0f;
    DDRD &= ~0x0f;
    PORTF |= 0xf0;
    DDRF &= ~0xf0;
}
void writeRegs(uint8_t OUT_WE, uint8_t buf[], uint16_t length) {
    // I/O pins: output
    DDRD |= 0x0f;
    DDRF |= 0xf0;

    for (uint16_t currByte = 0; currByte < length; currByte += 4) {
        uint16_t addr15 = buf[currByte] << 8;
        uint16_t addr8 = buf[currByte+1];
        uint8_t data = buf[currByte+2];
        noInterrupts();

        clearA00A07();
        for (uint16_t a = 1; a <= addr8; a++) {
            nextA00A07(a);
        }
        setA08A14(addr15);

        digitalWrite(OUT_WE, LOW);
        PORTD = (PORTD & 0xf0) | (data & 0x0f);
        PORTF = (PORTF & 0x0f) | (data & 0xf0);
        __asm__(
            "nop\n\t"
            "nop\n\t"
            "nop\n\t"
            "nop\n\t"
        );
        digitalWrite(OUT_WE, HIGH);
        __asm__(
            "nop\n\t"
            "nop\n\t"
            "nop\n\t"
            "nop\n\t"
        );

        interrupts();
    }

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
#define REQ_ECHO                 0
#define REQ_PHI2_INIT            1
#define REQ_CPU_READ_6502        2
#define REQ_CPU_READ             3
#define REQ_CPU_WRITE_6502       4
#define REQ_CPU_WRITE_6502_5BITS 5
#define REQ_PPU_READ             6
#define REQ_PPU_WRITE            7

#define REQ_CPU_WRITE_EEP       16
#define REQ_PPU_WRITE_EEP       17
#define REQ_CPU_WRITE_FLASH     18

#define REQ_RAW_READ            32
#define REQ_RAW_READ_LO         33
#define REQ_RAW_WRITE           34
#define REQ_RAW_WRITE_LO        35
#define REQ_RAW_READ_WO_CS      36
#define REQ_RAW_WRITE_WO_CS     37
#define REQ_RAW_WRITE_LO_WO_CS  38

#define REQ_RAW_ERASE_FLASH     48
#define REQ_RAW_WRITE_FLASH     49

#define REQ_GBM_WRITE_REGS      64

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


/******************************************
  main
*****************************************/
#define PRG_BASE 0x8000
static uint8_t readbuf[PACKET_SIZE];

void readBytes(uint8_t OUT_OE, uint32_t addr24, uint8_t buf[], uint16_t length) {
    uint16_t lo_addr = addr24 & 0xff;

    clearA00A07();
    for (uint16_t a = 1; a <= lo_addr; a++) {
        nextA00A07(a);
    }
    for (uint16_t currByte = 0; currByte < length; currByte++) {
        noInterrupts();
        nextA00A07(lo_addr + currByte);
        setA08A14(addr24 + currByte);
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
    if (msg.request == REQ_CPU_WRITE_6502_5BITS) {
        // addr: 0x8000-0xffff full
        addr = PRG_BASE | addr;
        uint8_t five = msg.length & 0x1f;
        clearA00A07();
        noInterrupts();
        nextA00A07(addr&1);
        setA08A14(addr);
        // phi2 pulse(L->H->L) is neccessary for MMC1.
        {
            ROMSEL(0); // select MMC:0 or W-RAM:1
            PHI2(1);
            __asm__(
                "nop\n\t"
                "nop\n\t"
            );
            PHI2(0);
            ROMSEL(1);
        }
        for (int i = 0; i < 5; i++) {
            writeByte(OUT_ROMSEL, OUT_CPU_RW, five&1);
            five >>= 1;
        }
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

    // Flash
    if (msg.request == REQ_CPU_WRITE_FLASH) {
        // addr15:
        //  even banks: 0x0000-0x3fff 16KB
        //  odd  banks: 0x4000-0x7fff 16KB
        digitalWrite(OUT_CPU_RW, LOW);
        if (msg.length <= PACKET_SIZE) {
            Serial.readBytes(readbuf, msg.length);
            writeFlash(OUT_ROMSEL, addr, readbuf, msg.length);
        }
        digitalWrite(OUT_CPU_RW, HIGH);
        return;
    }

    // RAW
    if (msg.request == REQ_RAW_READ) {
        // addr24: 16bit + zero 8bit = 16MB
        uint32_t addr24 = (uint32_t)addr << 8;
        setA15A18(addr24);
        pinMode(RAW_OUT_OE, OUTPUT); // open-drain -> out
        digitalWrite(RAW_OUT_OE, HIGH);
        digitalWrite(RAW_OUT_CE, LOW);
        if (msg.length <= PACKET_SIZE) {
            readBytes(RAW_OUT_OE, addr24, readbuf, msg.length);
            Serial.write(readbuf, msg.length);
        }
        digitalWrite(RAW_OUT_CE, HIGH);
        return;
    }
    if (msg.request == REQ_RAW_READ_LO) {
        setA15A18(addr);
        pinMode(RAW_OUT_OE, OUTPUT); // open-drain -> out
        digitalWrite(RAW_OUT_OE, HIGH);
        digitalWrite(RAW_OUT_CE, LOW);
        if (msg.length <= PACKET_SIZE) {
            readBytes(RAW_OUT_OE, addr, readbuf, msg.length);
            Serial.write(readbuf, msg.length);
        }
        digitalWrite(RAW_OUT_CE, HIGH);
        return;
    }
    if (msg.request == REQ_RAW_WRITE) {
        // addr24: 16bit + zero 8bit = 16MB
        uint32_t addr24 = (uint32_t)addr << 8;
        setA15A18(addr24);
        pinMode(RAW_OUT_OE, OUTPUT); // open-drain -> out
        digitalWrite(RAW_OUT_OE, HIGH);
        digitalWrite(RAW_OUT_CE, LOW);
        if (msg.length <= PACKET_SIZE) {
            Serial.readBytes(readbuf, msg.length);
            writeRaw(RAW_OUT_WE, addr24, readbuf, msg.length);
        }
        digitalWrite(RAW_OUT_CE, HIGH);
        return;
    }
    if (msg.request == REQ_RAW_WRITE_LO) {
        setA15A18(addr);
        pinMode(RAW_OUT_OE, OUTPUT); // open-drain -> out
        digitalWrite(RAW_OUT_OE, HIGH);
        digitalWrite(RAW_OUT_CE, LOW);
        if (msg.length <= PACKET_SIZE) {
            Serial.readBytes(readbuf, msg.length);
            writeRaw(RAW_OUT_WE, addr, readbuf, msg.length);
        }
        digitalWrite(RAW_OUT_CE, HIGH);
        return;
    }
    if (msg.request == REQ_RAW_READ_WO_CS) {
        // addr24: 16bit + zero 8bit = 16MB
        uint32_t addr24 = (uint32_t)addr << 8;
        setA15A18(addr24);
        pinMode(RAW_OUT_OE, OUTPUT); // open-drain -> out
        digitalWrite(RAW_OUT_OE, HIGH);
        if (msg.length <= PACKET_SIZE) {
            readBytes(RAW_OUT_OE, addr24, readbuf, msg.length);
            Serial.write(readbuf, msg.length);
        }
        return;
    }
    if (msg.request == REQ_RAW_WRITE_WO_CS) {
        // addr24: 16bit + zero 8bit = 16MB
        uint32_t addr24 = (uint32_t)addr << 8;
        setA15A18(addr24);
        pinMode(RAW_OUT_OE, OUTPUT); // open-drain -> out
        digitalWrite(RAW_OUT_OE, HIGH);
        if (msg.length <= PACKET_SIZE) {
            Serial.readBytes(readbuf, msg.length);
            writeRaw(RAW_OUT_WE, addr24, readbuf, msg.length);
        }
        return;
    }
    if (msg.request == REQ_RAW_WRITE_LO_WO_CS) {
        setA15A18(addr);
        pinMode(RAW_OUT_OE, OUTPUT); // open-drain -> out
        digitalWrite(RAW_OUT_OE, HIGH);
        if (msg.length <= PACKET_SIZE) {
            Serial.readBytes(readbuf, msg.length);
            writeRaw(RAW_OUT_WE, addr, readbuf, msg.length);
        }
        return;
    }
    // Raw Flash
    if (msg.request == REQ_RAW_ERASE_FLASH) {
        pinMode(RAW_OUT_OE, OUTPUT); // open-drain -> out
        digitalWrite(RAW_OUT_OE, HIGH);
        digitalWrite(RAW_OUT_CE, LOW);
        {
            eraseFlash(RAW_OUT_WE, addr/* == 0xff */);
        }
        digitalWrite(RAW_OUT_CE, HIGH);
        return;
    }
    if (msg.request == REQ_RAW_WRITE_FLASH) {
        // addr24: 16bit + zero 8bit = 16MB
        uint32_t addr24 = (uint32_t)addr << 8;
        setA15A18(addr24);
        pinMode(RAW_OUT_OE, OUTPUT); // open-drain -> out
        digitalWrite(RAW_OUT_OE, HIGH);
        digitalWrite(RAW_OUT_CE, LOW);
        if (msg.length <= PACKET_SIZE) {
            Serial.readBytes(readbuf, msg.length);
            writeFlash(RAW_OUT_WE, addr24, readbuf, msg.length);
        }
        digitalWrite(RAW_OUT_CE, HIGH);
        return;
    }
    // GBM
    if (msg.request == REQ_GBM_WRITE_REGS) {
        setA15A18(addr);
        pinMode(RAW_OUT_OE, OUTPUT); // open-drain -> out
        digitalWrite(RAW_OUT_OE, HIGH);
        if (msg.length <= PACKET_SIZE) {
            Serial.readBytes(readbuf, msg.length);
            writeRegs(RAW_OUT_WE, readbuf, msg.length);
        }
        return;
    }
}
