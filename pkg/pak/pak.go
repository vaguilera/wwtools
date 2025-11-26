package pak

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
)

type PakFile struct {
	Files []File
}

type File struct {
	Name   string
	Offset uint32
	Data   []byte
}

func NewPackFile() PakFile {
	return PakFile{
		Files: []File{},
	}
}

func readCString(r *bytes.Reader) (string, error) {
	var str []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		if b == 0 {
			return string(str), nil
		}
		str = append(str, b)
	}
}

func FromFile(f string) (*PakFile, error) {
	data, err := os.ReadFile(f)
	if err != nil {
		return nil, err
	}
	return ParsePakData(data)
}

func ParsePakData(data []byte) (*PakFile, error) {
	p := NewPackFile()
	r := bytes.NewReader(data)
	for {
		offset := uint32(0)
		err := binary.Read(r, binary.LittleEndian, &offset)
		if err != nil {
			return nil, err
		}
		if offset == 0 {
			break
		}
		name, err := readCString(r)
		if err != nil {
			return nil, err
		}
		p.Files = append(p.Files, File{Name: name, Offset: offset})
	}
	numOfFiles := len(p.Files)
	for i := 0; i < numOfFiles-1; i++ {
		size := p.Files[i+1].Offset - p.Files[i].Offset
		p.Files[i].Data = make([]byte, size)
		copy(p.Files[i].Data, data[p.Files[i].Offset:p.Files[i].Offset+size])
	}
	lastfile := len(p.Files) - 1
	size := uint32(len(data)) - p.Files[numOfFiles-1].Offset
	p.Files[lastfile].Data = make([]byte, size)
	copy(p.Files[lastfile].Data, data[p.Files[lastfile].Offset:p.Files[lastfile].Offset+size])
	return &p, nil
}

func (p *PakFile) ExtractFile(name string) error {
	if len(p.Files) == 0 {
		return errors.New("no files to extract")
	}

	for _, f := range p.Files {
		if f.Name == name {
			err := os.WriteFile(f.Name, f.Data, 0644)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *PakFile) ExtractAll() error {
	if len(p.Files) == 0 {
		return errors.New("no files to extract")
	}

	for _, f := range p.Files {
		err := os.WriteFile(f.Name, f.Data, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *PakFile) AddFile(data []byte, name string) error {
	if len(data) == 0 {
		return errors.New("data is empty")
	}

	for _, f := range p.Files {
		if f.Name == name {
			return errors.New("file already exists in the pak")
		}
	}

	p.Files = append(p.Files, File{
		Offset: 0,
		Data:   data,
		Name:   name,
	})

	p.recalculateOffsets()

	return nil
}

func (p *PakFile) recalculateOffsets() {
	var tableSize uint32
	for _, f := range p.Files {
		tableSize += uint32(len(f.Name)) + 5 // +1 for null terminator +4 for offset
	}

	offset := tableSize
	for i := range p.Files {
		p.Files[i].Offset = offset
		offset += uint32(len(p.Files[i].Data))
	}
}

func (p *PakFile) SaveToFile(filename string) error {
	var buf bytes.Buffer
	for _, f := range p.Files {
		err := binary.Write(&buf, binary.LittleEndian, f.Offset)
		if err != nil {
			return err
		}

		buf.WriteString(f.Name)
		buf.WriteByte(0) // Null terminator
	}
	for _, f := range p.Files {
		buf.Write(f.Data)
	}

	err := os.WriteFile(filename, buf.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}
