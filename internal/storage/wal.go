package storage

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	walMagic    = uint32(0xABCD1234)
	walFileName = "amber.wal"

	walHeaderSize = 12
)

var (
	ErrWALCorrupted = errors.New("wal: corrupted record")
	ErrWALBadMagic  = errors.New("wal: bad magic bytes")
	ErrWALBadCRC    = errors.New("wal: crc32 mismatch")
)

type WALRecord struct {
	Payload []byte
}

type WAL struct {
	mu   sync.Mutex
	file *os.File
	buf  *bufio.Writer
	path string
}

func OpenWAL(dir string) (*WAL, error) {
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("wal: mkdir %s: %w", dir, err)
	}

	path := filepath.Join(dir, walFileName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("wal: open %s: %w", path, err)
	}

	return &WAL{
		file: f,
		buf:  bufio.NewWriterSize(f, 64*1024),
		path: path,
	}, nil
}

func (w *WAL) Write(payload []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.writeRecord(payload); err != nil {
		return err
	}

	if err := w.buf.Flush(); err != nil {
		return fmt.Errorf("wal: flush: %w", err)
	}

	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("wal: sync: %w", err)
	}

	return nil
}

func (w *WAL) WriteBatch(payloads [][]byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, payload := range payloads {
		if err := w.writeRecord(payload); err != nil {
			return err
		}
	}

	if err := w.buf.Flush(); err != nil {
		return fmt.Errorf("wal: batch flush: %w", err)
	}

	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("wal: batch sync: %w", err)
	}

	return nil
}

func (w *WAL) writeRecord(payload []byte) error {
	crc := crc32.ChecksumIEEE(payload)
	length := uint32(len(payload))

	var header [walHeaderSize]byte
	binary.LittleEndian.PutUint32(header[0:4], walMagic)
	binary.LittleEndian.PutUint32(header[4:8], crc)
	binary.LittleEndian.PutUint32(header[8:12], length)

	if _, err := w.buf.Write(header[:]); err != nil {
		return fmt.Errorf("wal: write header: %w", err)
	}

	if _, err := w.buf.Write(payload); err != nil {
		return fmt.Errorf("wal: write payload: %w", err)
	}

	return nil
}

func (w *WAL) Replay(fn func(payload []byte) error) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return 0, fmt.Errorf("wal: replay seek: %w", err)
	}

	r := bufio.NewReader(w.file)
	count := 0

	for {
		var header [walHeaderSize]byte
		_, err := io.ReadFull(r, header[:])
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			break
		}
		if err != nil {
			return count, fmt.Errorf("wal: replay read header: %w", err)
		}

		magic := binary.LittleEndian.Uint32(header[0:4])
		if magic != walMagic {
			break
		}

		expectedCRC := binary.LittleEndian.Uint32(header[4:8])
		length := binary.LittleEndian.Uint32(header[8:12])

		payload := make([]byte, length)
		_, err = io.ReadFull(r, payload)
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			break
		}
		if err != nil {
			return count, fmt.Errorf("wal: replay read payload: %w", err)
		}

		actualCRC := crc32.ChecksumIEEE(payload)
		if actualCRC != expectedCRC {
			break
		}

		if err := fn(payload); err != nil {
			return count, fmt.Errorf("wal: replay handler: %w", err)
		}

		count++
	}

	if _, err := w.file.Seek(0, io.SeekEnd); err != nil {
		return count, fmt.Errorf("wal: replay seek to end: %w", err)
	}

	return count, nil
}

func (w *WAL) Truncate() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.buf.Flush(); err != nil {
		return fmt.Errorf("wal: truncate flush: %w", err)
	}

	if err := w.file.Truncate(0); err != nil {
		return fmt.Errorf("wal: truncate: %w", err)
	}

	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("wal: truncate seek: %w", err)
	}

	w.buf.Reset(w.file)

	return nil
}

func (w *WAL) Size() (int64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	info, err := w.file.Stat()
	if err != nil {
		return 0, fmt.Errorf("wal: stat: %w", err)
	}
	return info.Size(), nil
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.buf.Flush(); err != nil {
		return fmt.Errorf("wal: close flush: %w", err)
	}

	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("wal: close sync: %w", err)
	}

	if err := w.file.Close(); err != nil {
		return fmt.Errorf("wal: close: %w", err)
	}

	return nil
}
