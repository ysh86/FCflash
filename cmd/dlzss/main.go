package main

import (
	"errors"
	"flag"
	"io"
	"os"

	"github.com/blacktop/lzss"
)

func main() {
	// args
	var (
		mapper int
		prg    int
		chr    int
		mirror int
		offset int64
		length int
		input  string
		output string
	)
	flag.IntVar(&mapper, "mapper", 4, "mapper 0:NROM, 4:TxROM")
	flag.IntVar(&prg, "prg", 32, "Size of PRG ROM in 16KB units")
	flag.IntVar(&chr, "chr", 32, "Size of CHR ROM in 8KB units (Value 0 means the board uses CHR RAM)")
	flag.IntVar(&mirror, "mirror", 0, "0:H, 1:V, 2:battery-backed PRG RAM")
	flag.Int64Var(&offset, "offset", 0, "byte offset for the input file")
	flag.IntVar(&length, "length", 0, "length of the input file")
	flag.Parse()
	args := flag.Args()
	if len(args) < 2 {
		panic(errors.New("no input/output files"))
	}
	input = args[0]
	output = args[1]

	fsrc, err := os.Open(input)
	if err != nil {
		panic(err)
	}
	defer fsrc.Close()
	_, err = fsrc.Seek(offset, io.SeekStart)
	if err != nil {
		panic(err)
	}

	fdst, err := os.Create(output)
	if err != nil {
		panic(err)
	}
	defer fdst.Close()

	buf := make([]uint8, length)

	// header
	buf[0] = 'N'
	buf[1] = 'E'
	buf[2] = 'S'
	buf[3] = 0x1a
	buf[4] = uint8(prg)
	buf[5] = uint8(chr)
	buf[6] = uint8((mapper << 4) | mirror)
	_, err = fdst.Write(buf[0:16])
	if err != nil {
		panic(err)
	}

	// decompress
	_, err = io.ReadFull(fsrc, buf[:])
	if err != nil {
		panic(err)
	}
	dlzss := lzss.Decompress(buf[:])
	_, err = fdst.Write(dlzss)
	if err != nil {
		panic(err)
	}
}
