package storage

import (
	"encoding/binary"
	"os"
)

type Segment struct {
	File   *os.File
	Offset int64
}

func OpenSegment(path string) (*Segment, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return &Segment{File: f, Offset: info.Size()}, nil
}

func (s *Segment) Write(key, value []byte) (int64, error) {
	data := EncodeRecord(key, value)
	offset := s.Offset
	_, err := s.File.Write(data)
	if err != nil {
		return 0, err
	}
	s.Offset += int64(len(data))
	return offset, nil
}

func (s *Segment) ReadAt(offset int64) ([]byte, []byte, error) {
	header := make([]byte, HeaderSize)
	if _, err := s.File.ReadAt(header, offset); err != nil {
		return nil, nil, err
	}
	keySize := binary.LittleEndian.Uint32(header[12:])
	valSize := binary.LittleEndian.Uint32(header[16:])

	data := make([]byte, keySize+valSize)
	if _, err := s.File.ReadAt(data, offset+HeaderSize); err != nil {
		return nil, nil, err
	}
	return data[:keySize], data[keySize:], nil
}
