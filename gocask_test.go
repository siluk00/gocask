package gocask

import (
	"bytes"
	"os"
	"testing"
)

func TestLibraryAPI(t *testing.T) {
	dir := "test_library_data"
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	cfg := Config{Dir: dir}
	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("Failed to open library: %v", err)
	}
	defer db.Close()

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

		db.Put(key, val)
		if err := db.Delete(key); err != nil {
			t.Errorf("Delete failed: %v", err)
		}

		_, err := db.Get(key)
		if err == nil {
			t.Error("Expected error for deleted key")
		}
	})
}
