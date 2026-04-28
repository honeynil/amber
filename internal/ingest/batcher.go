package ingest

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/hnlbs/amber/internal/index"
	"github.com/hnlbs/amber/internal/model"
	"github.com/hnlbs/amber/internal/storage"
)

type item struct {
	log  *model.LogEntry
	span *model.SpanEntry
}

type Batcher struct {
	logManager   *storage.SegmentManager
	spanManager  *storage.SegmentManager
	logSparse    *index.SparseIndex
	spanSparse   *index.SparseIndex
	indexer      ActiveIndexer
	batchSize    int
	batchTimeout time.Duration
	queue        chan item
	log          *slog.Logger
	wg           sync.WaitGroup
}

func NewBatcher(
	logManager *storage.SegmentManager,
	spanManager *storage.SegmentManager,
	logSparse *index.SparseIndex,
	spanSparse *index.SparseIndex,
	indexer ActiveIndexer,
	batchSize int,
	batchTimeout time.Duration,
	queueSize int,
	log *slog.Logger,
) *Batcher {
	return &Batcher{
		logManager:   logManager,
		spanManager:  spanManager,
		logSparse:    logSparse,
		spanSparse:   spanSparse,
		indexer:      indexer,
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		queue:        make(chan item, queueSize),
		log:          log,
	}
}

func (b *Batcher) Start(ctx context.Context) {
	b.wg.Add(1)
	go b.run(ctx)
}

func (b *Batcher) Wait() {
	b.wg.Wait()
}

func (b *Batcher) SendLog(entry model.LogEntry) error {
	b.queue <- item{log: &entry}
	return nil
}

func (b *Batcher) SendSpan(span model.SpanEntry) error {
	b.queue <- item{span: &span}
	return nil
}

func (b *Batcher) TrySendLog(entry model.LogEntry) bool {
	select {
	case b.queue <- item{log: &entry}:
		return true
	default:
		b.log.Warn("ingest queue full, dropping log entry",
			"entry_id", entry.ID.String(),
			"service", entry.Service,
		)
		return false
	}
}

var bufPool = sync.Pool{
	New: func() any { return &bytes.Buffer{} },
}

func (b *Batcher) run(ctx context.Context) {
	defer b.wg.Done()

	batch := make([]item, 0, b.batchSize)
	ticker := time.NewTicker(b.batchTimeout)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		b.processBatch(ctx, batch)
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			for {
				select {
				case it := <-b.queue:
					batch = append(batch, it)
					if len(batch) >= b.batchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}

		case it := <-b.queue:
			batch = append(batch, it)
			if len(batch) >= b.batchSize {
				flush()
				ticker.Reset(b.batchTimeout)
			}

		case <-ticker.C:
			flush()
		}
	}
}

func (b *Batcher) processBatch(_ context.Context, batch []item) {
	if len(batch) == 0 {
		return
	}

	logItems := make([]storage.BatchItem, 0, len(batch))
	spanItems := make([]storage.BatchItem, 0)
	var logEntries []*model.LogEntry
	var spanEntries []*model.SpanEntry

	for _, it := range batch {
		buf := bufPool.Get().(*bytes.Buffer)
		buf.Reset()

		var ts int64
		var writeErr error

		if it.log != nil {
			_, writeErr = it.log.WriteTo(buf)
			ts = it.log.Timestamp.UnixNano()
		} else if it.span != nil {
			_, writeErr = it.span.WriteTo(buf)
			ts = it.span.StartTime.UnixNano()
		}

		if writeErr != nil {
			b.log.Error("serialize entry", "err", writeErr)
			bufPool.Put(buf)
			continue
		}

		data := make([]byte, buf.Len())
		copy(data, buf.Bytes())
		bufPool.Put(buf)

		bi := storage.BatchItem{Data: data, TS: ts}
		if it.log != nil {
			logItems = append(logItems, bi)
			if b.indexer != nil {
				if logEntries == nil {
					logEntries = make([]*model.LogEntry, 0, len(batch))
				}
				logEntries = append(logEntries, it.log)
			}
		} else {
			spanItems = append(spanItems, bi)
			if b.indexer != nil {
				if spanEntries == nil {
					spanEntries = make([]*model.SpanEntry, 0, len(batch))
				}
				spanEntries = append(spanEntries, it.span)
			}
		}
	}

	if len(logItems) > 0 {
		if err := b.logManager.WriteBatch(logItems); err != nil {
			b.log.Error("log batch write failed", "err", err, "count", len(logItems))
		} else {
			updateSparseFromBatch(b.logSparse, b.logManager, logItems)
			if b.indexer != nil && len(logEntries) > 0 {
				b.indexer.IndexLogEntries(logEntries)
			}
			if err := b.logManager.Flush(); err != nil {
				b.log.Warn("log segment flush failed", "err", err)
			}
		}
	}

	if len(spanItems) > 0 {
		if err := b.spanManager.WriteBatch(spanItems); err != nil {
			b.log.Error("span batch write failed", "err", err, "count", len(spanItems))
		} else {
			updateSparseFromBatch(b.spanSparse, b.spanManager, spanItems)
			if b.indexer != nil && len(spanEntries) > 0 {
				b.indexer.IndexSpanEntries(spanEntries)
			}
			if err := b.spanManager.Flush(); err != nil {
				b.log.Warn("span segment flush failed", "err", err)
			}
		}
	}
}

func updateSparseFromBatch(sparse *index.SparseIndex, manager *storage.SegmentManager, items []storage.BatchItem) {
	activeMeta, ok := manager.ActiveSegmentMeta()
	if !ok {
		return
	}
	var minTS, maxTS int64
	for _, it := range items {
		if minTS == 0 || it.TS < minTS {
			minTS = it.TS
		}
		if it.TS > maxTS {
			maxTS = it.TS
		}
	}
	sparse.TouchRange(activeMeta.ID, activeMeta.FileName, minTS, maxTS)
}
