package storage

import (
	"bytes"
	"testing"
)

// decode of encoding must be identity
func TestRecordEncoding(t *testing.T) {
	tests := []struct {
		name  string
		key   []byte
		value []byte
	}{
		{"Normal", []byte("key1"), []byte("value1")},
		{"EmptyValue", []byte("key2"), []byte("")},
		{"Tombstone", []byte("key3"), nil},
		{"LargeValue", []byte("key4"), bytes.Repeat([]byte("a"), 1024)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := EncodeRecord(tt.key, tt.value)
			if err != nil {
				t.Fatalf("Failed to encode record: %v", err)
			}

			decoded, err := DecodeRecord(encoded)
			if err != nil {
				t.Fatalf("Failed to decode record: %v", err)
			}

			if !bytes.Equal(decoded.Key, tt.key) {
				t.Errorf("Expected key %s, got %s", tt.key, decoded.Key)
			}

			if tt.value == nil {
				if decoded.Value != nil {
					t.Errorf("Expected nil value for tombstone, got %s", decoded.Value)
				}
				if decoded.ValSize != TombstoneBit {
					t.Errorf("Expected TombstoneBit in ValSize, got %d", decoded.ValSize)
				}
			} else {
				if !bytes.Equal(decoded.Value, tt.value) {
					t.Errorf("Expected value %s, got %s", tt.value, decoded.Value)
				}
				if decoded.ValSize != uint32(len(tt.value)) {
					t.Errorf("Expected ValSize %d, got %d", len(tt.value), decoded.ValSize)
				}
			}
		})
	}
}

// Corrupted CRC must be detected on reading
func TestCorruptedCRC(t *testing.T) {
	key := []byte("test-key")
	value := []byte("test-value")

	encoded, err := EncodeRecord(key, value)
	if err != nil {
		t.Fatalf("Failed to encode record: %v", err)
	}

	// Corrupt the CRC (first 4 bytes)
	encoded[0] ^= 0xFF

	_, err = DecodeRecord(encoded)
	if err == nil {
		t.Fatal("Expected error due to corrupted CRC, but got nil")
	}

	if err.Error() != "CRC mismatch" {
		t.Errorf("Expected 'CRC mismatch' error, got: %v", err)
	}
}
