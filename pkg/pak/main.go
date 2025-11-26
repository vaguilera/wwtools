package pakfile

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"os"
)

type (
	PakFile struct {
		Files []File
		Data  []byte
	}

	File struct {
		Offset uint32
		Size   uint32
		Name   string
	}
)

func NewPackFile() PakFile {
	return PakFile{
		Files: []File{},
	}
}

func readFile(f string) ([]byte, error) {
	file, err := os.Open(f)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	var size int64 = info.Size()
	data := make([]byte, size)

	buffer := bufio.NewReader(file)
	_, err = buffer.Read(data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (p *PakFile) Open(f string) error {
	data, err := readFile(f)
	if err != nil {
		return err
	}
	p.Data = data
	r := bytes.NewReader(data)
	for {
		name := []byte{}
		offset := uint32(0)
		err := binary.Read(r, binary.LittleEndian, &offset)
		if err != nil {
			return err
		}

		if offset == 0 {
			break
		}
		for {
			b, err := r.ReadByte()
			if err != nil {
				return err
			}

			if b == 0 {
				break
			}
			name = append(name, b)
		}
		p.Files = append(p.Files, File{Name: string(name), Offset: offset})
	}
	numOfFiles := len(p.Files)
	for i := 0; i < numOfFiles-1; i++ {
		p.Files[i].Size = p.Files[i+1].Offset - p.Files[i].Offset
	}
	p.Files[numOfFiles-1].Size = uint32(len(data)) - p.Files[numOfFiles-1].Offset

	return nil
}

func (p *PakFile) Extract() error {
	if len(p.Files) == 0 {
		return errors.New("no files to extract")
	}

	for _, f := range p.Files {
		err := os.WriteFile(f.Name, p.Data[f.Offset:f.Offset+f.Size], 0644)
		if err != nil {
			return err
		}
	}

	return nil
}
