package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"unicode/utf16"
)

type File struct {
	hoge int
}

const (
	RES_NULL_TYPE        = 0x0000
	RES_STRING_POOL_TYPE = 0x0001
	RES_TABLE_TYPE       = 0x0002
	RES_XML_TYPE         = 0x0003

	// Chunk types in RES_XML_TYPE
	RES_XML_FIRST_CHUNK_TYPE     = 0x0100
	RES_XML_START_NAMESPACE_TYPE = 0x0100
	RES_XML_END_NAMESPACE_TYPE   = 0x0101
	RES_XML_START_ELEMENT_TYPE   = 0x0102
	RES_XML_END_ELEMENT_TYPE     = 0x0103
	RES_XML_CDATA_TYPE           = 0x0104
	RES_XML_LAST_CHUNK_TYPE      = 0x017f

	// This contains a uint32_t array mapping strings in the string
	// pool back to resource identifiers.  It is optional.
	RES_XML_RESOURCE_MAP_TYPE = 0x0180

	// Chunk types in RES_TABLE_TYPE
	RES_TABLE_PACKAGE_TYPE   = 0x0200
	RES_TABLE_TYPE_TYPE      = 0x0201
	RES_TABLE_TYPE_SPEC_TYPE = 0x0202
)

type ResChunkHeader struct {
	Type       uint16
	HeaderSize uint16
	Size       uint32
}

const SORTED_FLAG = 1 << 0
const UTF8_FLAG = 1 << 8

type ResStringPoolHeader struct {
	Header      ResChunkHeader
	StringCount uint32
	StyleCount  uint32
	Flags       uint32
	StringStart uint32
	StylesStart uint32
}

type ResStringPool struct {
	Header  ResStringPoolHeader
	Strings []string
	Styles  []string
}

func NewFile(r io.ReaderAt) (*File, error) {
	f := new(File)
	sr := io.NewSectionReader(r, 0, 1<<63-1)

	header := new(ResChunkHeader)
	binary.Read(sr, binary.LittleEndian, header)
	offset := uint32(header.HeaderSize)

	for offset < header.Size {
		sr.Seek(int64(offset), os.SEEK_SET)
		chunkHeader := &ResChunkHeader{}
		binary.Read(sr, binary.LittleEndian, chunkHeader)

		chunkReader := io.NewSectionReader(r, int64(offset), int64(chunkHeader.Size))
		switch chunkHeader.Type {
		case RES_STRING_POOL_TYPE:
			fmt.Println(ReadStringPool(chunkReader))
		case RES_XML_RESOURCE_MAP_TYPE:
			fmt.Println("RES_XML_RESOURCE_MAP_TYPE")
		default:
			fmt.Println(chunkHeader.Type)
		}

		offset += chunkHeader.Size
	}
	return f, nil
}

func ReadStringPool(sr *io.SectionReader) (*ResStringPool, error) {
	sp := new(ResStringPool)
	binary.Read(sr, binary.LittleEndian, &sp.Header)

	stringStarts := make([]uint32, sp.Header.StringCount)
	binary.Read(sr, binary.LittleEndian, stringStarts)
	styleStarts := make([]uint32, sp.Header.StyleCount)
	binary.Read(sr, binary.LittleEndian, styleStarts)

	sp.Strings = make([]string, sp.Header.StringCount)
	for i, start := range stringStarts {
		var str string
		var err error
		if (sp.Header.Flags & UTF8_FLAG) == 0 {
			str, err = ReadUTF16(sr, int64(sp.Header.StringStart+start))
		} else {
			str, err = ReadUTF8(sr, int64(sp.Header.StringStart+start))
		}
		if err != nil {
			return nil, err
		}
		sp.Strings[i] = str
	}

	sp.Styles = make([]string, sp.Header.StyleCount)
	for i, start := range styleStarts {
		var str string
		var err error
		if (sp.Header.Flags & UTF8_FLAG) == 0 {
			str, err = ReadUTF16(sr, int64(sp.Header.StylesStart+start))
		} else {
			str, err = ReadUTF8(sr, int64(sp.Header.StylesStart+start))
		}
		if err != nil {
			return nil, err
		}
		sp.Styles[i] = str
	}

	return sp, nil
}

func ReadUTF16(sr *io.SectionReader, offset int64) (string, error) {
	var size uint16
	sr.Seek(offset, os.SEEK_SET)
	if err := binary.Read(sr, binary.LittleEndian, &size); err != nil {
		return "", err
	}
	buf := make([]uint16, size)
	if err := binary.Read(sr, binary.LittleEndian, buf); err != nil {
		return "", err
	}
	return string(utf16.Decode(buf)), nil
}

func ReadUTF8(sr *io.SectionReader, offset int64) (string, error) {
	var size uint16
	sr.Seek(offset, os.SEEK_SET)
	if err := binary.Read(sr, binary.LittleEndian, &size); err != nil {
		return "", err
	}
	buf := make([]uint8, size)
	if err := binary.Read(sr, binary.LittleEndian, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func main() {
	f, _ := os.Open("AndroidManifest.xml")
	NewFile(f)
}