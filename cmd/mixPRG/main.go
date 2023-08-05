package main

import (
	"os"
)

func main() {
	if len(os.Args) < 4 {
		panic(os.Args)
	}

	srcEven, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	srcOdd, err := os.ReadFile(os.Args[2])
	if err != nil {
		panic(err)
	}

	dst := make([]byte, len(srcEven))
	for i := 0; i < 128*1024; i++ {
		dst[i] = srcEven[i]
	}
	for i := 128 * 1024; i < 2*128*1024; i++ {
		dst[i] = srcOdd[i]
	}
	for i := 2 * 128 * 1024; i < 3*128*1024; i++ {
		dst[i] = srcEven[i]
	}
	for i := 3 * 128 * 1024; i < 4*128*1024; i++ {
		dst[i] = srcOdd[i]
	}

	os.WriteFile(os.Args[3], dst, 0644)
}
