ROM/EEPROM/Flash reader/writer for FC
=============================================

Arduino
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

cmd
---------------------------------------------

### tuna
Host tool of the reader/writer for FC.

### dlzss
LZSS decompressor.

| title | offset | compressed length |
| ----- | ------ | ----------------- |
| Mr.SPLASH | 713996 | 32KB+8KB (NOT compressed) |
| 8BIT MUSIC POWER | 430196 | 262626 |
| 8BIT MUSIC POWER ENCORE | 430212 | 175456 |
| 8BIT MUSIC POWER FINAL | 430212 | 289760 |

### hex2bin
TBD
