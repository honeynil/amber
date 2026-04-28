package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const metaFileName = "meta.json"

type SegmentMeta struct {
	ID          uint32 `json:"id"`
	FileName    string `json:"file_name"`
	MinTS       int64  `json:"min_ts"`
	MaxTS       int64  `json:"max_ts"`
	RecordCount uint64 `json:"record_count"`
	SizeBytes   int64  `json:"size_bytes"`
	Sealed      bool   `json:"sealed"`
}

type StoreMeta struct {
	NextSegmentID uint32        `json:"next_segment_id"`
	Segments      []SegmentMeta `json:"segments"`
}

func loadMeta(dir string) (*StoreMeta, error) {
	path := filepath.Join(dir, metaFileName)

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &StoreMeta{NextSegmentID: 1}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("meta: read %s: %w", path, err)
	}

	var m StoreMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("meta: parse %s: %w", path, err)
	}

	return &m, nil
}

func saveMeta(dir string, m *StoreMeta) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("meta: marshal: %w", err)
	}

	tmp := filepath.Join(dir, metaFileName+".tmp")
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("meta: write tmp: %w", err)
	}

	dst := filepath.Join(dir, metaFileName)
	if err := os.Rename(tmp, dst); err != nil {
		return fmt.Errorf("meta: rename: %w", err)
	}

	return nil
}

func segmentFileName(id uint32) string {
	return fmt.Sprintf("seg_%08d.alog", id)
}
