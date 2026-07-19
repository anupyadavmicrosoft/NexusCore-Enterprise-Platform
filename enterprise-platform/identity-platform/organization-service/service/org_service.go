package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nexuscore/identity-platform/organization-service/db"
)

var (
	ErrCycleDetected          = errors.New("cannot set parent node: cycle detected in hierarchy")
	ErrInvalidNodeName        = errors.New("invalid organization/department name: must be at least 2 characters")
	ErrUnauthorizedTransfer   = errors.New("unauthorized: only the current owner can transfer ownership")
	ErrInvitationExpired      = errors.New("invitation has expired")
	ErrInvitationNotPending   = errors.New("invitation is not in a pending state")
)

type OrgService struct {
	store db.OrgStore
}

func NewOrgService(store db.OrgStore) *OrgService {
	return &OrgService{
		store: store,
	}
}

func GenerateUUID(prefix string) string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}

// ---------------- HIERARCHY & DEPARTMENT OPERATIONS ----------------

type OrgTreeNode struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Path      string         `json:"path"`
	Type      string         `json:"type"`
	OwnerID   string         `json:"owner_id"`
	Children  []*OrgTreeNode `json:"children,omitempty"`
}

func (s *OrgService) CreateNode(ctx context.Context, tenantID, parentID, name, nodeType, ownerID string) (*db.OrganizationNode, error) {
	if len(name) < 2 {
		return nil, ErrInvalidNodeName
	}

	path := name
	if parentID != "" {
		parent, err := s.store.GetNodeByID(ctx, tenantID, parentID)
		if err != nil {
			return nil, fmt.Errorf("invalid parent node: %w", err)
		}
		path = parent.Path + "." + strings.ReplaceAll(name, " ", "")
	}

	node := &db.OrganizationNode{
		ID:        GenerateUUID("org"),
		TenantID:  tenantID,
		ParentID:  parentID,
		Name:      name,
		Path:      path,
		Type:      nodeType,
		OwnerID:   ownerID,
	}

	err := s.store.CreateNode(ctx, node)
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (s *OrgService) GetNodeTree(ctx context.Context, tenantID string) (*OrgTreeNode, error) {
	all, err := s.store.ListAllNodes(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if len(all) == 0 {
		return nil, errors.New("no organization nodes found for this tenant")
	}

	// Group nodes by parent ID
	nodesMap := make(map[string]*OrgTreeNode)
	childrenMap := make(map[string][]*OrgTreeNode)

	var rootNode *OrgTreeNode

	for _, n := range all {
		tn := &OrgTreeNode{
			ID:       n.ID,
			Name:     n.Name,
			Path:     n.Path,
			Type:     n.Type,
			OwnerID:  n.OwnerID,
			Children: []*OrgTreeNode{},
		}
		nodesMap[n.ID] = tn
		if n.ParentID == "" {
			rootNode = tn
		} else {
			childrenMap[n.ParentID] = append(childrenMap[n.ParentID], tn)
		}
	}

	// Reconstruct the tree hierarchy
	for id, tn := range nodesMap {
		if children, exists := childrenMap[id]; exists {
			tn.Children = children
		}
	}

	if rootNode == nil && len(all) > 0 {
		// Fallback: pick the first node with no parent or shortest path as root
		rootNode = nodesMap[all[0].ID]
	}

	return rootNode, nil
}

func (s *OrgService) UpdateNodeParent(ctx context.Context, tenantID, id, newParentID string) (*db.OrganizationNode, error) {
	node, err := s.store.GetNodeByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if newParentID == id {
		return nil, ErrCycleDetected
	}

	// Detect cycles
	currParentID := newParentID
	for currParentID != "" {
		pNode, err := s.store.GetNodeByID(ctx, tenantID, currParentID)
		if err != nil {
			return nil, fmt.Errorf("parent lookup failed: %w", err)
		}
		if pNode.ID == id {
			return nil, ErrCycleDetected
		}
		currParentID = pNode.ParentID
	}

	node.ParentID = newParentID

	// Recalculate path
	if newParentID != "" {
		parent, err := s.store.GetNodeByID(ctx, tenantID, newParentID)
		if err == nil {
			node.Path = parent.Path + "." + strings.ReplaceAll(node.Name, " ", "")
		}
	} else {
		node.Path = node.Name
	}

	err = s.store.UpdateNode(ctx, node)
	if err != nil {
		return nil, err
	}

	return node, nil
}

// ---------------- WORKSPACE MANAGEMENT ----------------

func (s *OrgService) CreateWorkspace(ctx context.Context, tenantID, orgID, name string) (*db.Workspace, error) {
	if name == "" {
		return nil, errors.New("workspace name cannot be empty")
	}

	ws := &db.Workspace{
		ID:       GenerateUUID("ws"),
		TenantID: tenantID,
		OrgID:    orgID,
		Name:     name,
		Status:   "ACTIVE",
	}

	err := s.store.CreateWorkspace(ctx, ws)
	if err != nil {
		return nil, err
	}
	return ws, nil
}

func (s *OrgService) ListWorkspaces(ctx context.Context, tenantID, orgID string) ([]*db.Workspace, error) {
	return s.store.ListWorkspaces(ctx, tenantID, orgID)
}

func (s *OrgService) UpdateWorkspace(ctx context.Context, tenantID, id, name, status string) (*db.Workspace, error) {
	ws, err := s.store.GetWorkspaceByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if name != "" {
		ws.Name = name
	}
	if status != "" {
		ws.Status = status
	}

	err = s.store.UpdateWorkspace(ctx, ws)
	if err != nil {
		return nil, err
	}
	return ws, nil
}

func (s *OrgService) DeleteWorkspace(ctx context.Context, tenantID, id string) error {
	return s.store.DeleteWorkspace(ctx, tenantID, id)
}

// ---------------- TEAM MANAGEMENT ----------------

func (s *OrgService) CreateTeam(ctx context.Context, tenantID, workspaceID, orgID, name string) (*db.Team, error) {
	if name == "" {
		return nil, errors.New("team name cannot be empty")
	}

	t := &db.Team{
		ID:          GenerateUUID("team"),
		TenantID:    tenantID,
		WorkspaceID: workspaceID,
		OrgID:       orgID,
		Name:        name,
	}

	err := s.store.CreateTeam(ctx, t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (s *OrgService) ListTeams(ctx context.Context, tenantID, workspaceID string) ([]*db.Team, error) {
	return s.store.ListTeams(ctx, tenantID, workspaceID)
}

func (s *OrgService) AddTeamMember(ctx context.Context, tenantID, teamID, userID, role string) (*db.TeamMember, error) {
	tm := &db.TeamMember{
		TeamID:   teamID,
		TenantID: tenantID,
		UserID:   userID,
		Role:     role,
	}

	err := s.store.AddTeamMember(ctx, tm)
	if err != nil {
		return nil, err
	}
	return tm, nil
}

func (s *OrgService) RemoveTeamMember(ctx context.Context, tenantID, teamID, userID string) error {
	return s.store.RemoveTeamMember(ctx, tenantID, teamID, userID)
}

func (s *OrgService) GetTeamMembers(ctx context.Context, tenantID, teamID string) ([]*db.TeamMember, error) {
	return s.store.GetTeamMembers(ctx, tenantID, teamID)
}

// ---------------- INVITATION SYSTEM ----------------

func (s *OrgService) InviteUser(ctx context.Context, tenantID, orgID, workspaceID, teamID, email, role, inviterID string) (*db.Invitation, error) {
	if !strings.Contains(email, "@") {
		return nil, errors.New("invalid email address format")
	}

	inv := &db.Invitation{
		ID:          GenerateUUID("inv"),
		TenantID:    tenantID,
		OrgID:       orgID,
		WorkspaceID: workspaceID,
		TeamID:      teamID,
		Email:       email,
		Role:        role,
		Status:      "PENDING",
		InviterID:   inviterID,
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
	}

	err := s.store.CreateInvitation(ctx, inv)
	if err != nil {
		return nil, err
	}
	return inv, nil
}

func (s *OrgService) AcceptInvitation(ctx context.Context, tenantID, id, userID string) (*db.Invitation, error) {
	inv, err := s.store.GetInvitationByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if inv.Status != "PENDING" {
		return nil, ErrInvitationNotPending
	}

	if time.Now().After(inv.ExpiresAt) {
		inv.Status = "EXPIRED"
		_ = s.store.UpdateInvitation(ctx, inv)
		return nil, ErrInvitationExpired
	}

	inv.Status = "ACCEPTED"
	err = s.store.UpdateInvitation(ctx, inv)
	if err != nil {
		return nil, err
	}

	// Auto-add user to team if specified
	if inv.TeamID != "" {
		_, _ = s.AddTeamMember(ctx, tenantID, inv.TeamID, userID, inv.Role)
	}

	return inv, nil
}

func (s *OrgService) RevokeInvitation(ctx context.Context, tenantID, id string) (*db.Invitation, error) {
	inv, err := s.store.GetInvitationByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if inv.Status != "PENDING" {
		return nil, ErrInvitationNotPending
	}

	inv.Status = "REVOKED"
	err = s.store.UpdateInvitation(ctx, inv)
	if err != nil {
		return nil, err
	}
	return inv, nil
}

func (s *OrgService) ListInvitations(ctx context.Context, tenantID, orgID string) ([]*db.Invitation, error) {
	return s.store.ListInvitations(ctx, tenantID, orgID)
}

// ---------------- OWNERSHIP TRANSFER ----------------

func (s *OrgService) InitiateOwnershipTransfer(ctx context.Context, tenantID, orgID, currentOwnerID, targetUserID string) (*db.OwnershipTransferRequest, error) {
	node, err := s.store.GetNodeByID(ctx, tenantID, orgID)
	if err != nil {
		return nil, err
	}

	if node.OwnerID != currentOwnerID {
		return nil, ErrUnauthorizedTransfer
	}

	req := &db.OwnershipTransferRequest{
		ID:           GenerateUUID("owtr"),
		TenantID:     tenantID,
		OrgID:        orgID,
		CurrentOwner: currentOwnerID,
		TargetUser:   targetUserID,
		Status:       "PENDING",
	}

	err = s.store.CreateOwnershipTransfer(ctx, req)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (s *OrgService) CompleteOwnershipTransfer(ctx context.Context, tenantID, requestID, targetUserID string) (*db.OwnershipTransferRequest, error) {
	req, err := s.store.GetOwnershipTransferByID(ctx, tenantID, requestID)
	if err != nil {
		return nil, err
	}

	if req.Status != "PENDING" {
		return nil, errors.New("ownership transfer request is not in PENDING state")
	}

	if req.TargetUser != targetUserID {
		return nil, errors.New("unauthorized target user mismatch")
	}

	node, err := s.store.GetNodeByID(ctx, tenantID, req.OrgID)
	if err != nil {
		return nil, err
	}

	// Execute ownership swap
	node.OwnerID = targetUserID
	err = s.store.UpdateNode(ctx, node)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	req.Status = "COMPLETED"
	req.CompletedAt = &now

	err = s.store.UpdateOwnershipTransfer(ctx, req)
	if err != nil {
		return nil, err
	}

	return req, nil
}
