package cps

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

const (
	COMPRESSION_NONE            = 0x0000
	COMPRESSION_WESTWOOD_LZW_12 = 0x0001
	COMPRESSION_WESTWOOD_LZW_14 = 0x0002
	COMPRESSION_WESTWOOD_RLE    = 0x0003
	COMPRESSION_WESTWOOD_LCW    = 0x0004
)

func compMethodToString(method uint16) string {
	switch method {
	case COMPRESSION_NONE:
		return "None"
	case COMPRESSION_WESTWOOD_LZW_12:
		return "Westwood LZW 12-bit"
	case COMPRESSION_WESTWOOD_LZW_14:
		return "Westwood LZW 14-bit"
	case COMPRESSION_WESTWOOD_RLE:
		return "Westwood RLE"
	case COMPRESSION_WESTWOOD_LCW:
		return "Westwood LCW"
	default:
		return "Unknown"
	}
}

type CPSHeader struct {
	FileSize         uint16
	CompressionType  uint16
	UncompressedSize uint32
	PaletteSize      uint16
}

type CPSImage struct {
	Compression string
	Palette     bool
	Width       int
	Height      int
	Image       *image.RGBA
}

func grayPalette() []byte {
	p := make([]byte, 256*3)
	for i := range p {
		p[i] = byte(i / 3)
	}
	return p
}

func expandPalette(src []byte) []byte {
	dst := make([]byte, len(src))
	for i := 0; i < len(src); i++ {
		v := int(src[i]) * 255 / 63
		dst[i] = byte(v)
	}
	return dst
}

func LoadCPS(data []byte, palette []byte) (*CPSImage, error) {
	var header CPSHeader
	r := bytes.NewReader(data)
	err := binary.Read(r, binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}
	width := 320
	height := 200

	var imgRaw []byte
	switch header.CompressionType {
	case COMPRESSION_WESTWOOD_LCW:
		imgRaw = decompressLCW(data[10:], width*height)
	case COMPRESSION_WESTWOOD_RLE:
		imgRaw = decompressRLE(data[10:], width*height)
	default:
		return nil, fmt.Errorf("unsupported compression method")
	}

	var palRaw []byte
	if palette == nil {
		palRaw = grayPalette()
	} else {
		palRaw = expandPalette(palette)
	}

	upLeft := image.Point{0, 0}
	lowRight := image.Point{width, height}

	i := 0
	img := image.NewRGBA(image.Rectangle{upLeft, lowRight})
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			col := int(imgRaw[i]) * 3
			img.Set(x, y, color.RGBA{palRaw[col], palRaw[col+1], palRaw[col+2], 0xff})
			i++
		}
	}

	return &CPSImage{
		Compression: compMethodToString(header.CompressionType),
		Palette:     header.PaletteSize != 0,
		Width:       width,
		Height:      height,
		Image:       img,
	}, nil
}

func (cps *CPSImage) SavePNG(filename string) error {
	if cps.Image == nil {
		return fmt.Errorf("no image data to save")
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	err = png.Encode(f, cps.Image)
	if err != nil {
		return err
	}
	return nil
}

func decompressLCW(data []byte, imgSize int) []byte {
	dest := make([]byte, imgSize)
	dp := 0
	sp := 0

	relative := false
	if len(data) > 0 && data[0] == 0 {
		relative = true
		sp++
	}

	for dp < imgSize && sp < len(data) {
		com := data[sp]
		sp++

		if (com & 0x80) == 0 { // bit 7
			// command 2 Existing block relative copy
			count := int(com>>4) + 3
			pos := (int(com&0x0F) << 8) + int(data[sp])
			sp++
			for i := 0; i < count && dp < imgSize; i++ {
				dest[dp] = dest[dp-pos]
				dp++
			}
			continue
		}

		if (com & 0x40) == 0 { // Command 1 - short copy
			count := int(com & 0x3F)
			if count == 0 {
				break
			}
			if sp+count > len(data) {
				count = len(data) - sp
			}
			if dp+count > imgSize {
				count = imgSize - dp
			}
			copy(dest[dp:dp+count], data[sp:sp+count])
			sp += count
			dp += count
			continue
		}

		count := int(com & 0x3F)
		switch {
		case count < 0x3E: // {large copy (3)}
			count += 3
			// {Next word = pos. from start of image}
			dat := (int(data[sp+1]) << 8) + int(data[sp])
			sp += 2
			var posit int
			if relative { // { relative }
				posit = dp - dat
			} else {
				posit = dat
			}
			for i := posit; i < posit+count && dp < imgSize; i++ {
				if i >= imgSize {
					break
				}
				dest[dp] = dest[i]
				dp++
			}

		case count == 0x3F: //{very large copy (5)}
			count := (int(data[sp+1]) << 8) + int(data[sp]) // {next 2 words are Count and Pos}
			sp += 2
			dat := (int(data[sp+1]) << 8) + int(data[sp])
			sp += 2
			var posit int
			if relative {
				posit = dp - dat
			} else {
				posit = dat
			}
			for i := posit; i < posit+count; i++ {
				dest[dp] = dest[i]
				dp++
			}
		default:
			count := (int(data[sp+1]) << 8) + int(data[sp]) // command 4
			sp += 2
			b := data[sp]
			sp++
			for i := 0; i < count; i++ {
				dest[dp] = b
				dp++
			}
		}
	}
	return dest
}

func decompressRLE(data []byte, imgSize int) []byte {
	dest := make([]byte, imgSize)
	dp := 0
	sp := 0

	for dp < imgSize && sp < len(data) {
		b := int(int8(data[sp]))
		sp++

		if b > 0x00 {
			copy(dest[dp:dp+b], data[sp:sp+b])
			sp += b
			dp += b
		}
		if b < 0x00 {
			count := -b
			val := data[sp]
			sp++
			for i := 0; i < count; i++ {
				dest[dp] = val
				dp++
			}
		}
		if b == 0x00 {
			count := int(data[sp])<<8 | int(data[sp+1])
			val := data[sp+2]
			sp += 3
			for i := 0; i < count; i++ {
				dest[dp] = val
				dp++
			}
		}
	}
	return dest
}
