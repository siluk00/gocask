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

func TestCreateHintFilesOnInit(t *testing.T) {
	dir, err := os.MkdirTemp("", "gocask-test-hint-init")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	// 1. Create a segment and write some data
	seg1Path := dir + "/" + GenerateSegmentName()
	seg1, _ := OpenSegment(seg1Path, false)
	
	// Key "a" = "v1"
	_, _ = seg1.Write([]byte("a"), []byte("v1"))
	// Key "b" = "v1"
	_, _ = seg1.Write([]byte("b"), []byte("v1"))
	_ = seg1.File.Close()

	// 2. Create another segment with an update and a delete
	seg2Path := dir + "/" + GenerateSegmentName()
	seg2, _ := OpenSegment(seg2Path, false)
	
	// Key "a" = "v2" (update)
	offA2, _ := seg2.Write([]byte("a"), []byte("v2"))
	// Key "b" (delete)
	_, _ = seg2.Write([]byte("b"), nil)
	_ = seg2.File.Close()

	// 3. Run the recovery function
	kd := NewKeydir()
	err = Recover(dir, kd)
	if err != nil {
		t.Fatalf("Recover failed: %v", err)
	}

	// 4. Verify Keydir state
	// "a" should be "v2" from seg2
	entryA, ok := kd.Get("a")
	if !ok || entryA.Offset != offA2 || entryA.SegmentPath != seg2Path {
		t.Errorf("Key 'a' mismatch. Got offset %d, expected %d", entryA.Offset, offA2)
	}
	// "b" should be deleted
	_, ok = kd.Get("b")
	if ok {
		t.Error("Key 'b' should have been deleted")
	}

	// 5. Verify Hint Files exist and are valid
	hint1Path := seg1Path[:len(seg1Path)-4] + ".hint"
	hint2Path := seg2Path[:len(seg2Path)-4] + ".hint"

	if _, err := os.Stat(hint1Path); os.IsNotExist(err) {
		t.Error("Hint file 1 was not created")
	}
	if _, err := os.Stat(hint2Path); os.IsNotExist(err) {
		t.Error("Hint file 2 was not created")
	}

	// Verify Metadata
	metaPath := dir + "/" + GenerateMetaName()
	meta, err := OpenMetadata(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	if !meta.Data.HintDone {
		t.Error("Metadata HintDone should be true")
	}
}
