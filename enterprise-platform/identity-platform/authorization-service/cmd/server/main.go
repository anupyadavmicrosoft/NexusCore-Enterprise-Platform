package main

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nexuscore/identity-platform/authorization-service/service"
	"github.com/nexuscore/identity-platform/shared-jwt-library"
)

type SimpleMetricsCollector struct {
	TotalEvaluations    int64
	SuccessfulAccesses  int64
	DeniedAccesses      int64
	TokenValidations    int64
	APIKeyValidations   int64
	OPAEvaluations      int64
}

var metrics = &SimpleMetricsCollector{}

type Server struct {
	rbacService *service.RBACService
	apiKeyService *service.APIKeyService
	policyEngine  *service.PolicyEngine
	jwtValidator  *service.JWTValidator
	privateKey    *rsa.PrivateKey // Cached for self-signed token generation in developer/mock scenarios
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	slog.Info("Initializing NexusCore Centralized Authorization Engine (authorization-service) on Sprint 3 Stack...")

	// 1. Initialize Cryptographic & Policy Core Components
	privKey, pubKey, err := jwt.GenerateRSAKeyPair()
	if err != nil {
		slog.Error("Failed to generate server-side cryptographic keypairs", "error", err)
		os.Exit(1)
	}

	rbacService := service.NewRBACService()
	apiKeyService := service.NewAPIKeyService()
	policyEngine := service.NewPolicyEngine()
	jwtValidator := service.NewJWTValidator(pubKey)

	srv := &Server{
		rbacService:   rbacService,
		apiKeyService: apiKeyService,
		policyEngine:  policyEngine,
		jwtValidator:  jwtValidator,
		privateKey:    privKey,
	}

	// 2. Define HTTP Router Configuration
	mux := http.NewServeMux()
	mux.HandleFunc("/authz/evaluate", srv.handleEvaluate)
	mux.HandleFunc("/authz/roles", srv.handleRoles)
	mux.HandleFunc("/authz/keys/generate", srv.handleGenerateAPIKey)
	mux.HandleFunc("/authz/keys/revoke", srv.handleRevokeAPIKey)
	mux.HandleFunc("/metrics", srv.handleMetrics)
	mux.HandleFunc("/healthz", srv.handleHealthz)

	// Admin utility endpoint to issue a dynamic self-signed JWT token for local testing/simulation purposes
	mux.HandleFunc("/authz/debug/issue-token", srv.handleDebugIssueToken)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000" // Binding to default port 3000 as per runtime environment requirements
	}

	server := &http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      corsMiddleware(loggingMiddleware(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// 3. Launch HTTP server
	go func() {
		slog.Info("Authorization REST/gRPC-simulation Gateway listening", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Authorization Service crashed unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown sequence
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	slog.Info("Shutting down authorization-service gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Graceful shutdown failed, forcing close", "error", err)
	}

	slog.Info("NexusCore Authorization Service shutdown complete.")
}

// REST Route Handlers

type AuthzEvaluationRequest struct {
	// Authentication Options (Provide one)
	JWT         string `json:"jwt,omitempty"`
	APIKey      string `json:"api_key,omitempty"`

	// Contextual identification (Fallbacks if auth is missing or parsed directly)
	Subject     string `json:"subject,omitempty"`
	TenantID    string `json:"tenant_id,omitempty"`
	Role        string `json:"role,omitempty"`
	Permissions []string `json:"permissions,omitempty"`

	// Target evaluation context
	Resource           string `json:"resource"`
	Action             string `json:"action"`
	RequiredPermission string `json:"required_permission,omitempty"`
	RequiredScope      string `json:"required_scope,omitempty"`

	// Dynamic Attributes for ABAC & OPA Evaluation
	ABACContext *service.ABACContext `json:"abac_context,omitempty"`
}

type AuthzEvaluationResponse struct {
	Allowed   bool   `json:"allowed"`
	Subject   string `json:"subject"`
	Reason    string `json:"reason"`
	EvalTime  string `json:"eval_time"`
	AuthType  string `json:"auth_type,omitempty"`
}

func (s *Server) handleEvaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AuthzEvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	metrics.TotalEvaluations++

	subject := req.Subject
	tenantID := req.TenantID
	role := req.Role
	permissions := req.Permissions
	authType := "CONTEXTUAL"

	// 1. Perform JWT validation if JWT token is passed in request
	if req.JWT != "" {
		metrics.TokenValidations++
		claims, err := s.jwtValidator.ValidateToken(req.JWT)
		if err != nil {
			metrics.DeniedAccesses++
			respondWithJSON(w, http.StatusOK, AuthzEvaluationResponse{
				Allowed:  false,
				Subject:  "",
				Reason:   fmt.Sprintf("Invalid JWT Token: %v", err),
				EvalTime: time.Now().Format(time.RFC3339),
				AuthType: "JWT",
			})
			return
		}
		subject = claims.Subject
		tenantID = claims.TenantID
		role = claims.Role
		permissions = claims.Permissions
		authType = "JWT"
	} else if req.APIKey != "" {
		// 2. Perform API Key validation if APIKey is provided
		metrics.APIKeyValidations++
		metadata, err := s.apiKeyService.ValidateAPIKey(r.Context(), req.APIKey, req.RequiredScope)
		if err != nil {
			metrics.DeniedAccesses++
			respondWithJSON(w, http.StatusOK, AuthzEvaluationResponse{
				Allowed:  false,
				Subject:  "",
				Reason:   fmt.Sprintf("Invalid or unpermitted API Key: %v", err),
				EvalTime: time.Now().Format(time.RFC3339),
				AuthType: "API_KEY",
			})
			return
		}
		subject = metadata.ID
		tenantID = metadata.TenantID
		role = "API_CLIENT"
		permissions = metadata.Scopes
		authType = "API_KEY"
	}

	// 3. Evaluate Hierarchical RBAC if requested and applicable
	if req.RequiredPermission != "" && role != "" {
		allowed, err := s.rbacService.EvaluateAccess(r.Context(), tenantID, role, req.RequiredPermission)
		if err != nil || !allowed {
			metrics.DeniedAccesses++
			respondWithJSON(w, http.StatusOK, AuthzEvaluationResponse{
				Allowed:  false,
				Subject:  subject,
				Reason:   "Access Denied: RBAC authorization policy check failed",
				EvalTime: time.Now().Format(time.RFC3339),
				AuthType: authType,
			})
			return
		}
	}

	// 4. Evaluate Dynamic ABAC & OPA Simulator Rules if dynamic attributes are provided
	if req.ABACContext != nil {
		metrics.OPAEvaluations++
		allowed, reason, err := s.policyEngine.EvaluateABACAndOPA(r.Context(), *req.ABACContext, req.Resource, req.Action)
		if err != nil || !allowed {
			metrics.DeniedAccesses++
			respondWithJSON(w, http.StatusOK, AuthzEvaluationResponse{
				Allowed:  false,
				Subject:  subject,
				Reason:   reason,
				EvalTime: time.Now().Format(time.RFC3339),
				AuthType: authType,
			})
			return
		}
	}

	metrics.SuccessfulAccesses++
	respondWithJSON(w, http.StatusOK, AuthzEvaluationResponse{
		Allowed:  true,
		Subject:  subject,
		Reason:   "Access Granted: Policy evaluation checks satisfied",
		EvalTime: time.Now().Format(time.RFC3339),
		AuthType: authType,
	})
}

func (s *Server) handleRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req service.Role
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	if req.Name == "" {
		respondWithError(w, http.StatusBadRequest, "role Name is required")
		return
	}

	err := s.rbacService.CreateRole(r.Context(), &req)
	if err != nil {
		if err == service.ErrDuplicateRole {
			respondWithError(w, http.StatusConflict, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"status":  "CREATED",
		"message": "Role created successfully",
		"role_id": req.ID,
	})
}

func (s *Server) handleGenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TenantID  string        `json:"tenant_id"`
		OrgID     string        `json:"org_id"`
		Name      string        `json:"name"`
		Scopes    []string      `json:"scopes"`
		ExpiresIn time.Duration `json:"expires_in"` // in nanoseconds
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	if req.TenantID == "" || req.Name == "" {
		respondWithError(w, http.StatusBadRequest, "TenantID and Name are required variables")
		return
	}

	if req.ExpiresIn == 0 {
		req.ExpiresIn = 365 * 24 * time.Hour // Default 1 year validity
	}

	rawKey, err := s.apiKeyService.GenerateAPIKey(r.Context(), req.TenantID, req.OrgID, req.Name, req.Scopes, req.ExpiresIn)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]string{
		"status":  "CREATED",
		"api_key": rawKey,
		"message": "Store this key securely. This raw secret will not be shown again.",
	})
}

func (s *Server) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		KeyID string `json:"key_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	err := s.apiKeyService.RevokeAPIKey(r.Context(), req.KeyID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"status":  "REVOKED",
		"message": "API key successfully deactivated.",
	})
}

func (s *Server) handleDebugIssueToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req jwt.Claims
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	if req.Subject == "" {
		req.Subject = "usr-debug-0001"
	}
	if req.TenantID == "" {
		req.TenantID = "ten-8888-0001"
	}
	if req.Expiry == 0 {
		req.Expiry = time.Now().Add(1 * time.Hour).Unix()
	}

	token, err := jwt.SignTokenRS256(req, s.privateKey, "nexuscore-authz-debug-key")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"token": token,
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// Format metrics in standard OpenTelemetry & Prometheus format
	fmt.Fprintf(w, "# HELP nexuscore_authz_evaluations_total Total authorization evaluate cycles executed\n")
	fmt.Fprintf(w, "# TYPE nexuscore_authz_evaluations_total counter\n")
	fmt.Fprintf(w, "nexuscore_authz_evaluations_total %d\n", metrics.TotalEvaluations)

	fmt.Fprintf(w, "# HELP nexuscore_authz_access_granted_total Total operations allowed\n")
	fmt.Fprintf(w, "# TYPE nexuscore_authz_access_granted_total counter\n")
	fmt.Fprintf(w, "nexuscore_authz_access_granted_total %d\n", metrics.SuccessfulAccesses)

	fmt.Fprintf(w, "# HELP nexuscore_authz_access_denied_total Total operations blocked\n")
	fmt.Fprintf(w, "# TYPE nexuscore_authz_access_denied_total counter\n")
	fmt.Fprintf(w, "nexuscore_authz_access_denied_total %d\n", metrics.DeniedAccesses)

	fmt.Fprintf(w, "# HELP nexuscore_authz_token_validations_total Total cryptographically validated JWTs\n")
	fmt.Fprintf(w, "# TYPE nexuscore_authz_token_validations_total counter\n")
	fmt.Fprintf(w, "nexuscore_authz_token_validations_total %d\n", metrics.TokenValidations)

	fmt.Fprintf(w, "# HELP nexuscore_authz_apikey_validations_total Total validated API Keys\n")
	fmt.Fprintf(w, "# TYPE nexuscore_authz_apikey_validations_total counter\n")
	fmt.Fprintf(w, "nexuscore_authz_apikey_validations_total %d\n", metrics.APIKeyValidations)

	fmt.Fprintf(w, "# HELP nexuscore_authz_opa_evaluations_total Total OPA declarative rules executed\n")
	fmt.Fprintf(w, "# TYPE nexuscore_authz_opa_evaluations_total counter\n")
	fmt.Fprintf(w, "nexuscore_authz_opa_evaluations_total %d\n", metrics.OPAEvaluations)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "HEALTHY",
		"service":   "authorization-service",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// HTTP Helper Decorators

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("Processed Gateway Ingress Request",
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
