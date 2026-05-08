package storage

import (
	"encoding/binary"
	"hash/crc32"
	"time"
)

const HeaderSize = 4 + 8 + 4 + 4 // crc + timestamp + keySize + valSize

type Record struct {
	CRC       uint32
	Timestamp int64
	KeySize   uint32
	ValSize   uint32
	Key       []byte
	Value     []byte
}

func EncodeRecord(key, value []byte) []byte {
	kSize := uint32(len(key))
	vSize := uint32(len(value))
	ts := time.Now().UnixNano()

	buf := make([]byte, HeaderSize+kSize+vSize)
	binary.LittleEndian.PutUint64(buf[4:], uint64(ts))
	binary.LittleEndian.PutUint32(buf[12:], kSize)
	binary.LittleEndian.PutUint32(buf[16:], vSize)
	copy(buf[HeaderSize:], key)
	copy(buf[HeaderSize+kSize:], value)

	crc := crc32.ChecksumIEEE(buf[4:])
	binary.LittleEndian.PutUint32(buf[0:], crc)
	return buf
}
