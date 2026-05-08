package gocask

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestConcurrency(t *testing.T) {
	dir := "test_concurrency_data"
	_ = os.RemoveAll(dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	cfg := Config{Dir: dir, MaxFileSize: 1024 * 1024}
	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("Failed to open library: %v", err)
	}
	defer func() { _ = db.Close() }()

	const numGoroutines = 10
	const opsPerGoroutine = 100
	var wg sync.WaitGroup

	// Concurrent Writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := []byte(fmt.Sprintf("key-%d-%d", id, j))
				val := []byte(fmt.Sprintf("value-%d-%d", id, j))
				if err := db.Put(key, val); err != nil {
					t.Errorf("Put failed: %v", err)
				}
			}
		}(i)
	}
	wg.Wait()

	// Concurrent Reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := []byte(fmt.Sprintf("key-%d-%d", id, j))
				expected := []byte(fmt.Sprintf("value-%d-%d", id, j))
				got, err := db.Get(key)
				if err != nil {
					t.Errorf("Get failed for %s: %v", string(key), err)
					continue
				}
				if !bytes.Equal(got, expected) {
					t.Errorf("Expected %s, got %s", expected, got)
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestSegmentRotation(t *testing.T) {
	dir := "test_rotation_data"
	_ = os.RemoveAll(dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	// Set a very small MaxFileSize to trigger rotation quickly
	cfg := Config{Dir: dir, MaxFileSize: 50} 
	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("Failed to open library: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Perform multiple writes to trigger rotation
	for i := 0; i < 5; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		val := []byte(fmt.Sprintf("value-large-enough-to-trigger-%d", i))
		if err := db.Put(key, val); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	// Verify that multiple segment files were created
	files, err := filepath.Glob(filepath.Join(dir, "*.seg"))
	if err != nil {
		t.Fatal(err)
	}

	if len(files) <= 1 {
		t.Errorf("Expected multiple segment files, found %d", len(files))
	}

	// Verify we can still read all data
	for i := 0; i < 5; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		expected := []byte(fmt.Sprintf("value-large-enough-to-trigger-%d", i))
		got, err := db.Get(key)
		if err != nil {
			t.Errorf("Get failed for %s: %v", string(key), err)
			continue
		}
		if !bytes.Equal(got, expected) {
			t.Errorf("Expected %s, got %s", expected, got)
		}
	}
}
