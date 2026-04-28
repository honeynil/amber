package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestManager(t *testing.T) (*SegmentManager, string) {
	t.Helper()
	dir := t.TempDir()
	sm, err := OpenSegmentManager(dir, DefaultRotationPolicy)
	if err != nil {
		t.Fatalf("OpenSegmentManager: %v", err)
	}
	t.Cleanup(func() { sm.Close() })
	return sm, dir
}

func writeN(t *testing.T, sm *SegmentManager, n int) {
	t.Helper()
	base := time.Now().UnixNano()
	for i := 0; i < n; i++ {
		data := []byte(fmt.Sprintf("record-%d", i))
		ts := base + int64(i)*int64(time.Millisecond)
		if err := sm.Write(data, ts); err != nil {
			t.Fatalf("Write[%d]: %v", i, err)
		}
	}
}

func TestSegmentManager_Open_CreatesStructure(t *testing.T) {
	dir := t.TempDir()
	sm, err := OpenSegmentManager(dir, DefaultRotationPolicy)
	if err != nil {
		t.Fatalf("OpenSegmentManager: %v", err)
	}
	defer sm.Close()

	if !fileExists(filepath.Join(dir, metaFileName)) {
		t.Error("meta.json not created")
	}
	if !fileExists(filepath.Join(dir, walFileName)) {
		t.Error("amber.wal not created")
	}
	if _, ok := sm.ActiveSegmentMeta(); !ok {
		t.Error("no active segment after open")
	}
}

func TestSegmentManager_Open_Idempotent(t *testing.T) {
	dir := t.TempDir()
	sm1, _ := OpenSegmentManager(dir, DefaultRotationPolicy)
	writeN(t, sm1, 10)
	sm1.Close()

	sm2, err := OpenSegmentManager(dir, DefaultRotationPolicy)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer sm2.Close()

	if _, ok := sm2.ActiveSegmentMeta(); !ok {
		t.Error("no active segment after reopen")
	}
}

func TestSegmentManager_Write_Single(t *testing.T) {
	sm, _ := newTestManager(t)
	if err := sm.Write([]byte("hello"), time.Now().UnixNano()); err != nil {
		t.Fatalf("Write: %v", err)
	}
}

func TestSegmentManager_Write_Many(t *testing.T) {
	sm, _ := newTestManager(t)
	writeN(t, sm, 100)
}

func TestSegmentManager_Rotation_ByRecordCount(t *testing.T) {
	dir := t.TempDir()
	sm, err := OpenSegmentManager(dir, RotationPolicy{MaxRecords: 10})
	if err != nil {
		t.Fatalf("OpenSegmentManager: %v", err)
	}
	defer sm.Close()

	writeN(t, sm, 25)

	if sealed := sm.Segments(); len(sealed) < 2 {
		t.Errorf("expected >=2 sealed segments, got %d", len(sealed))
	}
}

func TestSegmentManager_Rotation_ByBytes(t *testing.T) {
	dir := t.TempDir()
	sm, err := OpenSegmentManager(dir, RotationPolicy{MaxBytes: 200})
	if err != nil {
		t.Fatalf("OpenSegmentManager: %v", err)
	}
	defer sm.Close()

	base := time.Now().UnixNano()
	for i := 0; i < 20; i++ {
		data := []byte(fmt.Sprintf("record-with-some-content-%d-padding", i))
		sm.Write(data, base+int64(i))
	}

	if sealed := sm.Segments(); len(sealed) == 0 {
		t.Error("expected at least 1 sealed segment after byte limit rotation")
	}
}

func TestSegmentManager_Rotate_Manual(t *testing.T) {
	sm, _ := newTestManager(t)
	writeN(t, sm, 5)
	before := len(sm.Segments())

	if err := sm.Rotate(); err != nil {
		t.Fatalf("Rotate: %v", err)
	}

	if after := len(sm.Segments()); after != before+1 {
		t.Errorf("expected %d sealed segments, got %d", before+1, after)
	}
}

func TestSegmentManager_Rotate_EmptySegment_NoOp(t *testing.T) {
	sm, _ := newTestManager(t)
	before := len(sm.Segments())
	sm.Rotate()
	if after := len(sm.Segments()); after != before {
		t.Errorf("rotating empty segment should be no-op: %d -> %d", before, after)
	}
}

func TestSegmentManager_Meta_Persisted(t *testing.T) {
	dir := t.TempDir()
	sm1, _ := OpenSegmentManager(dir, RotationPolicy{MaxRecords: 5})
	writeN(t, sm1, 10)
	sm1.Close()

	sm2, err := OpenSegmentManager(dir, DefaultRotationPolicy)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer sm2.Close()

	if sealed := sm2.Segments(); len(sealed) == 0 {
		t.Error("sealed segments not persisted in meta.json")
	}
}

func TestSegmentManager_Meta_SealedHasTimestamps(t *testing.T) {
	dir := t.TempDir()
	sm, _ := OpenSegmentManager(dir, RotationPolicy{MaxRecords: 3})
	defer sm.Close()

	sm.Write([]byte("a"), int64(2_000_000))
	sm.Write([]byte("b"), int64(1_000_000))
	sm.Write([]byte("c"), int64(3_000_000))
	sm.Write([]byte("d"), int64(1_000_000))

	sealed := sm.Segments()
	if len(sealed) == 0 {
		t.Fatal("expected sealed segment")
	}
	s := sealed[0]
	if s.MinTS == 0 || s.MaxTS == 0 {
		t.Errorf("sealed segment has zero timestamps: min=%d max=%d", s.MinTS, s.MaxTS)
	}
	if s.MinTS > s.MaxTS {
		t.Errorf("minTS > maxTS: %d > %d", s.MinTS, s.MaxTS)
	}
}

func TestSegmentManager_WALRecovery(t *testing.T) {
	dir := t.TempDir()
	sm1, _ := OpenSegmentManager(dir, DefaultRotationPolicy)
	writeN(t, sm1, 5)
	sm1.Close()

	sm2, err := OpenSegmentManager(dir, DefaultRotationPolicy)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer sm2.Close()

	total := uint64(0)
	for _, seg := range sm2.Segments() {
		total += seg.RecordCount
	}
	if total == 0 {
		t.Error("all records lost after reopen")
	}
}

func TestSegmentManager_SegmentPath(t *testing.T) {
	sm, dir := newTestManager(t)
	writeN(t, sm, 5)
	sm.Rotate()

	for _, seg := range sm.Segments() {
		path := sm.SegmentPath(seg)
		if !fileExists(path) {
			t.Errorf("segment file not found: %s", path)
		}
		if want := filepath.Join(dir, seg.FileName); path != want {
			t.Errorf("path mismatch: got %s, want %s", path, want)
		}
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
