package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nexuscore/identity-platform/organization-service/db"
	"github.com/nexuscore/identity-platform/organization-service/service"
)

type Server struct {
	orgService *service.OrgService
	store      db.OrgStore
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	slog.Info("Initializing NexusCore Hierarchical Organization Service (organization-service) on Sprint 3 Stack...")

	orgStore := db.NewMockOrgDB()
	orgService := service.NewOrgService(orgStore)

	srv := &Server{
		orgService: orgService,
		store:      orgStore,
	}

	mux := http.NewServeMux()

	// Organization Nodes (Hierarchy & Department)
	mux.HandleFunc("/orgs/create", srv.handleCreateOrg)
	mux.HandleFunc("/orgs/tree", srv.handleGetOrgTree)
	mux.HandleFunc("/orgs/move", srv.handleMoveOrgNode)

	// Workspaces
	mux.HandleFunc("/workspaces/create", srv.handleCreateWorkspace)
	mux.HandleFunc("/workspaces/list", srv.handleListWorkspaces)
	mux.HandleFunc("/workspaces/update", srv.handleUpdateWorkspace)
	mux.HandleFunc("/workspaces/delete", srv.handleDeleteWorkspace)

	// Teams
	mux.HandleFunc("/teams/create", srv.handleCreateTeam)
	mux.HandleFunc("/teams/list", srv.handleListTeams)
	mux.HandleFunc("/teams/members/add", srv.handleAddTeamMember)
	mux.HandleFunc("/teams/members/remove", srv.handleRemoveTeamMember)
	mux.HandleFunc("/teams/members/list", srv.handleListTeamMembers)

	// Invitations
	mux.HandleFunc("/invitations/create", srv.handleInviteUser)
	mux.HandleFunc("/invitations/accept", srv.handleAcceptInvitation)
	mux.HandleFunc("/invitations/revoke", srv.handleRevokeInvitation)
	mux.HandleFunc("/invitations/list", srv.handleListInvitations)

	// Ownership Transfer
	mux.HandleFunc("/ownership/transfer/initiate", srv.handleInitiateTransfer)
	mux.HandleFunc("/ownership/transfer/complete", srv.handleCompleteTransfer)

	mux.HandleFunc("/healthz", handleHealthz)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	server := &http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      corsMiddleware(loggingMiddleware(mux)),
		ReadTimeout:  12 * time.Second,
		WriteTimeout: 12 * time.Second,
	}

	go func() {
		slog.Info("Organization Service listening", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Organization Server crashed unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	slog.Info("Shutting down organization-service gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Graceful shutdown failed, forcing close", "error", err)
	}

	slog.Info("NexusCore Organization Service shutdown complete.")
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("Inbound Organization Hierarchical request",
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Actor-ID, X-Tenant-ID")
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
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "HEALTHY", "service": "organization-service"})
}

// ---------------- HANDLERS ----------------

func (s *Server) getTenantID(r *http.Request) string {
	if tID := r.Header.Get("X-Tenant-ID"); tID != "" {
		return tID
	}
	if tID := r.URL.Query().Get("tenant_id"); tID != "" {
		return tID
	}
	return "ten-8888-0001" // Standard fallback tenant for development testing
}

type CreateOrgRequest struct {
	ParentID string `json:"parent_id"`
	Name     string `json:"name"`
	Type     string `json:"type"` // "ORGANIZATION", "DEPARTMENT"
	OwnerID  string `json:"owner_id"`
}

func (s *Server) handleCreateOrg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	tenantID := s.getTenantID(r)
	if req.Type == "" {
		req.Type = "DEPARTMENT"
	}
	if req.OwnerID == "" {
		req.OwnerID = "usr-9999-0001"
	}

	node, err := s.orgService.CreateNode(r.Context(), tenantID, req.ParentID, req.Name, req.Type, req.OwnerID)
	if err != nil {
		if err == service.ErrInvalidNodeName {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("Hierarchical organization node created successfully", "tenant", tenantID, "path", node.Path, "name", node.Name)

	respondWithJSON(w, http.StatusCreated, node)
}

func (s *Server) handleGetOrgTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := s.getTenantID(r)
	tree, err := s.orgService.GetNodeTree(r.Context(), tenantID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	slog.Info("Traversed organization node ltree hierarchy", "tenant", tenantID)

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"tenant_id": tenantID,
		"root":      tree,
	})
}

func (s *Server) handleMoveOrgNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID       string `json:"id"`
		ParentID string `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	tenantID := s.getTenantID(r)
	node, err := s.orgService.UpdateNodeParent(r.Context(), tenantID, req.ID, req.ParentID)
	if err != nil {
		if err == service.ErrCycleDetected {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err == db.ErrNodeNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, node)
}

// ---------------- WORKSPACE HANDLERS ----------------

func (s *Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrgID string `json:"org_id"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	tenantID := s.getTenantID(r)
	ws, err := s.orgService.CreateWorkspace(r.Context(), tenantID, req.OrgID, req.Name)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, ws)
}

func (s *Server) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := s.getTenantID(r)
	orgID := r.URL.Query().Get("org_id")

	workspaces, err := s.orgService.ListWorkspaces(r.Context(), tenantID, orgID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, workspaces)
}

func (s *Server) handleUpdateWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	tenantID := s.getTenantID(r)
	ws, err := s.orgService.UpdateWorkspace(r.Context(), tenantID, req.ID, req.Name, req.Status)
	if err != nil {
		if err == db.ErrWorkspaceNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, ws)
}

func (s *Server) handleDeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if r.Method == http.MethodDelete {
		req.ID = r.URL.Query().Get("id")
	} else {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	if req.ID == "" {
		respondWithError(w, http.StatusBadRequest, "id is required")
		return
	}

	tenantID := s.getTenantID(r)
	err := s.orgService.DeleteWorkspace(r.Context(), tenantID, req.ID)
	if err != nil {
		if err == db.ErrWorkspaceNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "DELETED", "id": req.ID})
}

// ---------------- TEAM HANDLERS ----------------

func (s *Server) handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WorkspaceID string `json:"workspace_id"`
		OrgID       string `json:"org_id"`
		Name        string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	tenantID := s.getTenantID(r)
	team, err := s.orgService.CreateTeam(r.Context(), tenantID, req.WorkspaceID, req.OrgID, req.Name)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, team)
}

func (s *Server) handleListTeams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := s.getTenantID(r)
	workspaceID := r.URL.Query().Get("workspace_id")

	teams, err := s.orgService.ListTeams(r.Context(), tenantID, workspaceID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, teams)
}

func (s *Server) handleAddTeamMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TeamID string `json:"team_id"`
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	tenantID := s.getTenantID(r)
	if req.Role == "" {
		req.Role = "MEMBER"
	}

	member, err := s.orgService.AddTeamMember(r.Context(), tenantID, req.TeamID, req.UserID, req.Role)
	if err != nil {
		if err == db.ErrMemberAlreadyExists {
			respondWithError(w, http.StatusConflict, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, member)
}

func (s *Server) handleRemoveTeamMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TeamID string `json:"team_id"`
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	tenantID := s.getTenantID(r)
	err := s.orgService.RemoveTeamMember(r.Context(), tenantID, req.TeamID, req.UserID)
	if err != nil {
		if err == db.ErrMemberNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "REMOVED", "user_id": req.UserID})
}

func (s *Server) handleListTeamMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := s.getTenantID(r)
	teamID := r.URL.Query().Get("team_id")

	members, err := s.orgService.GetTeamMembers(r.Context(), tenantID, teamID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, members)
}

// ---------------- INVITATION HANDLERS ----------------

func (s *Server) handleInviteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrgID       string `json:"org_id"`
		WorkspaceID string `json:"workspace_id"`
		TeamID      string `json:"team_id"`
		Email       string `json:"email"`
		Role        string `json:"role"`
		InviterID   string `json:"inviter_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	tenantID := s.getTenantID(r)
	if req.Role == "" {
		req.Role = "MEMBER"
	}
	if req.InviterID == "" {
		req.InviterID = "usr-system-admin"
	}

	inv, err := s.orgService.InviteUser(r.Context(), tenantID, req.OrgID, req.WorkspaceID, req.TeamID, req.Email, req.Role, req.InviterID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, inv)
}

func (s *Server) handleAcceptInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID     string `json:"id"`
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	tenantID := s.getTenantID(r)
	inv, err := s.orgService.AcceptInvitation(r.Context(), tenantID, req.ID, req.UserID)
	if err != nil {
		if err == service.ErrInvitationExpired || err == service.ErrInvitationNotPending {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err == db.ErrInvitationNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, inv)
}

func (s *Server) handleRevokeInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	tenantID := s.getTenantID(r)
	inv, err := s.orgService.RevokeInvitation(r.Context(), tenantID, req.ID)
	if err != nil {
		if err == service.ErrInvitationNotPending {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err == db.ErrInvitationNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, inv)
}

func (s *Server) handleListInvitations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := s.getTenantID(r)
	orgID := r.URL.Query().Get("org_id")

	invitations, err := s.orgService.ListInvitations(r.Context(), tenantID, orgID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, invitations)
}

// ---------------- OWNERSHIP TRANSFER HANDLERS ----------------

func (s *Server) handleInitiateTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrgID          string `json:"org_id"`
		CurrentOwnerID string `json:"current_owner_id"`
		TargetUserID   string `json:"target_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	tenantID := s.getTenantID(r)
	transfer, err := s.orgService.InitiateOwnershipTransfer(r.Context(), tenantID, req.OrgID, req.CurrentOwnerID, req.TargetUserID)
	if err != nil {
		if err == service.ErrUnauthorizedTransfer {
			respondWithError(w, http.StatusForbidden, err.Error())
			return
		}
		if err == db.ErrNodeNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, transfer)
}

func (s *Server) handleCompleteTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RequestID    string `json:"request_id"`
		TargetUserID string `json:"target_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "malformed request payload")
		return
	}

	tenantID := s.getTenantID(r)
	transfer, err := s.orgService.CompleteOwnershipTransfer(r.Context(), tenantID, req.RequestID, req.TargetUserID)
	if err != nil {
		if err == db.ErrOwnershipTransferNotFound {
			respondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, transfer)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
