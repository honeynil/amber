package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	ambergrpc "github.com/hnlbs/amber/internal/api/grpc"
	amberhttp "github.com/hnlbs/amber/internal/api/http"
	"github.com/hnlbs/amber/internal/bootstrap"
	"github.com/hnlbs/amber/internal/config"
	"github.com/hnlbs/amber/internal/index"
	"github.com/hnlbs/amber/internal/ingest"
	"github.com/hnlbs/amber/internal/query"
	"github.com/hnlbs/amber/internal/retention"
	"github.com/hnlbs/amber/internal/storage"
)

func main() {
	cfgPath := "config.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := setupLogger(cfg.Log)
	log.Info("amber starting",
		"data_dir", cfg.Storage.DataDir,
		"http_addr", cfg.API.HTTPAddr,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rotationPolicy := storage.RotationPolicy{
		MaxRecords: cfg.Storage.SegmentMaxRecords,
		MaxBytes:   cfg.Storage.SegmentMaxBytes,
	}

	logDir := filepath.Join(cfg.Storage.DataDir, "logs")
	spanDir := filepath.Join(cfg.Storage.DataDir, "spans")

	logManager, err := storage.OpenSegmentManager(logDir, rotationPolicy)
	if err != nil {
		log.Error("failed to open log segment manager", "err", err)
		os.Exit(1)
	}
	defer func() { _ = logManager.Close() }()

	spanManager, err := storage.OpenSegmentManager(spanDir, rotationPolicy)
	if err != nil {
		log.Error("failed to open span segment manager", "err", err)
		os.Exit(1)
	}
	defer func() { _ = spanManager.Close() }()

	log.Info("storage opened")

	logSparse, err := index.LoadSparseIndex(logDir)
	if err != nil {
		log.Error("failed to load log sparse index", "err", err)
		os.Exit(1)
	}

	spanSparse, err := index.LoadSparseIndex(spanDir)
	if err != nil {
		log.Error("failed to load span sparse index", "err", err)
		os.Exit(1)
	}

	exec := query.NewExecutorWithCache(
		logManager, spanManager, logSparse, spanSparse,
		logDir, spanDir, cfg.Storage.IndexCacheSize,
	)

	bootstrap.SetupSealCallbacks(exec, logManager, spanManager, logDir, spanDir, log)

	go func() {
		bootstrap.LoadSealedIndexes(exec, logManager, spanManager, logDir, spanDir, log)
		log.Info("sealed indexes loaded")
	}()

	batcher := ingest.NewBatcher(
		logManager,
		spanManager,
		logSparse,
		spanSparse,
		exec,
		cfg.Ingest.BatchSize,
		cfg.Ingest.BatchTimeout,
		cfg.Ingest.QueueSize,
		log,
	)
	batcher.Start(ctx)

	if cfg.Retention.MaxAge > 0 || cfg.Retention.MaxBytes > 0 || cfg.Retention.MaxSegments > 0 {
		policy := retention.Policy{
			MaxAge:        cfg.Retention.MaxAge,
			MaxTotalBytes: cfg.Retention.MaxBytes,
			MaxSegments:   cfg.Retention.MaxSegments,
		}
		interval := cfg.Retention.Interval
		if interval == 0 {
			interval = time.Hour
		}
		logCleaner := retention.NewCleaner(logManager, logSparse, policy, logDir, log)
		spanCleaner := retention.NewCleaner(spanManager, spanSparse, policy, spanDir, log)
		logCleaner.SetOnDelete(exec.InvalidateLogSegment)
		spanCleaner.SetOnDelete(exec.InvalidateSpanSegment)
		go logCleaner.StartLoop(interval, ctx.Done())
		go spanCleaner.StartLoop(interval, ctx.Done())
		log.Info("retention enabled", "max_age", cfg.Retention.MaxAge, "max_bytes", cfg.Retention.MaxBytes, "interval", interval)
	}

	if cfg.API.GRPCAddr != "" {
		grpcServer := ambergrpc.NewServer(batcher, log)
		go func() {
			log.Info("grpc server listening", "addr", cfg.API.GRPCAddr)
			if err := ambergrpc.ListenAndServe(grpcServer, cfg.API.GRPCAddr); err != nil {
				log.Error("grpc server error", "err", err)
			}
		}()
		go func() {
			<-ctx.Done()
			grpcServer.GracefulStop()
		}()
	}

	go func() {
		log.Info("pprof listening", "addr", "localhost:6060")
		pprofServer := &http.Server{
			Addr:              "localhost:6060",
			ReadHeaderTimeout: 5 * time.Second,
		}
		if err := pprofServer.ListenAndServe(); err != nil {
			log.Error("pprof server error", "err", err)
		}
	}()

	mux := http.NewServeMux()
	amberhttp.RegisterRoutes(mux, batcher, exec, logManager, logSparse, cfg.API.APIKey, log)

	httpServer := amberhttp.NewServer(cfg.API.HTTPAddr, mux, cfg.API.ReadTimeout, cfg.API.WriteTimeout, log)
	httpServer.Start()

	<-ctx.Done()
	log.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("http server shutdown error", "err", err)
	}

	batcher.Wait()

	if err := logSparse.Save(logDir); err != nil {
		log.Error("failed to save log sparse index", "err", err)
	}
	if err := spanSparse.Save(spanDir); err != nil {
		log.Error("failed to save span sparse index", "err", err)
	}

	log.Info("amber stopped")
}

func setupLogger(cfg config.LogConfig) *slog.Logger {
	level := slog.LevelInfo
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	if cfg.Format == "json" {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(h)
}
