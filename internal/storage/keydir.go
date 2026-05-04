package storage

import "sync"

type keydirEntry struct {
	segmentPath string
	offset      int64
	valueSize   uint32
	timestamp   int64
}

type keydir struct {
	mu      sync.RWMutex
	entries map[string]keydirEntry
}

func newKeydir() *keydir {
	return &keydir{entries: make(map[string]keydirEntry)}
}

func (k *keydir) put(key string, entry keydirEntry) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.entries[key] = entry
}

func (k *keydir) get(key string) (keydirEntry, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	e, ok := k.entries[key]
	return e, ok
}

func (k *keydir) delete(key string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	delete(k.entries, key)
}
