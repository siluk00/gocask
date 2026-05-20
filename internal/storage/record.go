package storage

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"time"
)

const (
	HeaderSize   = 4 + 8 + 4 + 4 // crc + timestamp + keySize + valSize
	TombstoneBit = uint32(1) << 31
	HintSize     = 8 + 4 + 4 + 8 // timestamp + keySize + valSize + offset
)

type Record struct {
	CRC       uint32
	Timestamp int64
	KeySize   uint32
	ValSize   uint32
	Key       []byte
	Value     []byte
}

func EncodeRecord(key, value []byte) ([]byte, error) {
	kSize := uint32(len(key))
	vSize := uint32(len(value))
	if vSize >= TombstoneBit {
		return nil, fmt.Errorf("value size exceeds maximum allowed size (2GB)")
	}
	ts := time.Now().UnixNano()

	buf := make([]byte, HeaderSize+kSize+vSize)
	binary.LittleEndian.PutUint64(buf[4:], uint64(ts))
	binary.LittleEndian.PutUint32(buf[12:], kSize)
	if value != nil {
		binary.LittleEndian.PutUint32(buf[16:], vSize)
		copy(buf[HeaderSize+kSize:], value)
	} else {
		binary.LittleEndian.PutUint32(buf[16:], TombstoneBit)
	}
	copy(buf[HeaderSize:], key)

	crc := crc32.ChecksumIEEE(buf[4:])
	binary.LittleEndian.PutUint32(buf[0:], crc)
	return buf, nil
}

func DecodeRecord(data []byte) (*Record, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("data too short for header")
	}

	crc := binary.LittleEndian.Uint32(data[0:4])
	ts := int64(binary.LittleEndian.Uint64(data[4:12]))
	kSize := binary.LittleEndian.Uint32(data[12:16])
	vSize := binary.LittleEndian.Uint32(data[16:20])

	realVSize := vSize
	if vSize == TombstoneBit {
		realVSize = 0
	}

	actualCRC := crc32.ChecksumIEEE(data[4 : HeaderSize+kSize+realVSize])
	if crc != actualCRC {
		return nil, fmt.Errorf("CRC mismatch")
	}

	if uint32(len(data)) < HeaderSize+kSize+realVSize {
		return nil, fmt.Errorf("data too short for key/value")
	}

	key := make([]byte, kSize)
	copy(key, data[HeaderSize:HeaderSize+kSize])

	var value []byte
	if vSize != TombstoneBit {
		value = make([]byte, vSize)
		copy(value, data[HeaderSize+kSize:HeaderSize+kSize+vSize])
	}

	return &Record{
		CRC:       crc,
		Timestamp: ts,
		KeySize:   kSize,
		ValSize:   vSize,
		Key:       key,
		Value:     value,
	}, nil
}

type Hint struct {
	Timestamp int64
	KeySize   uint32
	ValSize   uint32
	Offset    int64
	Key       []byte
}

func EncodeHint(h Hint) []byte {
	buf := make([]byte, HintSize+len(h.Key))
	binary.LittleEndian.PutUint64(buf[0:], uint64(h.Timestamp))
	binary.LittleEndian.PutUint32(buf[8:], h.KeySize)
	binary.LittleEndian.PutUint32(buf[12:], h.ValSize)
	binary.LittleEndian.PutUint64(buf[16:], uint64(h.Offset))
	copy(buf[HintSize:], h.Key)
	return buf
}

func DecodeHint(data []byte) (*Hint, error) {
	if len(data) < HintSize {
		return nil, fmt.Errorf("data too short for hint")
	}

	ts := int64(binary.LittleEndian.Uint64(data[0:8]))
	kSize := binary.LittleEndian.Uint32(data[8:12])
	vSize := binary.LittleEndian.Uint32(data[12:16])
	off := int64(binary.LittleEndian.Uint64(data[16:24]))

	if uint32(len(data)) < HintSize+kSize {
		return nil, fmt.Errorf("data too short for hint key")
	}

	key := make([]byte, kSize)
	copy(key, data[HintSize:HintSize+kSize])

	return &Hint{
		Timestamp: ts,
		KeySize:   kSize,
		ValSize:   vSize,
		Offset:    off,
		Key:       key,
	}, nil
}

type Metadata struct {
	Version        uint32
	CompactionDone bool
	ClosedCleanly  bool
	HintDone       bool
	ActiveSeg      string
	LastCompaction int64
	CreatedAt      int64
	ConfigFlags    byte
}

const MaxSegName = 256
const MetaHeaderSize = 4 + 4 + 1 + 1 + 8 + 8 + 4 // 30 bytes

func EncodeMetadata(m Metadata) []byte {
	buf := make([]byte, MetaHeaderSize+MaxSegName)

	binary.LittleEndian.PutUint32(buf[4:], m.Version)

	var flags byte
	if m.CompactionDone {
		flags |= 1
	}
	if m.ClosedCleanly {
		flags |= 2
	}
	if m.HintDone {
		flags |= 4
	}
	buf[8] = flags
	buf[9] = m.ConfigFlags

	binary.LittleEndian.PutUint64(buf[10:], uint64(m.LastCompaction))
	binary.LittleEndian.PutUint64(buf[18:], uint64(m.CreatedAt))

	actualLen := len(m.ActiveSeg)
	if actualLen > MaxSegName {
		actualLen = MaxSegName
	}
	binary.LittleEndian.PutUint32(buf[26:], uint32(actualLen))
	copy(buf[MetaHeaderSize:], m.ActiveSeg[:actualLen])

	crc := crc32.ChecksumIEEE(buf[4:])
	binary.LittleEndian.PutUint32(buf[0:], crc)
	return buf
}

func DecodeMetadata(data []byte) (*Metadata, error) {
	if len(data) < MetaHeaderSize+MaxSegName {
		return nil, fmt.Errorf("data too short for metadata")
	}

	crc := binary.LittleEndian.Uint32(data[0:4])
	actualCRC := crc32.ChecksumIEEE(data[4:])
	if crc != actualCRC {
		return nil, fmt.Errorf("CRC mismatch")
	}

	version := binary.LittleEndian.Uint32(data[4:8])
	flags := data[8]
	configFlags := data[9]
	lastComp := int64(binary.LittleEndian.Uint64(data[10:18]))
	created := int64(binary.LittleEndian.Uint64(data[18:26]))
	segLen := binary.LittleEndian.Uint32(data[26:30])

	if segLen > MaxSegName {
		segLen = MaxSegName
	}

	activeSeg := string(data[MetaHeaderSize : MetaHeaderSize+segLen])

	return &Metadata{
		Version:        version,
		CompactionDone: flags&1 != 0,
		ClosedCleanly:  flags&2 != 0,
		HintDone:       flags&4 != 0,
		ActiveSeg:      activeSeg,
		LastCompaction: lastComp,
		CreatedAt:      created,
		ConfigFlags:    configFlags,
	}, nil
}
