package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// SegmentStore is the persistence layer for sealed segment files and their
// index sidecars. The active segment and WAL are always node-local; this
// interface covers only sealed data.
//
// File names are base names without a directory component, e.g.
// "seg_00000001.alog" or "seg_00000001.alog.bidx".
//
// Implementations must be safe for concurrent use.
type SegmentStore interface {
	// Put writes the named file. For remote stores (S3, GCS) this uploads the
	// file; for LocalStore it is a no-op because the file is already in the
	// data directory. Called once per file after sealing and index builds.
	Put(name string, r io.Reader) error

	// Get returns a reader for the named file. For remote stores that cache
	// locally, Get downloads to the local cache on a miss before returning.
	// Returns os.ErrNotExist if the file is not present.
	Get(name string) (io.ReadCloser, error)

	// Delete removes the named file. Missing files are silently ignored.
	Delete(name string) error

	// List returns base names of all segment data files (*.alog) in the store.
	List() ([]string, error)
}

// LocalStore is a SegmentStore backed by a local filesystem directory.
// It is the default for single-node deployments without remote object storage.
//
// Put is a deliberate no-op: SegmentWriter creates the segment file in dir
// before sealing, so the file is already present when Put would be called.
// Remote stores override Put to upload to object storage.
type LocalStore struct {
	dir string
}

func NewLocalStore(dir string) *LocalStore {
	return &LocalStore{dir: dir}
}

func (s *LocalStore) Put(_ string, _ io.Reader) error { return nil }

func (s *LocalStore) Get(name string) (io.ReadCloser, error) {
	f, err := os.Open(filepath.Join(s.dir, name))
	if err != nil {
		return nil, fmt.Errorf("localstore: get %s: %w", name, err)
	}
	return f, nil
}

func (s *LocalStore) Delete(name string) error {
	err := os.Remove(filepath.Join(s.dir, name))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("localstore: delete %s: %w", name, err)
	}
	return nil
}

func (s *LocalStore) List() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("localstore: list: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".alog") {
			names = append(names, e.Name())
		}
	}
	return names, nil
}
