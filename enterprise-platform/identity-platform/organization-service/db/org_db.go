package db

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrNodeNotFound             = errors.New("organization node not found")
	ErrWorkspaceNotFound        = errors.New("workspace not found")
	ErrTeamNotFound             = errors.New("team not found")
	ErrMemberNotFound           = errors.New("team member not found")
	ErrMemberAlreadyExists      = errors.New("user is already a member of this team")
	ErrInvitationNotFound       = errors.New("invitation not found")
	ErrOwnershipTransferNotFound = errors.New("ownership transfer request not found")
)

type OrganizationNode struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	ParentID  string    `json:"parent_id"` // Empty for root nodes
	Name      string    `json:"name"`
	Path      string    `json:"path"` // e.g. "GlobalCorp.Europe.Sales"
	Type      string    `json:"type"` // "ORGANIZATION", "DEPARTMENT"
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Workspace struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	OrgID     string    `json:"org_id"` // Organization node ID
	Name      string    `json:"name"`
	Status    string    `json:"status"` // "ACTIVE", "ARCHIVED"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Team struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	WorkspaceID string    `json:"workspace_id"` // Empty if org-level
	OrgID       string    `json:"org_id"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TeamMember struct {
	TeamID    string    `json:"team_id"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"` // "LEADER", "MEMBER"
	JoinedAt  time.Time `json:"joined_at"`
}

type Invitation struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	OrgID       string    `json:"org_id"`
	WorkspaceID string    `json:"workspace_id"`
	TeamID      string    `json:"team_id"`
	Email       string    `json:"email"`
	Role        string    `json:"role"` // Target role e.g. "ADMIN", "MEMBER"
	Status      string    `json:"status"` // "PENDING", "ACCEPTED", "REVOKED", "EXPIRED"
	InviterID   string    `json:"inviter_id"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type OwnershipTransferRequest struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	OrgID        string     `json:"org_id"`
	CurrentOwner string     `json:"current_owner"`
	TargetUser   string     `json:"target_user"`
	Status       string     `json:"status"` // "PENDING", "COMPLETED", "REJECTED"
	CreatedAt    time.Time  `json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

type OrgStore interface {
	CreateNode(ctx context.Context, node *OrganizationNode) error
	GetNodeByID(ctx context.Context, tenantID, id string) (*OrganizationNode, error)
	UpdateNode(ctx context.Context, node *OrganizationNode) error
	ListNodeChildren(ctx context.Context, tenantID, parentID string) ([]*OrganizationNode, error)
	ListAllNodes(ctx context.Context, tenantID string) ([]*OrganizationNode, error)
	DeleteNode(ctx context.Context, tenantID, id string) error

	CreateWorkspace(ctx context.Context, w *Workspace) error
	GetWorkspaceByID(ctx context.Context, tenantID, id string) (*Workspace, error)
	UpdateWorkspace(ctx context.Context, w *Workspace) error
	ListWorkspaces(ctx context.Context, tenantID, orgID string) ([]*Workspace, error)
	DeleteWorkspace(ctx context.Context, tenantID, id string) error

	CreateTeam(ctx context.Context, t *Team) error
	GetTeamByID(ctx context.Context, tenantID, id string) (*Team, error)
	UpdateTeam(ctx context.Context, t *Team) error
	ListTeams(ctx context.Context, tenantID, workspaceID string) ([]*Team, error)
	DeleteTeam(ctx context.Context, tenantID, id string) error

	AddTeamMember(ctx context.Context, tm *TeamMember) error
	RemoveTeamMember(ctx context.Context, tenantID, teamID, userID string) error
	GetTeamMembers(ctx context.Context, tenantID, teamID string) ([]*TeamMember, error)

	CreateInvitation(ctx context.Context, inv *Invitation) error
	GetInvitationByID(ctx context.Context, tenantID, id string) (*Invitation, error)
	UpdateInvitation(ctx context.Context, inv *Invitation) error
	ListInvitations(ctx context.Context, tenantID, orgID string) ([]*Invitation, error)

	CreateOwnershipTransfer(ctx context.Context, req *OwnershipTransferRequest) error
	GetOwnershipTransferByID(ctx context.Context, tenantID, id string) (*OwnershipTransferRequest, error)
	UpdateOwnershipTransfer(ctx context.Context, req *OwnershipTransferRequest) error
}

type MockOrgDB struct {
	mu                 sync.RWMutex
	nodes              map[string]*OrganizationNode
	workspaces         map[string]*Workspace
	teams              map[string]*Team
	teamMembers        []*TeamMember
	invitations        map[string]*Invitation
	ownershipTransfers map[string]*OwnershipTransferRequest
}

func NewMockOrgDB() *MockOrgDB {
	db := &MockOrgDB{
		nodes:              make(map[string]*OrganizationNode),
		workspaces:         make(map[string]*Workspace),
		teams:              make(map[string]*Team),
		teamMembers:        make([]*TeamMember, 0),
		invitations:        make(map[string]*Invitation),
		ownershipTransfers: make(map[string]*OwnershipTransferRequest),
	}
	db.seedDemoOrg()
	return db
}

func (db *MockOrgDB) seedDemoOrg() {
	// Root Global Corp Org Node
	db.nodes["org_root_1122"] = &OrganizationNode{
		ID:        "org_root_1122",
		TenantID:  "ten-8888-0001",
		ParentID:  "",
		Name:      "Global Corp",
		Path:      "GlobalCorp",
		Type:      "ORGANIZATION",
		OwnerID:   "usr-9999-0001",
		CreatedAt: time.Now().Add(-10 * 24 * time.Hour),
		UpdatedAt: time.Now().Add(-10 * 24 * time.Hour),
	}

	// Department Europe Division
	db.nodes["org_eu_2233"] = &OrganizationNode{
		ID:        "org_eu_2233",
		TenantID:  "ten-8888-0001",
		ParentID:  "org_root_1122",
		Name:      "Europe Division",
		Path:      "GlobalCorp.Europe",
		Type:      "DEPARTMENT",
		OwnerID:   "usr-9999-0001",
		CreatedAt: time.Now().Add(-9 * 24 * time.Hour),
		UpdatedAt: time.Now().Add(-9 * 24 * time.Hour),
	}

	// Department EU Sales Team
	db.nodes["org_eu_sales_4455"] = &OrganizationNode{
		ID:        "org_eu_sales_4455",
		TenantID:  "ten-8888-0001",
		ParentID:  "org_eu_2233",
		Name:      "EU Sales Team",
		Path:      "GlobalCorp.Europe.Sales",
		Type:      "DEPARTMENT",
		OwnerID:   "usr-9999-0001",
		CreatedAt: time.Now().Add(-8 * 24 * time.Hour),
		UpdatedAt: time.Now().Add(-8 * 24 * time.Hour),
	}

	// Department US Division
	db.nodes["org_us_3344"] = &OrganizationNode{
		ID:        "org_us_3344",
		TenantID:  "ten-8888-0001",
		ParentID:  "org_root_1122",
		Name:      "US Division",
		Path:      "GlobalCorp.US",
		Type:      "DEPARTMENT",
		OwnerID:   "usr-9999-0001",
		CreatedAt: time.Now().Add(-9 * 24 * time.Hour),
		UpdatedAt: time.Now().Add(-9 * 24 * time.Hour),
	}

	// Seed Workspace
	db.workspaces["ws-1"] = &Workspace{
		ID:        "ws-1",
		TenantID:  "ten-8888-0001",
		OrgID:     "org_root_1122",
		Name:      "Default Workspace",
		Status:    "ACTIVE",
		CreatedAt: time.Now().Add(-10 * 24 * time.Hour),
		UpdatedAt: time.Now().Add(-10 * 24 * time.Hour),
	}

	// Seed Team
	db.teams["team-1"] = &Team{
		ID:          "team-1",
		TenantID:    "ten-8888-0001",
		WorkspaceID: "ws-1",
		OrgID:       "org_root_1122",
		Name:      "Alpha Team",
		CreatedAt:   time.Now().Add(-10 * 24 * time.Hour),
		UpdatedAt:   time.Now().Add(-10 * 24 * time.Hour),
	}

	// Team Members
	db.teamMembers = append(db.teamMembers, &TeamMember{
		TeamID:   "team-1",
		TenantID: "ten-8888-0001",
		UserID:   "usr-9999-0001",
		Role:     "LEADER",
		JoinedAt: time.Now().Add(-10 * 24 * time.Hour),
	})
}

// ---------------- NODE OPERATIONS ----------------

func (db *MockOrgDB) CreateNode(ctx context.Context, node *OrganizationNode) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	node.CreatedAt = time.Now()
	node.UpdatedAt = time.Now()
	db.nodes[node.ID] = node
	return nil
}

func (db *MockOrgDB) GetNodeByID(ctx context.Context, tenantID, id string) (*OrganizationNode, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	node, exists := db.nodes[id]
	if !exists || node.TenantID != tenantID {
		return nil, ErrNodeNotFound
	}

	copied := *node
	return &copied, nil
}

func (db *MockOrgDB) UpdateNode(ctx context.Context, node *OrganizationNode) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	existing, exists := db.nodes[node.ID]
	if !exists || existing.TenantID != node.TenantID {
		return ErrNodeNotFound
	}

	node.UpdatedAt = time.Now()
	db.nodes[node.ID] = node
	return nil
}

func (db *MockOrgDB) ListNodeChildren(ctx context.Context, tenantID, parentID string) ([]*OrganizationNode, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var children []*OrganizationNode
	for _, n := range db.nodes {
		if n.TenantID == tenantID && n.ParentID == parentID {
			copied := *n
			children = append(children, &copied)
		}
	}
	return children, nil
}

func (db *MockOrgDB) ListAllNodes(ctx context.Context, tenantID string) ([]*OrganizationNode, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var matched []*OrganizationNode
	for _, n := range db.nodes {
		if n.TenantID == tenantID {
			copied := *n
			matched = append(matched, &copied)
		}
	}
	return matched, nil
}

func (db *MockOrgDB) DeleteNode(ctx context.Context, tenantID, id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	node, exists := db.nodes[id]
	if !exists || node.TenantID != tenantID {
		return ErrNodeNotFound
	}

	// Also delete sub-hierarchy or children references? 
	// To prevent orphaned children, we delete the node.
	delete(db.nodes, id)
	return nil
}

// ---------------- WORKSPACE OPERATIONS ----------------

func (db *MockOrgDB) CreateWorkspace(ctx context.Context, w *Workspace) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	w.CreatedAt = time.Now()
	w.UpdatedAt = time.Now()
	db.workspaces[w.ID] = w
	return nil
}

func (db *MockOrgDB) GetWorkspaceByID(ctx context.Context, tenantID, id string) (*Workspace, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	w, exists := db.workspaces[id]
	if !exists || w.TenantID != tenantID {
		return nil, ErrWorkspaceNotFound
	}

	copied := *w
	return &copied, nil
}

func (db *MockOrgDB) UpdateWorkspace(ctx context.Context, w *Workspace) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	existing, exists := db.workspaces[w.ID]
	if !exists || existing.TenantID != w.TenantID {
		return ErrWorkspaceNotFound
	}

	w.UpdatedAt = time.Now()
	db.workspaces[w.ID] = w
	return nil
}

func (db *MockOrgDB) ListWorkspaces(ctx context.Context, tenantID, orgID string) ([]*Workspace, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var result []*Workspace
	for _, w := range db.workspaces {
		if w.TenantID == tenantID && (orgID == "" || w.OrgID == orgID) {
			copied := *w
			result = append(result, &copied)
		}
	}
	return result, nil
}

func (db *MockOrgDB) DeleteWorkspace(ctx context.Context, tenantID, id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	w, exists := db.workspaces[id]
	if !exists || w.TenantID != tenantID {
		return ErrWorkspaceNotFound
	}

	delete(db.workspaces, id)
	return nil
}

// ---------------- TEAM OPERATIONS ----------------

func (db *MockOrgDB) CreateTeam(ctx context.Context, t *Team) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	db.teams[t.ID] = t
	return nil
}

func (db *MockOrgDB) GetTeamByID(ctx context.Context, tenantID, id string) (*Team, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	t, exists := db.teams[id]
	if !exists || t.TenantID != tenantID {
		return nil, ErrTeamNotFound
	}

	copied := *t
	return &copied, nil
}

func (db *MockOrgDB) UpdateTeam(ctx context.Context, t *Team) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	existing, exists := db.teams[t.ID]
	if !exists || existing.TenantID != t.TenantID {
		return ErrTeamNotFound
	}

	t.UpdatedAt = time.Now()
	db.teams[t.ID] = t
	return nil
}

func (db *MockOrgDB) ListTeams(ctx context.Context, tenantID, workspaceID string) ([]*Team, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var result []*Team
	for _, t := range db.teams {
		if t.TenantID == tenantID && (workspaceID == "" || t.WorkspaceID == workspaceID) {
			copied := *t
			result = append(result, &copied)
		}
	}
	return result, nil
}

func (db *MockOrgDB) DeleteTeam(ctx context.Context, tenantID, id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	t, exists := db.teams[id]
	if !exists || t.TenantID != tenantID {
		return ErrTeamNotFound
	}

	delete(db.teams, id)
	return nil
}

// ---------------- TEAM MEMBER OPERATIONS ----------------

func (db *MockOrgDB) AddTeamMember(ctx context.Context, tm *TeamMember) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for _, m := range db.teamMembers {
		if m.TeamID == tm.TeamID && m.UserID == tm.UserID && m.TenantID == tm.TenantID {
			return ErrMemberAlreadyExists
		}
	}

	tm.JoinedAt = time.Now()
	db.teamMembers = append(db.teamMembers, tm)
	return nil
}

func (db *MockOrgDB) RemoveTeamMember(ctx context.Context, tenantID, teamID, userID string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	idx := -1
	for i, m := range db.teamMembers {
		if m.TenantID == tenantID && m.TeamID == teamID && m.UserID == userID {
			idx = i
			break
		}
	}

	if idx == -1 {
		return ErrMemberNotFound
	}

	db.teamMembers = append(db.teamMembers[:idx], db.teamMembers[idx+1:]...)
	return nil
}

func (db *MockOrgDB) GetTeamMembers(ctx context.Context, tenantID, teamID string) ([]*TeamMember, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var members []*TeamMember
	for _, m := range db.teamMembers {
		if m.TenantID == tenantID && m.TeamID == teamID {
			copied := *m
			members = append(members, &copied)
		}
	}
	return members, nil
}

// ---------------- INVITATION OPERATIONS ----------------

func (db *MockOrgDB) CreateInvitation(ctx context.Context, inv *Invitation) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	inv.CreatedAt = time.Now()
	db.invitations[inv.ID] = inv
	return nil
}

func (db *MockOrgDB) GetInvitationByID(ctx context.Context, tenantID, id string) (*Invitation, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	inv, exists := db.invitations[id]
	if !exists || inv.TenantID != tenantID {
		return nil, ErrInvitationNotFound
	}

	copied := *inv
	return &copied, nil
}

func (db *MockOrgDB) UpdateInvitation(ctx context.Context, inv *Invitation) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	existing, exists := db.invitations[inv.ID]
	if !exists || existing.TenantID != inv.TenantID {
		return ErrInvitationNotFound
	}

	db.invitations[inv.ID] = inv
	return nil
}

func (db *MockOrgDB) ListInvitations(ctx context.Context, tenantID, orgID string) ([]*Invitation, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var result []*Invitation
	for _, inv := range db.invitations {
		if inv.TenantID == tenantID && (orgID == "" || inv.OrgID == orgID) {
			copied := *inv
			result = append(result, &copied)
		}
	}
	return result, nil
}

// ---------------- OWNERSHIP TRANSFER OPERATIONS ----------------

func (db *MockOrgDB) CreateOwnershipTransfer(ctx context.Context, req *OwnershipTransferRequest) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	req.CreatedAt = time.Now()
	db.ownershipTransfers[req.ID] = req
	return nil
}

func (db *MockOrgDB) GetOwnershipTransferByID(ctx context.Context, tenantID, id string) (*OwnershipTransferRequest, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	req, exists := db.ownershipTransfers[id]
	if !exists || req.TenantID != tenantID {
		return nil, ErrOwnershipTransferNotFound
	}

	copied := *req
	return &copied, nil
}

func (db *MockOrgDB) UpdateOwnershipTransfer(ctx context.Context, req *OwnershipTransferRequest) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	existing, exists := db.ownershipTransfers[req.ID]
	if !exists || existing.TenantID != req.TenantID {
		return ErrOwnershipTransferNotFound
	}

	db.ownershipTransfers[req.ID] = req
	return nil
}
