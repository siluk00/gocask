package storage

import (
	"encoding/binary"
	"hash/crc32"
	"time"
)

const headerSize = 4 + 8 + 4 + 4 // crc + timestamp + keySize + valSize

type record struct {
	CRC       uint32
	Timestamp int64
	KeySize   uint32
	ValSize   uint32
	Key       []byte
	Value     []byte
}

func encodeRecord(key, value []byte) []byte {
	keySize := uint32(len(key))
	valSize := uint32(len(value))
	ts := time.Now().UnixNano()

	buf := make([]byte, headerSize+keySize+valSize)
	binary.LittleEndian.PutUint64(buf[4:], uint64(ts))
	binary.LittleEndian.PutUint32(buf[12:], keySize)
	binary.LittleEndian.PutUint32(buf[16:], valSize)
	copy(buf[headerSize:], key)
	copy(buf[headerSize+keySize:], value)

	crc := crc32.ChecksumIEEE(buf[4:])
	binary.LittleEndian.PutUint32(buf[0:], crc)
	return buf
}
