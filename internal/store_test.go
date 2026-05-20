package gocask

import (
	"bytes"
	"os"
	"testing"
)

func TestPutAndGet(t *testing.T) {
	dir, err := os.MkdirTemp("", "gocask-test-put-get")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	cfg := Config{
		Dir: dir,
	}
	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	key := []byte("hello")
	val := []byte("world")

	if err := db.Put(key, val); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, err := db.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !bytes.Equal(got, val) {
		t.Errorf("Expected %s, got %s", val, got)
	}
}

func TestDelete(t *testing.T) {
	dir, err := os.MkdirTemp("", "gocask-test-delete")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	cfg := Config{Dir: dir}
	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	key := []byte("delete-me")
	val := []byte("temporary-value")

	if err := db.Put(key, val); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if err := db.Delete(key); err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	_, err = db.Get(key)
	if err == nil {
		t.Error("Expected error when getting deleted key")
	}
}

// multuiple puts and gets in sequence, all keys must be recoverable
func TestMultiplePuts(t *testing.T) {
	dir, err := os.MkdirTemp("", "gocask-test-multi")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	cfg := Config{Dir: dir}
	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	data := map[string]string{
		"k1": "v1",
		"k2": "v2",
		"k3": "v3",
	}

	for k, v := range data {
		if err := db.Put([]byte(k), []byte(v)); err != nil {
			t.Errorf("Put failed for %s: %v", k, err)
		}
	}

	for k, v := range data {
		got, err := db.Get([]byte(k))
		if err != nil {
			t.Errorf("Get failed for %s: %v", k, err)
		}
		if string(got) != v {
			t.Errorf("Expected %s, got %s", v, string(got))
		}
	}
}

func TestDeleteNonExistentKey(t *testing.T) {
	dir, err := os.MkdirTemp("", "gocask-test-nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	cfg := Config{Dir: dir}
	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	_, err = db.Get([]byte("not-here"))
	if err == nil {
		t.Error("Expected error for non-existent key")
	}
}

func TestPutOverwrite(t *testing.T) {
	dir, err := os.MkdirTemp("", "gocask-test-overwrite")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	cfg := Config{Dir: dir}
	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	key := []byte("overwrite-key")
	val1 := []byte("value-1")
	val2 := []byte("value-2")

	if err := db.Put(key, val1); err != nil {
		t.Fatalf("First Put failed: %v", err)
	}

	if err := db.Put(key, val2); err != nil {
		t.Fatalf("Second Put failed: %v", err)
	}

	got, err := db.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !bytes.Equal(got, val2) {
		t.Errorf("Expected %s, got %s", val2, got)
	}
}

// It should be put and get as an empty value, not "not founbd error"
func TestPutEmptyValue(t *testing.T) {
	dir, err := os.MkdirTemp("", "gocask-test-empty")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	cfg := Config{Dir: dir}
	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	key := []byte("empty-key")
	val := []byte("") // Empty value

	if err := db.Put(key, val); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, err := db.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Expected empty value, got %s", got)
	}
}

// put with key and values of like 64kb each, it shouln't truncate not even corrupt
func TestLargeKeyValue(t *testing.T) {
	dir, err := os.MkdirTemp("", "gocask-test-large")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	cfg := Config{Dir: dir}
	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	size := 64 * 1024
	key := make([]byte, size)
	val := make([]byte, size)
	for i := 0; i < size; i++ {
		key[i] = byte(i % 256)
		val[i] = byte((i + 1) % 256)
	}

	if err := db.Put(key, val); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, err := db.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !bytes.Equal(got, val) {
		t.Error("Large value mismatch")
	}
}
