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
	for i := range dst {
		if i&2 == 0 {
			dst[i] = srcEven[i]
		} else {
			dst[i] = srcOdd[i]
		}
	}

	os.WriteFile(os.Args[3], dst, 0644)
}
