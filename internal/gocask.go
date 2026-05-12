package gocask

import (
	"fmt"
	"os"
	"sync"

	"github.com/siluk00/gocask/internal/storage"
)

type ConfigFlags byte

const (
	SyncOnWrite ConfigFlags = 1 << iota
)

var ErrKeyNotFound = fmt.Errorf("not found")

// Bitcask defines the public API for the store.
type Bitcask interface {
	Put(key, value []byte) error
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
	Close() error
}

// Config holds configuration for the gocask store.
type Config struct {
	Dir         string
	MaxFileSize uint32
	Policy      ConfigFlags
}

// Store is the concrete implementation of the Bitcask interface.
type Store struct {
	config           Config
	segment          *storage.Segment
	readonlySegments map[string]*storage.Segment
	mu               sync.RWMutex
	keydir           *storage.Keydir
}

// Open initializes the store with the given configuration.
func Open(cfg Config) (*Store, error) {
	if cfg.Dir == "" {
		return nil, fmt.Errorf("directory is required")
	}
	// Ensure directory exists
	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	keydir := storage.NewKeydir()
	if err := storage.Recover(cfg.Dir, keydir); err != nil {
		return nil, err
	}

	readonlySegments := make(map[string]*storage.Segment)
	files, err := os.ReadDir(cfg.Dir)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if !f.IsDir() && (len(f.Name()) > 4 && f.Name()[len(f.Name())-4:] == ".seg") {
			path := cfg.Dir + "/" + f.Name()
			seg, err := storage.OpenSegment(path, SyncOnWrite&cfg.Policy != 0)
			if err != nil {
				return nil, err
			}
			if err := seg.ToReadOnly(); err != nil {
				return nil, err
			}
			readonlySegments[path] = seg
		}
	}

	dataPath := cfg.Dir + "/" + storage.GenerateSegmentName()
	seg, err := storage.OpenSegment(dataPath, SyncOnWrite&cfg.Policy != 0)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	return &Store{
		config:           cfg,
		segment:          seg,
		readonlySegments: readonlySegments,
		keydir:           keydir,
	}, nil
}

func (s *Store) Put(key, value []byte) error {
	s.mu.Lock()
	segment := s.segment
	offset, err := segment.Write(key, value)

	if err != nil {
		s.mu.Unlock()
		return err
	}

	s.keydir.Put(string(key), storage.KeydirEntry{
		SegmentPath: segment.NamePath,
		Offset:      offset,
		ValueSize:   uint32(len(value)),
	})

	s.mu.Unlock()

	if s.config.MaxFileSize > 0 && s.config.MaxFileSize <= uint32(offset) {
		s.mu.Lock()
		defer s.mu.Unlock()

		// Re-check if the segment is still the same (another goroutine might have rotated it)
		if segment.NamePath == s.segment.NamePath {
			err = segment.ToReadOnly()
			if err != nil {
				return err
			}

			s.readonlySegments[segment.NamePath] = segment
			dataPath := s.config.Dir + "/" + storage.GenerateSegmentName()
			s.segment, err = storage.OpenSegment(dataPath, SyncOnWrite&s.config.Policy != 0)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Store) Get(key []byte) ([]byte, error) {
	entry, ok := s.keydir.Get(string(key))
	if !ok {
		return nil, ErrKeyNotFound
	}

	s.mu.RLock()
	var seg *storage.Segment
	if entry.SegmentPath == s.segment.NamePath {
		seg = s.segment
	} else {
		seg = s.readonlySegments[entry.SegmentPath]
	}
	s.mu.RUnlock()

	if seg == nil {
		return nil, ErrKeyNotFound
	}

	_, value, err := seg.ReadAt(entry.Offset)
	if err != nil {
		if err == storage.ErrTombstone {
			return nil, ErrKeyNotFound
		}
		return nil, err
	}
	return value, nil
}

func (s *Store) Delete(key []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	segment := s.segment
	_, err := segment.Write(key, nil)
	if err != nil {
		return err
	}

	s.keydir.Delete(string(key))
	return nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var firstErr error
	if err := s.segment.File.Close(); err != nil {
		firstErr = err
	}

	for _, seg := range s.readonlySegments {
		if err := seg.File.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
