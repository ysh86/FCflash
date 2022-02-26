package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	fw, err := os.Create(os.Args[2])
	if err != nil {
		panic(err)
	}
	defer fw.Close()

	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()

	nextAddr := uint32(0)
	data := make([]byte, 16)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := scanner.Text()
		elms := strings.Split(s, " ")

		// validate
		if len(elms) != 17 {
			continue
		}

		// address
		a, err := strconv.ParseUint(elms[0], 16, 32)
		if err != nil {
			panic(err)
		}
		addr := uint32(a)
		if nextAddr == 0x10000 {
			// wrap around
			nextAddr = 0
		}
		if nextAddr == 0 {
			nextAddr = addr
		}
		if nextAddr != addr {
			panic(fmt.Errorf("invalid address: %08x", addr))
		}
		nextAddr = addr + 16

		// data
		for i := 0; i < 16; i++ {
			d, err := strconv.ParseUint(elms[i+1], 16, 8)
			if err != nil {
				panic(err)
			}
			data[i] = byte(d)
		}

		// dump
		_, err = fw.Write(data)
		if err != nil {
			panic(err)
		}
	}
}
