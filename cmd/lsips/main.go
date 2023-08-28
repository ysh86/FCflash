package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: cmd ips_file\n")
		os.Exit(-1)
	}

	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %+v\n", err)
		os.Exit(-1)
	}

	size := len(raw)
	if size < 8 {
		fmt.Fprintf(os.Stderr, "invalid format\n")
		os.Exit(-1)
	}
	if string(raw[0:5]) != "PATCH" || string(raw[size-3:size]) != "EOF" {
		fmt.Fprintf(os.Stderr, "invalid format\n")
		os.Exit(-1)
	}

	raw = raw[5:]
	for len(raw) > 3+2 {
		offset := (int(raw[0]) << 16) | (int(raw[1]) << 8) | int(raw[2])
		size := (int(raw[3]) << 8) | int(raw[4])
		data := raw[5 : 5+size]
		raw = raw[5+size:]

		fmt.Printf("%06x:", offset)
		for _, d := range data {
			fmt.Printf(" %02x", d)
		}
		fmt.Println("")
	}
}
