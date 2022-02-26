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

constexpr uint8_t IN_D0 = 5;
constexpr uint8_t IN_D1 = 6;
constexpr uint8_t IN_D2 = 7;
constexpr uint8_t IN_D3 = 8;
constexpr uint8_t IN_D4 = 9;
constexpr uint8_t IN_D5 = 10;
constexpr uint8_t IN_D6 = 11;
constexpr uint8_t IN_D7 = 12;

// CHR
// TODO: OUT_PPU_A13 -> /CS HIGH
constexpr uint8_t OUT_RD = OUT_A13; // shared
constexpr uint8_t OUT_WR = OUT_A14; // shared N.C.

// WRAM
//DigitalOut RW(PB_8, 1); // /WE: W:0, R:1
//DigitalOut O2(PB_9, 0); // /CS


/******************************************
  FC
*****************************************/
// Set Cartridge address
void setAddr(uint16_t addr, uint8_t OUT_OE)
{
    digitalWrite(OUT_OE, HIGH);

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
    digitalWrite(OUT_A13, (addr >> 13)&1);
    digitalWrite(OUT_A14, (addr >> 14)&1);

    __asm__(
        "nop\n\t"
        "nop\n\t"
    );
}

// Read one word out of the cartridge
uint8_t readByte(uint8_t OUT_OE) {
    // Pull read low
    digitalWrite(OUT_OE, LOW);
    __asm__(
        "nop\n\t"
        "nop\n\t"
    );

    // read
    uint8_t temp = (
        digitalRead(IN_D0)      |
        digitalRead(IN_D1) << 1 |
        digitalRead(IN_D2) << 2 |
        digitalRead(IN_D3) << 3 |
        digitalRead(IN_D4) << 4 |
        digitalRead(IN_D5) << 5 |
        digitalRead(IN_D6) << 6 |
        digitalRead(IN_D7) << 7
    );


    // Pull read high
    digitalWrite(OUT_OE, HIGH);
    __asm__(
        "nop\n\t"
        "nop\n\t"
    );

    return temp;
}

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

    digitalWrite(OUT_ROMSEL, HIGH);
    //digitalWrite(OUT_PPU_A13, HIGH);
    digitalWrite(OUT_RD, HIGH);
    digitalWrite(OUT_WR, HIGH);

    pinMode(IN_D0, INPUT_PULLUP);
    pinMode(IN_D1, INPUT_PULLUP);
    pinMode(IN_D2, INPUT_PULLUP);
    pinMode(IN_D3, INPUT_PULLUP);
    pinMode(IN_D4, INPUT_PULLUP);
    pinMode(IN_D5, INPUT_PULLUP);
    pinMode(IN_D6, INPUT_PULLUP);
    pinMode(IN_D7, INPUT_PULLUP);
}


#define PRG_BASE 0x8000
#define CHR_BASE 0x6000 // PPU /WR, PPU /RD -> HIGH

#define READ_BUF_SIZE (1024)
static uint8_t readbuf[READ_BUF_SIZE];


/******************************************
  main
*****************************************/

void confirm()
{
    Serial.println("");
    Serial.println("ready?");
    int tmp;
    do {
        delay(10); // [msec]
        tmp = Serial.read();
    } while (tmp == -1);
}

void loop() {
    confirm();

    uint16_t prg_bytes = 32 * 1024;
    Serial.print("read PRG ");
    Serial.print(prg_bytes);
    Serial.println("[bytes]");
    uint16_t chr_bytes = 8 * 1024;
    Serial.print("read CHR ");
    Serial.print(chr_bytes);
    Serial.println("[bytes]");

    Serial.println("set PPU A13(56) to HIGH!");
    confirm();

    // PRG
    {
        for(uint16_t size = 0; size < prg_bytes; size += READ_BUF_SIZE) {
            for (uint16_t currByte = 0; currByte < READ_BUF_SIZE; currByte++) {
                noInterrupts();
                setAddr(PRG_BASE + size + currByte, OUT_ROMSEL);
                readbuf[currByte] = readByte(OUT_ROMSEL);
                interrupts();
            }
            // dump
            for (uint16_t y = 0; y < READ_BUF_SIZE; y += 16) {
                Serial.print(PRG_BASE + size + y, HEX);
                for (uint16_t x = 0; x < 16; x++) {
                    Serial.print(" ");
                    Serial.print(readbuf[y+x], HEX);
                }
                Serial.println("");
            }
            confirm();
        }
    }
    Serial.println("PRG done");

    Serial.println("set PPU A13(56) to LOW!");
    confirm();

    // CHR
    {
        for(uint16_t size = 0; size < chr_bytes; size += READ_BUF_SIZE) {
            for (uint16_t currByte = 0; currByte < READ_BUF_SIZE; currByte++) {
                noInterrupts();
                setAddr(CHR_BASE + size + currByte, OUT_RD);
                readbuf[currByte] = readByte(OUT_RD);
                interrupts();
            }
            // dump
            for (uint16_t y = 0; y < READ_BUF_SIZE; y += 16) {
                Serial.print(CHR_BASE + size + y, HEX);
                for (uint16_t x = 0; x < 16; x++) {
                    Serial.print(" ");
                    Serial.print(readbuf[y+x], HEX);
                }
                Serial.println("");
            }
        }
    }
    Serial.println("CHR done");
}
