package storage

import "sync"

type KeydirEntry struct {
	SegmentPath string
	Offset      int64
	ValueSize   uint32
	Timestamp   int64
}

type Keydir struct {
	mu      sync.RWMutex
	Entries map[string]KeydirEntry
}

func NewKeydir() *Keydir {
	return &Keydir{Entries: make(map[string]KeydirEntry)}
}

func (k *Keydir) Put(key string, entry KeydirEntry) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.Entries[key] = entry
}

func (k *Keydir) Get(key string) (KeydirEntry, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	e, ok := k.Entries[key]
	return e, ok
}

func (k *Keydir) Delete(key string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	delete(k.Entries, key)
}
