package bootstrap

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/hnlbs/amber/internal/index"
	"github.com/hnlbs/amber/internal/query"
	"github.com/hnlbs/amber/internal/storage"
)

func LoadSealedIndexes(
	exec *query.Executor,
	logManager, spanManager *storage.SegmentManager,
	logDir, spanDir string,
	log *slog.Logger,
) {
	workers := runtime.NumCPU()
	if workers < 2 {
		workers = 2
	}

	loadLogSegments(exec, logManager, logDir, workers, log)
	loadSpanSegments(exec, spanManager, spanDir, workers, log)
}

func loadLogSegments(
	exec *query.Executor,
	logManager *storage.SegmentManager,
	logDir string,
	workers int,
	log *slog.Logger,
) {
	segs := logManager.Segments()
	if len(segs) == 0 {
		return
	}

	jobs := make(chan storage.SegmentMeta, len(segs))
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Go(func() {
			for seg := range jobs {
				segPath := filepath.Join(logDir, seg.FileName)

				bidxPath := filepath.Join(logDir, seg.FileName+".bidx")
				if _, err := os.Stat(bidxPath); err != nil {
					if _, err := index.BuildLogBitmapIndex(segPath, log); err != nil {
						log.Warn("failed to build log bitmap on startup", "segment", seg.FileName, "err", err)
					}
				}

				fidxPath := filepath.Join(logDir, seg.FileName+".fidx")
				if _, err := os.Stat(fidxPath); err != nil {
					if _, err := index.BuildLogFTSIndex(segPath, log); err != nil {
						log.Warn("failed to build log fts on startup", "segment", seg.FileName, "err", err)
					}
				}

				if ribbon, err := index.LoadRibbonFilter(filepath.Join(logDir, seg.FileName+".filt")); err == nil {
					exec.RegisterLogRibbon(seg.FileName, ribbon)
				} else if ribbon, err := index.BuildLogRibbonFilter(segPath, log); err == nil {
					exec.RegisterLogRibbon(seg.FileName, ribbon)
				} else {
					log.Warn("failed to build log ribbon on startup", "segment", seg.FileName, "err", err)
				}

				if ribbon, err := index.LoadRibbonFilter(filepath.Join(logDir, seg.FileName+".fts.filt")); err == nil {
					exec.RegisterLogFTSRibbon(seg.FileName, ribbon)
				} else if ribbon, err := index.BuildLogFTSRibbon(segPath, log); err == nil {
					exec.RegisterLogFTSRibbon(seg.FileName, ribbon)
				} else {
					log.Warn("failed to build log fts ribbon on startup", "segment", seg.FileName, "err", err)
				}
			}
		})
	}

	for _, seg := range segs {
		jobs <- seg
	}
	close(jobs)
	wg.Wait()
}

func loadSpanSegments(
	exec *query.Executor,
	spanManager *storage.SegmentManager,
	spanDir string,
	workers int,
	log *slog.Logger,
) {
	segs := spanManager.Segments()
	if len(segs) == 0 {
		return
	}

	jobs := make(chan storage.SegmentMeta, len(segs))
	var wg sync.WaitGroup

	for range workers {
		wg.Go(func() {
			for seg := range jobs {
				segPath := filepath.Join(spanDir, seg.FileName)

				bidxPath := filepath.Join(spanDir, seg.FileName+".bidx")
				if _, err := os.Stat(bidxPath); err != nil {
					if _, err := index.BuildSpanBitmapIndex(segPath, log); err != nil {
						log.Warn("failed to build span bitmap on startup", "segment", seg.FileName, "err", err)
					}
				}

				if ribbon, err := index.LoadRibbonFilter(filepath.Join(spanDir, seg.FileName+".filt")); err == nil {
					exec.RegisterSpanRibbon(seg.FileName, ribbon)
				} else if ribbon, err := index.BuildSpanRibbonFilter(segPath, log); err == nil {
					exec.RegisterSpanRibbon(seg.FileName, ribbon)
				} else {
					log.Warn("failed to build span ribbon on startup", "segment", seg.FileName, "err", err)
				}
			}
		})
	}

	for _, seg := range segs {
		jobs <- seg
	}
	close(jobs)
	wg.Wait()
}

func SetupSealCallbacks(
	exec *query.Executor,
	logManager, spanManager *storage.SegmentManager,
	logDir, spanDir string,
	log *slog.Logger,
) {
	logManager.SetOnSeal(func(meta storage.SegmentMeta) {
		segPath := filepath.Join(logDir, meta.FileName)
		if _, err := index.BuildLogBitmapIndex(segPath, log); err != nil {
			log.Warn("seal: build log bitmap failed", "segment", meta.FileName, "err", err)
		}
		if _, err := index.BuildLogFTSIndex(segPath, log); err != nil {
			log.Warn("seal: build log fts failed", "segment", meta.FileName, "err", err)
		}
		if ribbon, err := index.BuildLogRibbonFilter(segPath, log); err == nil {
			exec.RegisterLogRibbon(meta.FileName, ribbon)
		}
		if ribbon, err := index.BuildLogFTSRibbon(segPath, log); err == nil {
			exec.RegisterLogFTSRibbon(meta.FileName, ribbon)
		} else {
			log.Warn("seal: build log fts ribbon failed", "segment", meta.FileName, "err", err)
		}
	})

	spanManager.SetOnSeal(func(meta storage.SegmentMeta) {
		segPath := filepath.Join(spanDir, meta.FileName)
		if _, err := index.BuildSpanBitmapIndex(segPath, log); err != nil {
			log.Warn("seal: build span bitmap failed", "segment", meta.FileName, "err", err)
		}
		if ribbon, err := index.BuildSpanRibbonFilter(segPath, log); err == nil {
			exec.RegisterSpanRibbon(meta.FileName, ribbon)
		}
	})
}
