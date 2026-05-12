package storage

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"os"
	"testing"
)

func TestEncodeMeta(t *testing.T) {
	m := Metadata{
		Version:        1,
		CompactionDone: true,
		ClosedCleanly:  false,
		ActiveSeg:      "0000000000000000001_12345_000000000001.seg",
		LastCompaction: 1620000000,
		CreatedAt:      1610000000,
		ConfigFlags:    0xAA,
	}

	buf := EncodeMetadata(m)

	if len(buf) != MetaHeaderSize+MaxSegName {
		t.Errorf("Expected length %d, got %d", MetaHeaderSize+MaxSegName, len(buf))
	}

	// Verify CRC
	expectedCRC := crc32.ChecksumIEEE(buf[4:])
	actualCRC := binary.LittleEndian.Uint32(buf[0:4])
	if expectedCRC != actualCRC {
		t.Errorf("CRC mismatch: expected %d, got %d", expectedCRC, actualCRC)
	}

	decoded, err := DecodeMetadata(buf)
	if err != nil {
		t.Fatalf("DecodeMetadata failed: %v", err)
	}

	if decoded.Version != m.Version || decoded.ActiveSeg != m.ActiveSeg || decoded.ConfigFlags != m.ConfigFlags {
		t.Errorf("Decoded metadata mismatch")
	}
}

func TestSegment(t *testing.T) {
	path := "test_segment.log"
	defer func() { _ = os.Remove(path) }()

	seg, err := OpenSegment(path, false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = seg.File.Close() }()

	k, v := []byte("key"), []byte("value")
	off, err := seg.Write(k, v)
	if err != nil {
		t.Fatal(err)
	}

	rk, rv, err := seg.ReadAt(off)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(k, rk) || !bytes.Equal(v, rv) {
		t.Error("data mismatch")
	}

	t.Run("Tombstone", func(t *testing.T) {
		tk := []byte("tombstone-key")
		toff, err := seg.Write(tk, nil)
		if err != nil {
			t.Fatal(err)
		}

		_, _, err = seg.ReadAt(toff)
		if err != ErrTombstone {
			t.Errorf("expected ErrTombstone, got %v", err)
		}
	})
}

func TestKeydir(t *testing.T) {
	kd := NewKeydir()
	key := "foo"
	entry := KeydirEntry{SegmentPath: "test", Offset: 100}

	kd.Put(key, entry)

	got, ok := kd.Get(key)
	if !ok {
		t.Fatal("Expected to find key")
	}
	if got.Offset != 100 {
		t.Errorf("Expected offset 100, got %d", got.Offset)
	}

	kd.Delete(key)
	_, ok = kd.Get(key)
	if ok {
		t.Error("Expected key to be deleted")
	}
}
