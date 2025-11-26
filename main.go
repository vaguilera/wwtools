package main

import (
	"fmt"
	"os"

	"github.com/vaguilera/wwtools/pkg/cps"
)

func main() {
	data, err := os.ReadFile("./data/brandon.cps")
	if err != nil {
		panic(err)
	}

	pal, err := os.ReadFile("./data/origpal.col")
	if err != nil {
		panic(err)
	}

	img, err := cps.LoadCPS(data, pal)
	if err != nil {
		panic(err)
	}

	img.SavePNG("image.png")
	fmt.Println("Compression Type:", img.Compression)
	fmt.Println("Embedded palette:", img.Palette)
}
