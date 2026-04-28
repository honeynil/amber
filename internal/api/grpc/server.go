package grpc

import (
	"log/slog"
	"net"

	collectorlogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	collectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"

	"github.com/hnlbs/amber/internal/ingest"
)

func NewServer(batcher *ingest.Batcher, log *slog.Logger) *grpc.Server {
	s := grpc.NewServer()
	collectorlogs.RegisterLogsServiceServer(s, &logsServer{batcher: batcher, log: log})
	collectortrace.RegisterTraceServiceServer(s, &tracesServer{batcher: batcher, log: log})
	return s
}

func ListenAndServe(s *grpc.Server, addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.Serve(lis)
}
