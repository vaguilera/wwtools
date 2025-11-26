# wwtools
Westwood Studios resource loader

Allows to read and write PAK files and uncompress CPS graphic files

# Usage

### CPS 
``LoadCPS**(data []byte, palette []byte) (\*CPSImage, error)``
Load and decompress image file from a byte array. It returns a CPSIMage struct


```
type CPSImage struct {
	Compression string
	Palette     bool
	Width       int
	Height      int
	Image       *image.RGBA
}
```

```func SavePNG(filename string) error```
Save a CPSImage to a specified PNG 


### PAK

```func FromFile(f string) (*PakFile, error)```
Load and Parse PAK file from disk

```func ParsePakData(data []byte) (*PakFile, error)```
Parse a PAK file in memory returning a the following struct:

```
type PakFile struct {
	Files []File
}
type File struct {
    Name   string
    Offset uint32
    Data   []byte
}
```

```func (p *PakFile) ExtractFile(name string) error {```
Extract a file by name from a loaded PAK

```func (p *PakFile) ExtractAll() error```
Extract every file within the PAK file

```func (p *PakFile) AddFile(data []byte, name string) error```
Add a new file to the PAK file

```func (p *PakFile) SaveToFile(filename string) error```
Save current PAK file to the specified file