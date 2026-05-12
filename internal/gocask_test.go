package gocask

import (
	"bytes"
	"os"
	"testing"
)

func TestLibraryAPI(t *testing.T) {
	dir := "test_library_data"
	_ = os.RemoveAll(dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	cfg := Config{Dir: dir}
	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("Failed to open library: %v", err)
	}
	defer func() { _ = db.Close() }()

	t.Run("Write and Read", func(t *testing.T) {
		key := []byte("api-key")
		val := []byte("api-value")

		if err := db.Put(key, val); err != nil {
			t.Errorf("Put failed: %v", err)
		}

		got, err := db.Get(key)
		if err != nil {
			t.Errorf("Get failed: %v", err)
		}
		if !bytes.Equal(got, val) {
			t.Errorf("Expected %s, got %s", val, got)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		key := []byte("delete-me")
		val := []byte("temporary")

		if err := db.Put(key, val); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if err := db.Delete(key); err != nil {
			t.Errorf("Delete failed: %v", err)
		}

		_, err := db.Get(key)
		if err == nil {
			t.Error("Expected error for deleted key")
		}
	})
}

func TestPersistence(t *testing.T) {
	dir, err := os.MkdirTemp("", "gocask-test-persistence")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	cfg := Config{Dir: dir, MaxFileSize: 1024}

	// 1. Initial write
	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("Failed to open: %v", err)
	}
	
	_ = db.Put([]byte("key1"), []byte("val1"))
	_ = db.Put([]byte("key2"), []byte("val2"))
	_ = db.Delete([]byte("key1"))
	_ = db.Close()

	// 2. Re-open and verify
	db2, err := Open(cfg)
	if err != nil {
		t.Fatalf("Failed to re-open: %v", err)
	}
	defer func() { _ = db2.Close() }()

	if _, err := db2.Get([]byte("key1")); err == nil {
		t.Error("key1 should be deleted")
	}

	val2, err := db2.Get([]byte("key2"))
	if err != nil || !bytes.Equal(val2, []byte("val2")) {
		t.Errorf("key2 mismatch: %v", err)
	}
}
