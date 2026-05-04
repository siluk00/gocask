package storage

import "fmt"

type Store struct {
	dir     string
	segment *segment
	keydir  *keydir
}

func Open(dir string) (*Store, error) {
	seg, err := openSegment(dir + "/data.log")
	if err != nil {
		return nil, fmt.Errorf("open segment: %w", err)
	}
	return &Store{
		dir:     dir,
		segment: seg,
		keydir:  newKeydir(),
	}, nil
}

func (s *Store) Put(key, value []byte) error {
	offset, err := s.segment.write(key, value)
	if err != nil {
		return fmt.Errorf("put: %w", err)
	}
	s.keydir.put(string(key), keydirEntry{
		segmentPath: s.dir + "/data.log",
		offset:      offset,
	})
	return nil
}

func (s *Store) Get(key []byte) ([]byte, error) {
	entry, ok := s.keydir.get(string(key))
	if !ok {
		return nil, fmt.Errorf("key not found")
	}
	_, value, err := s.segment.readAt(entry.offset)
	if err != nil {
		return nil, fmt.Errorf("get: %w", err)
	}
	return value, nil
}

func (s *Store) Delete(key []byte) error {
	// tombstone: escreve um registro com value nil
	_, err := s.segment.write(key, nil)
	if err != nil {
		return err
	}
	s.keydir.delete(string(key))
	return nil
}

func (s *Store) Close() error {
	return s.segment.file.Close()
}
