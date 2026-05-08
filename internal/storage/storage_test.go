package storage

import (
	"bytes"
	"os"
	"testing"
)

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
