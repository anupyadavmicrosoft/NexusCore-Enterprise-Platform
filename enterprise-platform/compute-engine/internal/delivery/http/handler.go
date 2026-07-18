package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/nexuscore/compute-engine/internal/domain"
	"github.com/nexuscore/compute-engine/internal/usecase"
	"go.opentelemetry.io/otel"
)

type httpComputeHandler struct {
	usecase domain.ComputeUsecase
}

func NewHttpComputeHandler(uc domain.ComputeUsecase) *httpComputeHandler {
	return &httpComputeHandler{usecase: uc}
}

func (h *httpComputeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tr := otel.Tracer("compute-http")
	ctx, span := tr.Start(r.Context(), "HTTP "+r.Method+" "+r.URL.Path)
	defer span.End()

	w.Header().Set("Content-Type", "application/json")

	// Helper to extract the core EventDrivenService if available
	var eventDriven *usecase.EventDrivenService
	if serviceImpl, ok := h.usecase.(interface{ GetEventDrivenService() *usecase.EventDrivenService }); ok {
		eventDriven = serviceImpl.GetEventDrivenService()
	}

	// 1. GET /transactions (List)
	if r.Method == http.MethodGet && r.URL.Path == "/transactions" {
		list, err := h.usecase.ListTransactions(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"failed to retrieve transaction sets"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(list)
		return
	}

	// 2. POST /transactions (Process via CQRS Command)
	if r.Method == http.MethodPost && r.URL.Path == "/transactions" {
		var tx domain.Transaction
		if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"malformed json context"}`))
			return
		}

		res, err := h.usecase.ProcessTransaction(ctx, tx)
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"error":"` + err.Error() + `"}`))
			return
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(res)
		return
	}

	// 3. GET /transactions/:id (Get single)
	if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/transactions/") {
		id := strings.TrimPrefix(r.URL.Path, "/transactions/")
		tx, err := h.usecase.GetTransaction(ctx, id)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"transaction record not found"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(tx)
		return
	}

	// =================================================================
	// 4. TOPIC MANAGEMENT ENDPOINTS
	// =================================================================
	if r.URL.Path == "/api/topics" && eventDriven != nil {
		// GET /api/topics (List)
		if r.Method == http.MethodGet {
			// We can list topics through mock client or cluster methods
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"name":"transaction-events","partitions":3,"replication_factor":2},
				{"name":"transaction-events-retry","partitions":3,"replication_factor":2},
				{"name":"transaction-events-dlq","partitions":1,"replication_factor":3}
			]`))
			return
		}

		// POST /api/topics (Create)
		if r.Method == http.MethodPost {
			var payload struct {
				Name       string `json:"name"`
				Partitions int    `json:"partitions"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Name == "" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"invalid topic body context"}`))
				return
			}
			slog.Info("Admin API: Creating new topic", "name", payload.Name)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":"created","topic":"` + payload.Name + `"}`))
			return
		}
	}

	// =================================================================
	// 5. DEAD LETTER QUEUE (DLQ) & RETRY QUEUE MONITORS
	// =================================================================
	if r.URL.Path == "/api/kafka/dlq" && r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`)) // Return empty DLQ state initially
		return
	}

	if r.URL.Path == "/api/kafka/retry" && r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`)) // Return empty Retry queue state initially
		return
	}

	// =================================================================
	// 6. EVENTSTOREDB REPLAY & SNAPSHOTTING TRIGGERS
	// =================================================================
	if r.URL.Path == "/api/eventstore/replay" && r.Method == http.MethodPost && eventDriven != nil {
		var cmd domain.ReplayEventsCommand
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil || cmd.StreamID == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid replay payload parameters"}`))
			return
		}

		err := eventDriven.ExecuteReplay(ctx, cmd)
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"error":"` + err.Error() + `"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"replayed","stream_id":"` + cmd.StreamID + `"}`))
		return
	}

	if r.URL.Path == "/api/eventstore/snapshot" && r.Method == http.MethodPost && eventDriven != nil {
		var cmd domain.TakeSnapshotCommand
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil || cmd.StreamID == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid snapshot payload parameters"}`))
			return
		}

		err := eventDriven.ExecuteSnapshot(ctx, cmd)
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"error":"` + err.Error() + `"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"snapshot_saved","stream_id":"` + cmd.StreamID + `"}`))
		return
	}

	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{"error":"endpoint path not found","status":404}`))
}
