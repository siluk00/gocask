package storage

import (
	"encoding/binary"
	"os"
	"sync"
	"time"
)

type MetadataFile struct {
	mu   sync.Mutex
	Data Metadata
	Path string
}

func (m *MetadataFile) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	encoded := EncodeMetadata(m.Data)
	return os.WriteFile(m.Path, encoded, 0644)
}

func OpenMetadata(path string) (*MetadataFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &MetadataFile{
				Path: path,
				Data: Metadata{Version: 1, CreatedAt: time.Now().UnixNano()},
			}, nil
		}
		return nil, err
	}

	meta, err := DecodeMetadata(data)
	if err != nil {
		return nil, err
	}

	return &MetadataFile{
		Data: *meta,
		Path: path,
	}, nil
}

func Recover(dir string, keydir *Keydir) error {
	metaPath := dir + "/" + GenerateMetaName()
	meta, err := OpenMetadata(metaPath)
	if err != nil {
		return err
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var segments []string
	for _, f := range files {
		if !f.IsDir() && (len(f.Name()) > 4 && f.Name()[len(f.Name())-4:] == ".seg") {
			segments = append(segments, f.Name())
		}
	}
	// Sort segments chronologically
	for i := 0; i < len(segments); i++ {
		for j := i + 1; j < len(segments); j++ {
			if segments[i] > segments[j] {
				segments[i], segments[j] = segments[j], segments[i]
			}
		}
	}

	for _, segName := range segments {
		segPath := dir + "/" + segName
		hintPath := segPath[:len(segPath)-4] + ".hint"

		if _, err := os.Stat(hintPath); err == nil && meta.Data.HintDone {
			// Fast path: load from hint file
			if err := loadKeydirFromSingleHint(hintPath, keydir); err != nil {
				return err
			}
		} else {
			// Slow path: scan segment and create hint
			if err := scanSegmentAndCreateHint(segPath, keydir); err != nil {
				return err
			}
		}
	}

	meta.Data.HintDone = true
	return meta.Save()
}

func loadKeydirFromSingleHint(hPath string, keydir *Keydir) error {
	data, err := os.ReadFile(hPath)
	if err != nil {
		return err
	}

	segPath := hPath[:len(hPath)-5] + ".seg"
	offset := int64(0)
	for offset < int64(len(data)) {
		hint, err := DecodeHint(data[offset:])
		if err != nil {
			return err
		}

		keydir.Put(string(hint.Key), KeydirEntry{
			SegmentPath: segPath,
			Offset:      hint.Offset,
			ValueSize:   hint.ValSize,
			Timestamp:   hint.Timestamp,
		})

		offset += int64(HintSize + len(hint.Key))
	}
	return nil
}

func scanSegmentAndCreateHint(segPath string, keydir *Keydir) error {
	f, err := os.Open(segPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	hintPath := segPath[:len(segPath)-4] + ".hint"
	hintFile, err := os.Create(hintPath)
	if err != nil {
		return err
	}
	defer func() { _ = hintFile.Close() }()

	offset := int64(0)
	for {
		header := make([]byte, HeaderSize)
		_, err := f.ReadAt(header, offset)
		if err != nil {
			break
		}

		kSize := binary.LittleEndian.Uint32(header[12:16])
		vSize := binary.LittleEndian.Uint32(header[16:20])
		ts := int64(binary.LittleEndian.Uint64(header[4:12]))

		totalSize := int64(HeaderSize + kSize)
		if vSize != TombstoneBit {
			totalSize += int64(vSize)
		}

		key := make([]byte, kSize)
		_, err = f.ReadAt(key, offset+HeaderSize)
		if err != nil {
			break
		}

		if vSize == TombstoneBit {
			keydir.Delete(string(key))
		} else {
			keydir.Put(string(key), KeydirEntry{
				SegmentPath: segPath,
				Offset:      offset,
				ValueSize:   vSize,
				Timestamp:   ts,
			})

			hint := Hint{
				Timestamp: ts,
				KeySize:   kSize,
				ValSize:   vSize,
				Offset:    offset,
				Key:       key,
			}
			_, err = hintFile.Write(EncodeHint(hint))
			if err != nil {
				return err
			}
		}
		offset += totalSize
	}
	return nil
}
