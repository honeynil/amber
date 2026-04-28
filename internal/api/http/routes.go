package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/hnlbs/amber/internal/index"
	"github.com/hnlbs/amber/internal/ingest"
	"github.com/hnlbs/amber/internal/query"
	"github.com/hnlbs/amber/internal/storage"
	"github.com/hnlbs/amber/internal/ui"
)

func RegisterRoutes(
	mux *http.ServeMux,
	batcher *ingest.Batcher,
	exec *query.Executor,
	logManager *storage.SegmentManager,
	logSparse *index.SparseIndex,
	apiKey string,
	log *slog.Logger,
) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	auth := func(h http.Handler) http.Handler {
		return APIKeyMiddleware(apiKey, h)
	}

	mux.Handle("POST /api/v1/logs", auth(NewIngestHandler(batcher, log)))
	mux.Handle("GET /api/v1/logs", auth(NewQueryHandler(exec, log)))
	mux.Handle("GET /api/v1/traces/", auth(NewTraceHandler(exec, log)))
	mux.Handle("GET /api/v1/traces", auth(NewTracesHandler(exec, log)))
	mux.Handle("GET /api/v1/services", auth(NewServicesHandler(exec, log)))

	otlpH := NewOTLPHandler(batcher, log)
	mux.Handle("POST /v1/logs", auth(otlpH))
	mux.Handle("POST /v1/traces", auth(otlpH))

	adminH := NewAdminHandler(logManager, logSparse, log)
	mux.Handle("GET /api/v1/admin/stats", auth(http.HandlerFunc(adminH.Stats)))
	mux.Handle("GET /api/v1/admin/segments", auth(http.HandlerFunc(adminH.Segments)))

	mux.Handle("/", ui.Handler())
}
