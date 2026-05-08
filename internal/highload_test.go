package gocask

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"
)

func TestHighLoadRace(t *testing.T) {
	dir := "test_high_load_data"
	_ = os.RemoveAll(dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	// Small MaxFileSize to force frequent rotations during high load
	cfg := Config{Dir: dir, MaxFileSize: 10 * 1024}
	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("Failed to open library: %v", err)
	}
	defer func() { _ = db.Close() }()

	const numGoroutines = 50
	const opsPerGoroutine = 500
	var wg sync.WaitGroup

	// Mix of concurrent Writes, Reads, and Deletes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := []byte(fmt.Sprintf("key-%d-%d", id, j))
				val := []byte(fmt.Sprintf("value-%d-%d", id, j))

				// Put
				if err := db.Put(key, val); err != nil {
					t.Errorf("Put failed: %v", err)
				}

				// Immediate Get
				got, err := db.Get(key)
				if err != nil {
					t.Errorf("Get failed: %v", err)
				} else if !bytes.Equal(got, val) {
					t.Errorf("Data mismatch: expected %s, got %s", val, got)
				}

				// Occasional Delete
				if j%10 == 0 {
					if err := db.Delete(key); err != nil {
						t.Errorf("Delete failed: %v", err)
					}
					// Verify deletion
					_, err := db.Get(key)
					if err == nil {
						t.Errorf("Expected error for deleted key %s", string(key))
					}
				}
			}
		}(i)
	}

	wg.Wait()
}
