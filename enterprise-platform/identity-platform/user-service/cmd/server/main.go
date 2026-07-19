package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nexuscore/identity-platform/user-service/db"
	"github.com/nexuscore/identity-platform/user-service/service"
)

type SimpleMetricsCollector struct {
	UserCreations    int64
	ProfileUpdates   int64
	AvatarUploads    int64
	EmailUpdates     int64
	PhoneUpdates     int64
	StateTransitions int64
	SoftDeletes      int64
	HardDeletes      int64
	SearchQueries    int64
	AuditLogQueries  int64
}

var metrics = &SimpleMetricsCollector{}

type Server struct {
	userService *service.UserService
	store       db.UserStore
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	slog.Info("Starting NexusCore User Profiles & Lifecycles Service (user-service) on Sprint 3 Stack...")

	userStore := db.NewMockUserDB()
	userService := service.NewUserService(userStore)

	srv := &Server{
		userService: userService,
		store:       userStore,
	}

	mux := http.NewServeMux()
	// Legacy / Backwards Compatibility routes
	mux.HandleFunc("/users/create", srv.handleLegacyCreateUser)
	mux.HandleFunc("/users/lifecycle", srv.handleLegacyLifecycle)

	// Modern REST/Enterprise API Routes
	mux.HandleFunc("/users/profile", srv.handleGetOrUpdateProfile) // GET (Retrieve) and PUT/POST (Update)
	mux.HandleFunc("/users/search", srv.handleSearchUsers)
	mux.HandleFunc("/users/avatar/upload", srv.handleAvatarUpload)
	mux.HandleFunc("/users/email/update", srv.handleEmailUpdate)
	mux.HandleFunc("/users/phone/update", srv.handlePhoneUpdate)
	mux.HandleFunc("/users/deactivate", srv.handleDeactivate)
	mux.HandleFunc("/users/reactivate", srv.handleReactivate)
	mux.HandleFunc("/users/soft-delete", srv.handleSoftDelete)
	mux.HandleFunc("/users/hard-delete", srv.handleHardDelete)
	mux.HandleFunc("/users/audit-logs", srv.handleAuditLogs)
	mux.HandleFunc("/metrics", srv.handleMetrics)
	mux.HandleFunc("/healthz", srv.handleHealthz)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000" // Bound to 3000 to comply with runtime environment constraints
	}

	httpServer := &http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      corsMiddleware(loggingMiddleware(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		slog.Info("User Gateway Rest interface listening", "port", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("User API gateway crash", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	slog.Info("Shutting down user-service gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("Graceful shutdown failed, forcing close", "error", err)
	}

	slog.Info("NexusCore User Service shutdown complete.")
}

// REST Route Handlers

func (s *Server) handleLegacyCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TenantID  string `json:"tenant_id"`
		Email     string `json:"email"`
		Password  string `json:"password"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Phone     string `json:"phone"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	actorID := s.getActorID(r)
	ip, ua := s.getNetworkMetadata(r)

	user, err := s.userService.CreateUser(r.Context(), req.Email, req.Phone, req.Password, req.FirstName, req.LastName, "Unassigned", "Member", req.TenantID, actorID, ip, ua)
	if err != nil {
		if err == service.ErrInvalidPassword || err == service.ErrInvalidEmail {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err == db.ErrUserAlreadyExists {
			respondWithError(w, http.StatusConflict, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	metrics.UserCreations++
	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"user_id":    user.ID,
		"email":      user.Email,
		"status":     user.Status,
		"created_at": user.CreatedAt.Format(time.RFC3339),
	})
}

func (s *Server) handleLegacyLifecycle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID string `json:"user_id"`
		Action string `json:"action"` // "SUSPEND", "ACTIVATE", "ARCHIVE", "LOCK", "UNLOCK"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	actorID := s.getActorID(r)
	ip, ua := s.getNetworkMetadata(r)

	var u *db.User
	var err error

	switch req.Action {
	case "SUSPEND", "LOCK":
		u, err = s.userService.DeactivateUser(r.Context(), req.UserID, actorID, ip, ua)
	case "ACTIVATE", "UNLOCK":
		u, err = s.userService.ReactivateUser(r.Context(), req.UserID, actorID, ip, ua)
	case "ARCHIVE":
		err = s.userService.SoftDeleteUser(r.Context(), req.UserID, actorID, ip, ua)
		if err == nil {
			metrics.SoftDeletes++
			respondWithJSON(w, http.StatusOK, map[string]interface{}{
				"user_id":      req.UserID,
				"state":        "ARCHIVED",
				"completed_at": time.Now().Format(time.RFC3339),
			})
			return
		}
	default:
		respondWithError(w, http.StatusBadRequest, "invalid state machine action requested")
		return
	}

	if err != nil {
		if err == db.ErrUserNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	metrics.StateTransitions++
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":      u.ID,
		"state":        u.Status,
		"completed_at": u.UpdatedAt.Format(time.RFC3339),
	})
}

func (s *Server) handleGetOrUpdateProfile(w http.ResponseWriter, r *http.Request) {
	actorID := s.getActorID(r)
	ip, ua := s.getNetworkMetadata(r)

	switch r.Method {
	case http.MethodGet:
		userID := r.URL.Query().Get("id")
		if userID == "" {
			respondWithError(w, http.StatusBadRequest, "Query parameter 'id' is required")
			return
		}

		user, err := s.userService.GetUser(r.Context(), userID, actorID, ip, ua)
		if err != nil {
			if err == db.ErrUserNotFound {
				respondWithError(w, http.StatusNotFound, err.Error())
				return
			}
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondWithJSON(w, http.StatusOK, user)

	case http.MethodPost, http.MethodPut:
		var req struct {
			ID         string `json:"id"`
			FirstName  string `json:"first_name"`
			LastName   string `json:"last_name"`
			Department string `json:"department"`
			Title      string `json:"title"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "malformed request payload")
			return
		}

		if req.ID == "" {
			respondWithError(w, http.StatusBadRequest, "Field 'id' is required")
			return
		}

		user, err := s.userService.UpdateProfile(r.Context(), req.ID, req.FirstName, req.LastName, req.Department, req.Title, actorID, ip, ua)
		if err != nil {
			if err == db.ErrUserNotFound {
				respondWithError(w, http.StatusNotFound, err.Error())
				return
			}
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		metrics.ProfileUpdates++
		respondWithJSON(w, http.StatusOK, user)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSearchUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query    string `json:"query"`
		Status   string `json:"status"`
		TenantID string `json:"tenant_id"`
		Limit    int    `json:"limit"`
		Offset   int    `json:"offset"`
	}

	// Read fields from request body if POST, or query params if GET
	if r.Method == http.MethodPost {
		_ = json.NewDecoder(r.Body).Decode(&req)
	} else {
		req.Query = r.URL.Query().Get("query")
		req.Status = r.URL.Query().Get("status")
		req.TenantID = r.URL.Query().Get("tenant_id")
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			req.Limit, _ = strconv.Atoi(lStr)
		}
		if oStr := r.URL.Query().Get("offset"); oStr != "" {
			req.Offset, _ = strconv.Atoi(oStr)
		}
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	actorID := s.getActorID(r)
	ip, ua := s.getNetworkMetadata(r)

	users, total, err := s.userService.SearchUsers(r.Context(), req.Query, req.Status, req.TenantID, req.Limit, req.Offset, actorID, ip, ua)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	metrics.SearchQueries++
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"users":  users,
		"total":  total,
		"limit":  req.Limit,
		"offset": req.Offset,
	})
}

func (s *Server) handleAvatarUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID       string `json:"user_id"`
		AvatarBase64 string `json:"avatar_base64"`
		Filename     string `json:"filename"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	if req.UserID == "" {
		respondWithError(w, http.StatusBadRequest, "UserID variable is required")
		return
	}

	if req.Filename == "" {
		req.Filename = "avatar.png"
	}

	actorID := s.getActorID(r)
	ip, ua := s.getNetworkMetadata(r)

	user, err := s.userService.UpdateAvatar(r.Context(), req.UserID, req.AvatarBase64, req.Filename, actorID, ip, ua)
	if err != nil {
		if err == db.ErrUserNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	metrics.AvatarUploads++
	respondWithJSON(w, http.StatusOK, map[string]string{
		"status":     "UPLOADED",
		"avatar_url": user.AvatarURL,
		"message":    "Profile avatar modified successfully.",
	})
}

func (s *Server) handleEmailUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID   string `json:"user_id"`
		NewEmail string `json:"new_email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	if req.UserID == "" || req.NewEmail == "" {
		respondWithError(w, http.StatusBadRequest, "UserID and NewEmail are required variables")
		return
	}

	actorID := s.getActorID(r)
	ip, ua := s.getNetworkMetadata(r)

	user, err := s.userService.UpdateEmail(r.Context(), req.UserID, req.NewEmail, actorID, ip, ua)
	if err != nil {
		if err == db.ErrUserNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		if err == service.ErrInvalidEmail {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	metrics.EmailUpdates++
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "EMAIL_UPDATED",
		"email":          user.Email,
		"email_verified": user.EmailVerified,
		"message":        "User email address has been modified. A fresh verification OTP is required.",
	})
}

func (s *Server) handlePhoneUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID   string `json:"user_id"`
		NewPhone string `json:"new_phone"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	if req.UserID == "" || req.NewPhone == "" {
		respondWithError(w, http.StatusBadRequest, "UserID and NewPhone are required variables")
		return
	}

	actorID := s.getActorID(r)
	ip, ua := s.getNetworkMetadata(r)

	user, err := s.userService.UpdatePhone(r.Context(), req.UserID, req.NewPhone, actorID, ip, ua)
	if err != nil {
		if err == db.ErrUserNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	metrics.PhoneUpdates++
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "PHONE_UPDATED",
		"phone":          user.Phone,
		"phone_verified": user.PhoneVerified,
		"message":        "User telephone number has been modified. A fresh SMS OTP challenge is required.",
	})
}

func (s *Server) handleDeactivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID string `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	actorID := s.getActorID(r)
	ip, ua := s.getNetworkMetadata(r)

	user, err := s.userService.DeactivateUser(r.Context(), req.UserID, actorID, ip, ua)
	if err != nil {
		if err == db.ErrUserNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	metrics.StateTransitions++
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "DEACTIVATED",
		"user_id": user.ID,
		"message": "User account deactivation completed successfully.",
	})
}

func (s *Server) handleReactivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID string `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	actorID := s.getActorID(r)
	ip, ua := s.getNetworkMetadata(r)

	user, err := s.userService.ReactivateUser(r.Context(), req.UserID, actorID, ip, ua)
	if err != nil {
		if err == db.ErrUserNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	metrics.StateTransitions++
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ACTIVE",
		"user_id": user.ID,
		"message": "User account reactivation completed successfully.",
	})
}

func (s *Server) handleSoftDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID string `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	actorID := s.getActorID(r)
	ip, ua := s.getNetworkMetadata(r)

	err := s.userService.SoftDeleteUser(r.Context(), req.UserID, actorID, ip, ua)
	if err != nil {
		if err == db.ErrUserNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	metrics.SoftDeletes++
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "SOFT_DELETED",
		"user_id": req.UserID,
		"message": "User soft deleted and GDPR data scrubbing/shredding accomplished.",
	})
}

func (s *Server) handleHardDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID string `json:"user_id"`
	}

	if r.Method == http.MethodDelete {
		req.UserID = r.URL.Query().Get("user_id")
	} else {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	if req.UserID == "" {
		respondWithError(w, http.StatusBadRequest, "UserID parameter is required")
		return
	}

	actorID := s.getActorID(r)
	ip, ua := s.getNetworkMetadata(r)

	err := s.userService.HardDeleteUser(r.Context(), req.UserID, actorID, ip, ua)
	if err != nil {
		if err == db.ErrUserNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	metrics.HardDeletes++
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "HARD_DELETED",
		"user_id": req.UserID,
		"message": "User account record hard-deleted permanently from central directories.",
	})
}

func (s *Server) handleAuditLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	targetUserID := r.URL.Query().Get("user_id")
	limit := 50
	offset := 0

	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		limit, _ = strconv.Atoi(lStr)
	}
	if oStr := r.URL.Query().Get("offset"); oStr != "" {
		offset, _ = strconv.Atoi(oStr)
	}

	logs, err := s.userService.GetAuditLogs(r.Context(), targetUserID, limit, offset)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	metrics.AuditLogQueries++
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"audit_logs": logs,
		"count":      len(logs),
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// Format metrics in standard OpenTelemetry & Prometheus format
	fmt.Fprintf(w, "# HELP nexuscore_user_creations_total Total users created via API\n")
	fmt.Fprintf(w, "# TYPE nexuscore_user_creations_total counter\n")
	fmt.Fprintf(w, "nexuscore_user_creations_total %d\n", metrics.UserCreations)

	fmt.Fprintf(w, "# HELP nexuscore_user_profile_updates_total Total profile metadata updates processed\n")
	fmt.Fprintf(w, "# TYPE nexuscore_user_profile_updates_total counter\n")
	fmt.Fprintf(w, "nexuscore_user_profile_updates_total %d\n", metrics.ProfileUpdates)

	fmt.Fprintf(w, "# HELP nexuscore_user_avatar_uploads_total Total profile avatar uploads completed\n")
	fmt.Fprintf(w, "# TYPE nexuscore_user_avatar_uploads_total counter\n")
	fmt.Fprintf(w, "nexuscore_user_avatar_uploads_total %d\n", metrics.AvatarUploads)

	fmt.Fprintf(w, "# HELP nexuscore_user_email_updates_total Total email modification operations\n")
	fmt.Fprintf(w, "# TYPE nexuscore_user_email_updates_total counter\n")
	fmt.Fprintf(w, "nexuscore_user_email_updates_total %d\n", metrics.EmailUpdates)

	fmt.Fprintf(w, "# HELP nexuscore_user_phone_updates_total Total phone modification operations\n")
	fmt.Fprintf(w, "# TYPE nexuscore_user_phone_updates_total counter\n")
	fmt.Fprintf(w, "nexuscore_user_phone_updates_total %d\n", metrics.PhoneUpdates)

	fmt.Fprintf(w, "# HELP nexuscore_user_state_transitions_total Total manual lock/suspension transitions\n")
	fmt.Fprintf(w, "# TYPE nexuscore_user_state_transitions_total counter\n")
	fmt.Fprintf(w, "nexuscore_user_state_transitions_total %d\n", metrics.StateTransitions)

	fmt.Fprintf(w, "# HELP nexuscore_user_soft_deletes_total Total GDPR-compliant soft deletes executed\n")
	fmt.Fprintf(w, "# TYPE nexuscore_user_soft_deletes_total counter\n")
	fmt.Fprintf(w, "nexuscore_user_soft_deletes_total %d\n", metrics.SoftDeletes)

	fmt.Fprintf(w, "# HELP nexuscore_user_hard_deletes_total Total database record hard deletes executed\n")
	fmt.Fprintf(w, "# TYPE nexuscore_user_hard_deletes_total counter\n")
	fmt.Fprintf(w, "nexuscore_user_hard_deletes_total %d\n", metrics.HardDeletes)

	fmt.Fprintf(w, "# HELP nexuscore_user_search_queries_total Total search and query requests handled\n")
	fmt.Fprintf(w, "# TYPE nexuscore_user_search_queries_total counter\n")
	fmt.Fprintf(w, "nexuscore_user_search_queries_total %d\n", metrics.SearchQueries)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "HEALTHY",
		"service":   "user-service",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// Helpers for request parsing

func (s *Server) getActorID(r *http.Request) string {
	// 1. Try reading standard header
	if actor := r.Header.Get("X-Actor-ID"); actor != "" {
		return actor
	}
	// 2. Fallback to authorization header simulation
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		// Mock token subject parse or return bearer prefix value for testing
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 {
			return "usr-parsed-" + parts[1][:5]
		}
	}
	// 3. Fallback standard default system user
	return "usr-system-admin"
}

func (s *Server) getNetworkMetadata(r *http.Request) (string, string) {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip == "" {
		ip = "127.0.0.1"
	}
	ua := r.UserAgent()
	if ua == "" {
		ua = "Mozilla/5.0 (NexusCore REST Client)"
	}
	return ip, ua
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Actor-ID")
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
