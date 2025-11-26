package main

import (
	"github.com/vaguilera/wwtools/pkg/pak"
)

func main() {
	// data, err := os.ReadFile("./data/falls.cps")
	// if err != nil {
	// 	panic(err)
	// }

	// pal, err := os.ReadFile("./data/origpal.col")
	// if err != nil {
	// 	panic(err)
	// }

	// img, err := cps.LoadCPS(data, pal)
	// if err != nil {
	// 	panic(err)
	// }

	// img.SavePNG("image.png")
	// fmt.Println("Compression Type:", img.Compression)
	// fmt.Println("Embedded palette:", img.Palette)

	pakFile := pak.NewPackFile()
	pakFile.AddFile([]byte("sdjfklsdjlkfjskdfkdsf"), "test.ddd")
	pakFile.AddFile([]byte("ghjghjghjghjhg"), "test2la.ddd")

	err := pakFile.SaveToFile("test.pak")
	if err != nil {
		panic(err)
	}

}
