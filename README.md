ROM/EEPROM/Flash/SRAM reader/writer for FC/MD/GB/GBC/GBA
=============================================

Arduino 5V
---------------------------------------------

pins:
```
TBD
```

build:
```
$ arduino-cli compile -b arduino:avr:micro --build-cache-path ./build --build-path ./build ../FCflash
```

program:
```
$ avrdude.exe -p m32u4 -c avr109 -P COM5 -U flash:w:./build/FCflash.ino.hex:i
```

mbed 3.3V
---------------------------------------------

pins:
```
TBD
```

cmd
---------------------------------------------

### tuna
Host tool of the reader/writer for FC/MD.
```bash
$ ./tuna -h
Usage of ./tuna:
  -baud int
        baud rate (default 115200)
  -chr int
        Size of CHR ROM in 8KB units (Value 0 means the board uses CHR RAM)
  -com int
        com port (default 5)
  -eeprom
        write NROM EEPROM
  -flash
        write Flash
  -mapper int
        mapper 0:NROM, 1:SxROM, 4:TxROM (default 1)
  -mirror int
        0:H, 1:V, 2:battery-backed PRG RAM (default 2)
  -prg int
        Size of PRG ROM in 16KB units (default 16)
  -raw
        raw access to ROM/RAM/EEPROM/Flash ICs
```

### tunag
Host tool of the reader/writer for GB/GBC
```bash
$ ./tunag -h
Usage of ./tunag:
  -a    dump both ROM & RAM
  -baud int
        baud rate (default 115200)
  -com int
        com port (default 5)
  -flash
        write Flash
  -ram
        write RAM in cartridge
```

### tunaa
Host tool of the reader/writer for GBA (and FC 3.3V cartridges)
```bash
$ ./tunaa -h
Usage of ./tunaa:
  -baud int
        baud rate (default 500000)
  -com int
        com port (default 7)
  -fc7
        dump FC mapper7 AxROM PRG
  -fc8bit
        dump FC 8BIT TxROM
  -flash
        write Flash
  -ram
        write RAM in cartridge
```

### dlzss
LZSS decompressor.

| title | offset | compressed length |
| ----- | ------ | ----------------- |
| Mr.SPLASH | 713996 | 32KB+8KB (NOT compressed) |
| 8BIT MUSIC POWER | 430196 | 262626 |
| 8BIT MUSIC POWER FINAL | 430212 | 289760-1 |
| 8BIT MUSIC POWER ENCORE | 430212 | 175456 |
