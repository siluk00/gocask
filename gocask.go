package gocask

import (
	"fmt"

	"github.com/siluk00/gocask/storage"
)

// Bitcask defines the public API for the store.
type Bitcask interface {
	Put(key, value []byte) error
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
	Close() error
}

// Config holds configuration for the gocask store.
type Config struct {
	Dir string
}

// Store is the concrete implementation of the Bitcask interface.
type Store struct {
	config  Config
	segment *storage.Segment
	keydir  *storage.Keydir
}

// Open initializes the store with the given configuration.
func Open(cfg Config) (*Store, error) {
	if cfg.Dir == "" {
		return nil, fmt.Errorf("directory is required")
	}
	dataPath := cfg.Dir + "/data.log"
	seg, err := storage.OpenSegment(dataPath)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	return &Store{
		config:  cfg,
		segment: seg,
		keydir:  storage.NewKeydir(),
	}, nil
}

func (s *Store) Put(key, value []byte) error {
	offset, err := s.segment.Write(key, value)
	if err != nil {
		return err
	}
	s.keydir.Put(string(key), storage.KeydirEntry{
		SegmentPath: s.config.Dir + "/data.log",
		Offset:      offset,
	})
	return nil
}


func (s *Store) Get(key []byte) ([]byte, error) {
	entry, ok := s.keydir.Get(string(key))
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	_, value, err := s.segment.ReadAt(entry.Offset)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (s *Store) Delete(key []byte) error {
	_, err := s.segment.Write(key, nil)
	if err != nil {
		return err
	}
	s.keydir.Delete(string(key))
	return nil
}

func (s *Store) Close() error {
	return s.segment.File.Close()
}
