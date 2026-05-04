package storage

import (
    "encoding/binary"
    "os"
)

type segment struct {
    file   *os.File
    offset int64  // pr¢xima posi‡Æo de escrita
}

func openSegment(path string) (*segment, error) {
    f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
    if err != nil {
        return nil, err
    }
    info, err := f.Stat()
    if err != nil {
        return nil, err
    }
    return &segment{file: f, offset: info.Size()}, nil
}

func (s *segment) write(key, value []byte) (int64, error) {
    data := encodeRecord(key, value)
    offset := s.offset
    _, err := s.file.Write(data)
    if err != nil {
        return 0, err
    }
    s.offset += int64(len(data))
    return offset, nil
}

func (s *segment) readAt(offset int64) ([]byte, []byte, error) {
    header := make([]byte, headerSize)
    if _, err := s.file.ReadAt(header, offset); err != nil {
        return nil, nil, err
    }
    keySize := binary.LittleEndian.Uint32(header[12:])
    valSize := binary.LittleEndian.Uint32(header[16:])

    data := make([]byte, keySize+valSize)
    if _, err := s.file.ReadAt(data, offset+headerSize); err != nil {
        return nil, nil, err
    }
    return data[:keySize], data[keySize:], nil
}
