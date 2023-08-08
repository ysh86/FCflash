package main

import (
	"os"
)

func main() {
	if len(os.Args) < 4 {
		panic(os.Args)
	}

	srcH, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	srcL, err := os.ReadFile(os.Args[2])
	if err != nil {
		panic(err)
	}

	dst := make([]byte, len(srcH)+len(srcL))
	for i := range dst {
		if i&1 == 0 {
			dst[i] = srcH[i>>1]
		} else {
			dst[i] = srcL[i>>1]
		}
	}

	os.WriteFile(os.Args[3], dst, 0644)
}
