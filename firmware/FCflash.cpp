#include "mbed.h"
#include "USBSerial.h"

#include <stdbool.h>
#include <stdint.h>

/******************************************
  Board
*****************************************/

//*** USBDEVICE ***

extern const PinMap PinMap_USB_FS[] = {
//  {PA_8,  USB_FS, STM_PIN_DATA(STM_MODE_AF_PP, GPIO_PULLUP, GPIO_AF10_OTG_FS)}, // USB_OTG_FS_SOF  // Connected to D5
//  {PA_9,  USB_FS, STM_PIN_DATA(STM_MODE_INPUT, GPIO_NOPULL, GPIO_AF10_OTG_FS)}, // USB_OTG_FS_VBUS // Connected to UART TXD1
    {PA_9,  USB_FS, STM_PIN_DATA(STM_MODE_INPUT, GPIO_NOPULL, GPIO_AF_NONE    )}, // USB_OTG_FS_VBUS // Connected to UART TXD1
    {PA_10, USB_FS, STM_PIN_DATA(STM_MODE_AF_OD, GPIO_PULLUP, GPIO_AF10_OTG_FS)}, // USB_OTG_FS_ID   // Connected to UART RXD1
    {PA_11, USB_FS, STM_PIN_DATA(STM_MODE_AF_PP, GPIO_PULLUP, GPIO_AF10_OTG_FS)}, // USB_OTG_FS_DM
    {PA_12, USB_FS, STM_PIN_DATA(STM_MODE_AF_PP, GPIO_PULLUP, GPIO_AF10_OTG_FS)}, // USB_OTG_FS_DP
    {NC,    NC,     0}
};

// TODO: 設定がちょっと違う
// https://os.mbed.com/users/mbed_official/code/USBDevice//file/461d954eee6b/USBDevice/USBHAL_STM32F4.cpp/
/*
MBED_WEAK const PinMap PinMap_USB_FS[] = {
//  {PA_8,      USB_FS, STM_PIN_DATA(STM_MODE_AF_PP, GPIO_PULLUP, GPIO_AF10_OTG_FS)}, // USB_OTG_FS_SOF // Connected to USB_SOF [TP1]
    {PA_9,      USB_FS, STM_PIN_DATA(STM_MODE_INPUT, GPIO_NOPULL, GPIO_AF10_OTG_FS)}, // USB_OTG_FS_VBUS // Connected to USB_VBUS
    {PA_10,     USB_FS, STM_PIN_DATA(STM_MODE_AF_OD, GPIO_PULLUP, GPIO_AF10_OTG_FS)}, // USB_OTG_FS_ID // Connected to USB_ID
    {PA_11,     USB_FS, STM_PIN_DATA(STM_MODE_AF_PP, GPIO_PULLUP, GPIO_AF10_OTG_FS)}, // USB_OTG_FS_DM // Connected to USB_DM
    {PA_12,     USB_FS, STM_PIN_DATA(STM_MODE_AF_PP, GPIO_PULLUP, GPIO_AF10_OTG_FS)}, // USB_OTG_FS_DP // Connected to USB_DP
    {NC, NC, 0}
};
MBED_WEAK const PinMap PinMap_USB_FS[] = {
//  {PA_8,      USB_FS, STM_PIN_DATA(STM_MODE_AF_PP, GPIO_PULLUP, GPIO_AF10_OTG_FS)}, // USB_OTG_FS_SOF // Connected to I2C3_SCL [ACP/RF_SCL]
//  {PA_9,      USB_FS, STM_PIN_DATA(STM_MODE_INPUT, GPIO_NOPULL, GPIO_AF_NONE    )}, // USB_OTG_FS_VBUS // Connected to STDIO_UART_TX
//  {PA_10,     USB_FS, STM_PIN_DATA(STM_MODE_AF_OD, GPIO_PULLUP, GPIO_AF10_OTG_FS)}, // USB_OTG_FS_ID // Connected to STDIO_UART_RX
    {PA_11,     USB_FS, STM_PIN_DATA(STM_MODE_AF_PP, GPIO_PULLUP, GPIO_AF10_OTG_FS)}, // USB_OTG_FS_DM // Connected to R4
    {PA_12,     USB_FS, STM_PIN_DATA(STM_MODE_AF_PP, GPIO_PULLUP, GPIO_AF10_OTG_FS)}, // USB_OTG_FS_DP // Connected to R5
    {NC, NC, 0}
};
MSTD_CONSTEXPR_OBJ_11 PinMap PinMap_USB_FS[] = {
//  {PA_8,      USB_FS, STM_PIN_DATA(STM_MODE_AF_PP, GPIO_PULLUP, GPIO_AF10_OTG_FS)}, // USB_OTG_FS_SOF // Connected to USB_SOF [TP1]
    {PA_9,      USB_FS, STM_PIN_DATA(STM_MODE_INPUT, GPIO_NOPULL, GPIO_AF_NONE    )}, // USB_OTG_FS_VBUS // Connected to USB_VBUS
    {PA_10,     USB_FS, STM_PIN_DATA(STM_MODE_AF_OD, GPIO_PULLUP, GPIO_AF10_OTG_FS)}, // USB_OTG_FS_ID // Connected to USB_ID
    {PA_11,     USB_FS, STM_PIN_DATA(STM_MODE_AF_PP, GPIO_PULLUP, GPIO_AF10_OTG_FS)}, // USB_OTG_FS_DM // Connected to USB_DM
    {PA_12,     USB_FS, STM_PIN_DATA(STM_MODE_AF_PP, GPIO_PULLUP, GPIO_AF10_OTG_FS)}, // USB_OTG_FS_DP // Connected to USB_DP
    {NC, NC, 0}
};
*/
/*
// mbed_app.json
{
    "config": {
        "usb_speed": {
            "help": "USE_USB_OTG_FS or USE_USB_OTG_HS or USE_USB_HS_IN_FS",
            "value": "USE_USB_OTG_FS"
        }
    },
    "target_overrides": {
        "*": {
            "target.device_has_add": ["USBDEVICE"]
        }
    }
}
*/

// Pins
//USBSerial serial; // via OTG port
BufferedSerial serial(USBTX, USBRX, 500000); // via mbed I/F
//Serial pcd(PC_6, PC_7);

// for GBA
BusInOut AD0_15(
    D0,  D1,  D2,  D3,
    D4,  D5,  D6,  D7,
    D8,  D9,  D10, D11,
    D12, D13, D14, D15
);
// shared
BusInOut A16_23_ROM_D0_7_RAM(
    PC_8,  PC_9, PC_10, PC_11,
    PC_12, PD_2, PC_13, PB_0
);

// pins are active low therefore everything is disabled now
DigitalOut WR(PE_4, 1);
DigitalOut RD(PE_5, 1);
DigitalOut CS(PE_6, 1);
DigitalOut CS2(PB_1, 1);

// for FC
//
// A0-12:       AD0-12
// CPU A13-14:  AD13-14
// PPU_A13:     AD15
// D0-7:        A16_23_ROM_D0_7_RAM
#define ROMSEL  CS
#define PHI2    CS2
#define CPU_RW  WR // R:1, W:0
#define PPU_RD  RD // TODO: 8BIT だと効かないので手動で変える

void setupBoardRegs()
{
    // disable
    CS = 1;
    CS2 = 1;
    WR = 1;
    RD = 1;

    // Input pull-up
    AD0_15.input();
    AD0_15.mode(PullUp);
    A16_23_ROM_D0_7_RAM.input();
    A16_23_ROM_D0_7_RAM.mode(PullUp);

    thread_sleep_for(400);
}


/******************************************
  GBA
*****************************************/

void readWordsCS(uint32_t addr24, uint8_t buf[], uint32_t length) {
    // addr
    AD0_15.output();
    A16_23_ROM_D0_7_RAM.output();
    AD0_15 = addr24 & 0xffff;
    A16_23_ROM_D0_7_RAM = addr24 >> 16;
    CS = 0;
    wait_ns(500);

    AD0_15.input();

    // read
    uint32_t lenWord = length >> 1;
    for (uint32_t i = 0; i < lenWord; ++i) {
        RD = 0;
        wait_ns(500);

        uint16_t tempWord = AD0_15;

        // inc addr
        RD = 1;
        wait_ns(100);

        buf[(i<<1)+0] = tempWord & 0xff;
        buf[(i<<1)+1] = tempWord >> 8;

        // TODO: check hi addr
        // (addr24>>16) == ((addr24 + (i<<1))>>16)
    }

    CS = 1;
    AD0_15.input();
    AD0_15.mode(PullUp);
    A16_23_ROM_D0_7_RAM.input();
    A16_23_ROM_D0_7_RAM.mode(PullUp);
}

void readBytesCS2(uint32_t addr16, uint8_t buf[], uint32_t length) {
    AD0_15.output();
    CS2 = 0;

    // read
    for (uint32_t i = 0; i < length; ++i) {
        AD0_15 = (addr16+i) & 0xffff;
        wait_ns(100);

        RD = 0;
        wait_ns(500);

        buf[i] = A16_23_ROM_D0_7_RAM;

        RD = 1;
    }

    CS2 = 1;
    AD0_15.input();
    AD0_15.mode(PullUp);
}


// Write one word to data pins of the cartridge
#if 0
void writeByte(word addr, byte data)
{
    // Set data pins to Output
    D0_7.output();
    A0_15 = addr;
    D0_7 = data;

#if 0
    __asm__("nop\n\t"
            "nop\n\t"
            "nop\n\t"
            "nop\n\t");
#else
    wait_us(1);
#endif

    // Switch CS and WR to LOW
    CS = 0;
    WR = 0;

#if 0
    __asm__("nop\n\t"
            "nop\n\t"
            "nop\n\t"
            "nop\n\t");
#else
    wait_us(1);
#endif

    // switch CS and WR to HIGH
    WR = 1;
    CS = 1;

#if 0
    __asm__("nop\n\t"
            "nop\n\t"
            "nop\n\t"
            "nop\n\t");
#else
    wait_us(1);
#endif

    // Set data pins to Input (or read errors?)
    D0_7.input();
}
#endif


/******************************************
  FC
*****************************************/
#define PRG_BASE 0x8000

uint8_t readByteCPU() {
    // already disabled all chips (PRG, W-RAM & CHR)
    // PRG
    ROMSEL = 0; // select chip
    PHI2 = 1;   // enable read & set addr
    wait_ns(200);

    // read
    uint8_t temp = A16_23_ROM_D0_7_RAM;

    // PRG
    PHI2 = 0;
    ROMSEL = 1;
    wait_ns(100);

    return temp;
}
uint8_t readBytePPU() {
    // already disabled all chips (PRG, W-RAM & CHR)
    // CHR, RAW
    PPU_RD = 0; // enable read
    wait_ns(200);

    // read
    uint8_t temp = A16_23_ROM_D0_7_RAM;

    // CHR, RAW
    PPU_RD = 1;
    wait_ns(100);

    return temp;
}
void writeByteCPU(uint8_t data) {
    // write
    A16_23_ROM_D0_7_RAM.output();
    A16_23_ROM_D0_7_RAM = data;
    wait_ns(100);

    // already disabled both chips (MMC & W-RAM)
    CPU_RW = 0; // enable write

    ROMSEL = 0; // select MMC:0 or W-RAM:1
    PHI2 = 1;   // enable that chip & set addr
    wait_ns(900);

    // latch
    PHI2 = 0;
    ROMSEL = 1;
    wait_ns(400);

    CPU_RW = 1;

    wait_ns(100);
    A16_23_ROM_D0_7_RAM.input();
    A16_23_ROM_D0_7_RAM.mode(PullUp);
}


/******************************************
  Host
*****************************************/
#define PACKET_SIZE (0x10000)

// request
// for GBA
#define REQ_READ16         0
#define REQ_WRITE16        1
#define REQ_WRITE16_RND    2
#define REQ_READ8_CS2      3
#define REQ_WRITE8_CS2     4
#define REQ_WRITE8_CS2_RND 5
// for FC
#define REQ_CPU_READ       16
#define REQ_CPU_WRITE      17
#define REQ_PPU_READ       18

typedef struct Message {
    uint32_t request;
    uint32_t value;
    uint32_t length;
    uint32_t _reserved;
} Message_t;


/******************************************
  main
*****************************************/
static uint8_t readbuf[PACKET_SIZE];

int main() {
    setupBoardRegs();
    while (1) {
        if (/*serial.available() < 1*/ !serial.readable()) {
            thread_sleep_for(10);
            continue;
        }

        Message_t msg;
        ssize_t n = serial.read(&msg, sizeof(msg));
        if (n <= 0) break;
        while (n < sizeof(msg)) {
            ssize_t m = serial.read((uint8_t *)&msg+n, sizeof(msg)-n);
            n += m;
        }

        // for GBA
        if (msg.request == REQ_READ16) {
            uint32_t addr24 = msg.value & 0xffffff;
            if (msg.length <= PACKET_SIZE) {
                readWordsCS(addr24, readbuf, msg.length);
                serial.write(readbuf, msg.length);
            }
            continue;
        }
        if (msg.request == REQ_WRITE16) {
            continue;
        }
        if (msg.request == REQ_WRITE16_RND) {
            continue;
        }
        if (msg.request == REQ_READ8_CS2) {
            uint32_t addr16 = msg.value & 0xffff;
            if (msg.length <= PACKET_SIZE) {
                readBytesCS2(addr16, readbuf, msg.length);
                serial.write(readbuf, msg.length);
            }
            continue;
        }
        if (msg.request == REQ_WRITE8_CS2) {
            continue;
        }
        if (msg.request == REQ_WRITE8_CS2_RND) {
            continue;
        }
        // for FC
        if (msg.request == REQ_CPU_READ) {
            PHI2 = 0;
            AD0_15 = PRG_BASE; // PPU_A13 = 1
            AD0_15.output();
            // addr: 0b1xxx_xxxx... 32KB full
            uint32_t addr16 = PRG_BASE | (msg.value & 0xffff);
            if (msg.length <= PACKET_SIZE) {
                for (uint32_t i = 0; i < msg.length; i++) {
                    AD0_15 = addr16 + i;
                    wait_ns(100);
                    readbuf[i] = readByteCPU();
                }
                serial.write(readbuf, msg.length);
            }
            continue;
        }
        if (msg.request == REQ_CPU_WRITE) {
            PHI2 = 0;
            AD0_15 = PRG_BASE; // PPU_A13 = 1
            AD0_15.output();
            // addr: 0b100x_xxxx... 0x8000-0x9fff only
            uint32_t addr16 = PRG_BASE | (msg.value & ~0x6000);
            uint8_t data = msg.length & 0xff;
            AD0_15 = addr16;
            wait_ns(500);
            writeByteCPU(data);
            continue;
        }
        if (msg.request == REQ_PPU_READ) {
            PHI2 = 0;
            AD0_15 = PRG_BASE; // PPU_A13 = 1
            AD0_15.output();
            // addr: 0b000x_xxxx... 8KB full
            uint32_t addr16 = (msg.value & 0x1fff);
            if (msg.length <= PACKET_SIZE) {
                for (uint32_t i = 0; i < msg.length; i++) {
                    AD0_15 = addr16 + i;
                    AD0_15[15] = 0; // PPU_A13 = 0
                    wait_ns(100);
                    readbuf[i] = readBytePPU();
                    AD0_15[15] = 1; // PPU_A13 = 1
                }
                serial.write(readbuf, msg.length);
            }
            continue;
        }
   }
}
