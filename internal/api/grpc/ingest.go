package grpc

import (
	"context"
	"log/slog"

	collectorlogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	collectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"

	"github.com/hnlbs/amber/internal/ingest"
)

type logsServer struct {
	collectorlogs.UnimplementedLogsServiceServer
	batcher *ingest.Batcher
	log     *slog.Logger
}

func (s *logsServer) Export(ctx context.Context, req *collectorlogs.ExportLogsServiceRequest) (*collectorlogs.ExportLogsServiceResponse, error) {
	for _, rl := range req.ResourceLogs {
		service, host := ingest.ExtractResource(rl.Resource.GetAttributes())
		for _, sl := range rl.ScopeLogs {
			for _, lr := range sl.LogRecords {
				entry, err := ingest.OTLPLogToEntry(lr, service, host)
				if err != nil {
					s.log.Debug("grpc: skip log record", "err", err)
					continue
				}
				if err := s.batcher.SendLog(entry); err != nil {
					s.log.Debug("grpc: send log failed", "err", err)
				}
			}
		}
	}
	return &collectorlogs.ExportLogsServiceResponse{}, nil
}

type tracesServer struct {
	collectortrace.UnimplementedTraceServiceServer
	batcher *ingest.Batcher
	log     *slog.Logger
}

func (s *tracesServer) Export(ctx context.Context, req *collectortrace.ExportTraceServiceRequest) (*collectortrace.ExportTraceServiceResponse, error) {
	for _, rs := range req.ResourceSpans {
		service, _ := ingest.ExtractResource(rs.Resource.GetAttributes())
		for _, ss := range rs.ScopeSpans {
			for _, sp := range ss.Spans {
				entry, err := ingest.OTLPSpanToEntry(sp, service)
				if err != nil {
					s.log.Debug("grpc: skip span", "err", err)
					continue
				}
				if err := s.batcher.SendSpan(entry); err != nil {
					s.log.Debug("grpc: send span failed", "err", err)
				}
			}
		}
	}
	return &collectortrace.ExportTraceServiceResponse{}, nil
}
