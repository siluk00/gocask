package storage

import (
	"encoding/binary"
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

var (
	counter uint64
	pid     = os.Getpid()
	//host    = getHostHash()
)

/*type OffBoundaryOffsetError struct {
	AttemptedOffset int64
	FileSize        int64
	RequiredBytes   int64
	SegmentName     string
}

error sent when segment offst is greater than max file size
func (e *OffBoundaryOffsetError) Error() string {
	return fmt.Sprintf("segment %s: write overflow - offset %d + neewded %d > file size %d", e.SegmentName, e.AttemptedOffset, e.RequiredBytes, e.FileSize)
}*/

type Segment struct {
	File        *os.File
	Offset      int64
	NamePath    string
	syncOnWrite bool
}

func OpenSegment(path string, syncOnWrite bool) (*Segment, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return &Segment{File: f, Offset: info.Size(), syncOnWrite: syncOnWrite, NamePath: path}, nil
}

// Writes to segment and returns the new offset
func (s *Segment) Write(key, value []byte) (int64, error) {
	data := EncodeRecord(key, value)
	dataLen := int64(len(data))
	
	// Atomically reserve space in the file
	offset := atomic.AddInt64(&s.Offset, dataLen) - dataLen

	_, err := s.File.WriteAt(data, offset)
	if err != nil {
		return 0, err
	}

	if s.syncOnWrite {
		s.File.Sync()
	}

	return offset, nil
}

// returns key, value and error
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

func (s *Segment) ToReadOnly() error {
	err := s.File.Close()
	if err != nil {
		return err
	}

	file, err := os.OpenFile(s.NamePath, os.O_RDONLY, 0o644)
	if err != nil {
		return err
	}

	s.File = file
	return nil
}

func GenerateSegmentName() string {
	counter := atomic.AddUint64(&counter, 1)
	return fmt.Sprintf("%019d_%05d_%012d.seg", time.Now().Unix(), pid, counter)
}

// just in case of distributed system
/* func getHostHash() uint32 {
	h, _ := os.Hostname()
	hasher := fnv.New32a()
	hasher.Write([]byte(h))
	return hasher.Sum32()
} */
