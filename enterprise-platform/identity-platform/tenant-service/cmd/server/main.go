package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nexuscore/identity-platform/tenant-service/db"
	"github.com/nexuscore/identity-platform/tenant-service/service"
)

type Server struct {
	tenantService *service.TenantService
	store         db.TenantStore
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	slog.Info("Initializing NexusCore SaaS Multi-Tenant Manager (tenant-service) with production stack...")

	tenantStore := db.NewMockTenantDB()
	tenantService := service.NewTenantService(tenantStore)

	srv := &Server{
		tenantService: tenantService,
		store:         tenantStore,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/tenants/provision", srv.handleProvisionTenant)
	mux.HandleFunc("/tenants/shred", srv.handleShredTenant)
	mux.HandleFunc("/tenants/suspend", srv.handleSuspendTenant)
	mux.HandleFunc("/tenants/reactivate", srv.handleReactivateTenant)
	mux.HandleFunc("/tenants/list", srv.handleListTenants)
	mux.HandleFunc("/tenants/get", srv.handleGetTenant)
	mux.HandleFunc("/healthz", handleHealthz)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	server := &http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      corsMiddleware(loggingMiddleware(mux)),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("SaaS Tenant Service listening", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Tenant Server crashed unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	slog.Info("Shutting down tenant-service gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Graceful shutdown failed, forcing close", "error", err)
	}

	slog.Info("NexusCore Tenant Service shutdown complete.")
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("Inbound SaaS Tenant request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start).String(),
		)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Actor-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "HEALTHY", "service": "tenant-service"})
}

type ProvisionTenantRequest struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

func (s *Server) handleProvisionTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ProvisionTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed provision request payload")
		return
	}

	tenant, err := s.tenantService.ProvisionTenant(r.Context(), req.Name, req.Domain)
	if err != nil {
		if err == service.ErrInvalidTenantName || err == service.ErrInvalidDomain {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err == db.ErrTenantAlreadyExists {
			respondWithError(w, http.StatusConflict, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("Successfully provisioned tenant dedicated boundary schemas", "name", tenant.Name, "tenant_id", tenant.ID)

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"tenant_id":        tenant.ID,
		"name":             tenant.Name,
		"domain":           tenant.Domain,
		"database_schema":  tenant.DatabaseSchema,
		"status":           tenant.Status,
		"provisioned_at":   tenant.CreatedAt.Format(time.RFC3339),
	})
}

type ShredTenantRequest struct {
	TenantID string `json:"tenant_id"`
}

func (s *Server) handleShredTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ShredTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed shred request payload")
		return
	}

	tenant, err := s.tenantService.CryptoShredTenant(r.Context(), req.TenantID)
	if err != nil {
		if err == db.ErrTenantNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("CRYPTO-SHREDDING ENCRYPTION KEYS FOR TENANT OFFBOARDING COMPLETE", "tenant_id", req.TenantID)

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"tenant_id":        tenant.ID,
		"action":           "CRYPTO_SHRED_SUCCESS",
		"status":           "TERMINATED",
		"completed_at":     time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleSuspendTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TenantID string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed suspend request payload")
		return
	}

	tenant, err := s.tenantService.SuspendTenant(r.Context(), req.TenantID)
	if err != nil {
		if err == db.ErrTenantNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, tenant)
}

func (s *Server) handleReactivateTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TenantID string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed reactivate request payload")
		return
	}

	tenant, err := s.tenantService.ReactivateTenant(r.Context(), req.TenantID)
	if err != nil {
		if err == db.ErrTenantNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, tenant)
}

func (s *Server) handleListTenants(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 10
	offset := 0

	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		limit, _ = strconv.Atoi(lStr)
	}
	if oStr := r.URL.Query().Get("offset"); oStr != "" {
		offset, _ = strconv.Atoi(oStr)
	}

	tenants, total, err := s.tenantService.ListTenants(r.Context(), limit, offset)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"tenants": tenants,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

func (s *Server) handleGetTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	domain := r.URL.Query().Get("domain")

	var tenant *db.Tenant
	var err error

	if id != "" {
		tenant, err = s.tenantService.GetTenant(r.Context(), id)
	} else if domain != "" {
		tenant, err = s.tenantService.GetTenantByDomain(r.Context(), domain)
	} else {
		respondWithError(w, http.StatusBadRequest, "either 'id' or 'domain' query parameter is required")
		return
	}

	if err != nil {
		if err == db.ErrTenantNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, tenant)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
