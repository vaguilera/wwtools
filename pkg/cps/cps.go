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

func GreyPalette() []byte {
	p := make([]byte, 256*3)
	for i := range p {
		p[i] = byte(i / 3)
	}
	return p
}

func expandPalette(src []byte) []byte {
	dst := make([]byte, len(src))
	for i := 0; i < len(src); i++ {
		// src[i] en [0..63], lo subimos a [0..255]
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

	var imgRaw []byte
	switch header.CompressionType {
	case COMPRESSION_WESTWOOD_LCW:
		imgRaw = decompressLCW(data[10:])
	case COMPRESSION_WESTWOOD_RLE:
		imgRaw = decompressRLE(data[10:])
	default:
		return nil, fmt.Errorf("unsupported compression method")
	}

	width := 320
	height := 200

	var palRaw []byte
	if palette == nil {
		palRaw = GreyPalette()
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

func decompressLCW(data []byte) []byte {
	dest := make([]byte, 64000)

	dp := uint16(0)
	sp := uint16(0)
	relative := false
	if data[0] == 0 {
		relative = true
		sp++
	}
	for {
		if dp >= 64000 {
			return dest
		}
		com := data[sp]
		sp++

		if (com & 0x80) == 0 { // bit 7
			// command 2 Existing block relative copy
			count := uint16((com >> 4) + 3)
			if count > 64000-dp {
				count = 64000 - dp
			}
			pos := ((uint16(com & 0x0F)) << 8) + uint16(data[sp])
			sp++
			for i := uint16(0); i < count; i++ {
				dest[dp] = dest[dp-pos]
				dp++
			}
		} else {
			if (com & 0x40) == 0 { // Command 1 - short copy
				count := com & 0x3F
				if count == 0 {
					return dest
				}
				for i := byte(0); i < count; i++ {
					dest[dp] = data[sp]
					sp++
					dp++
				}
			} else {
				count := com & 0x3F
				if count < 0x3E { // {large copy (3)}
					count += 3
					// {Next word = pos. from start of image}
					posit := uint16(0)
					dat := (uint16(data[sp+1]) << 8) + uint16(data[sp])
					sp += 2
					if relative { // { relative }
						posit = dp - dat
					} else {
						posit = dat
					}
					for i := posit; i < posit+uint16(count); i++ {
						if dp >= 64000 || i >= 64000 {
							return dest
						}
						dest[dp] = dest[i]
						dp++
					}

				} else if count == 0x3F { //{very large copy (5)}
					count := (uint16(data[sp+1]) << 8) + uint16(data[sp]) // {next 2 words are Count and Pos}
					sp += 2
					posit := uint16(0)
					dat := (uint16(data[sp+1]) << 8) + uint16(data[sp])
					sp += 2
					if relative {
						posit = dp - dat
					} else {
						posit = dat
					}
					for i := posit; i < posit+count; i++ {
						dest[dp] = dest[i]
						dp++
					}
				} else {
					// Oush. It was littleEndian
					count := (uint16(data[sp+1]) << 8) + uint16(data[sp]) // command 4
					sp += 2
					b := data[sp]
					sp++
					for i := uint16(0); i < count; i++ {
						dest[dp] = b
						dp++
					}
				}
			}
		}
	}
}

func decompressRLE(data []byte) []byte {
	dest := make([]byte, 64000)

	dp := 0
	sp := 0

	for {
		if dp >= 64000 {
			return dest
		}
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
}
