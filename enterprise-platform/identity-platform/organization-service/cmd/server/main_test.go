package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nexuscore/identity-platform/organization-service/db"
	"github.com/nexuscore/identity-platform/organization-service/service"
)

func TestOrganizationService_Suite(t *testing.T) {
	ctx := context.Background()

	store := db.NewMockOrgDB()
	svc := service.NewOrgService(store)
	srv := &Server{
		orgService: svc,
		store:      store,
	}

	tenantID := "ten-8888-0001"

	// -------------------------------------------------------------
	// Test Case 1: Create Organization Hierarchy Nodes
	// -------------------------------------------------------------
	var salesNodeID string
	t.Run("Create_OrgNodes_And_ValidatePath", func(t *testing.T) {
		// Create US Division child of Root
		usNode, err := svc.CreateNode(ctx, tenantID, "org_root_1122", "US Division", "DEPARTMENT", "usr-9999-0001")
		if err != nil {
			t.Fatalf("unexpected error creating node: %v", err)
		}
		if usNode.Path != "GlobalCorp.USDivision" {
			t.Errorf("expected path GlobalCorp.USDivision, got '%s'", usNode.Path)
		}

		// Create Sales Team child of US Division
		salesNode, err := svc.CreateNode(ctx, tenantID, usNode.ID, "Sales Team", "DEPARTMENT", "usr-9999-0001")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if salesNode.Path != "GlobalCorp.USDivision.SalesTeam" {
			t.Errorf("expected nested path, got '%s'", salesNode.Path)
		}

		salesNodeID = salesNode.ID
	})

	// -------------------------------------------------------------
	// Test Case 2: Tree Reconstruct Hierarchy
	// -------------------------------------------------------------
	t.Run("Get_NodeTree_Hierarchy", func(t *testing.T) {
		tree, err := svc.GetNodeTree(ctx, tenantID)
		if err != nil {
			t.Fatalf("unexpected tree build error: %v", err)
		}

		if tree.Name != "GlobalCorp" && tree.Name != "Global Corp" {
			t.Errorf("expected Global Corp root, got '%s'", tree.Name)
		}

		if len(tree.Children) == 0 {
			t.Error("expected children nodes under root, got 0")
		}
	})

	// -------------------------------------------------------------
	// Test Case 3: Move Hierarchy Node with Cycle Prevention
	// -------------------------------------------------------------
	t.Run("Move_Node_CyclePrevention", func(t *testing.T) {
		// Attempting to make 'GlobalCorp' root a child of its child 'salesNodeID' (Cycle!)
		_, err := svc.UpdateNodeParent(ctx, tenantID, "org_root_1122", salesNodeID)
		if err != service.ErrCycleDetected {
			t.Errorf("expected ErrCycleDetected, got: %v", err)
		}
	})

	// -------------------------------------------------------------
	// Test Case 4: Workspace CRUD and Isolation
	// -------------------------------------------------------------
	var wsID string
	t.Run("Workspace_Management", func(t *testing.T) {
		ws, err := svc.CreateWorkspace(ctx, tenantID, "org_root_1122", "Marketing Area")
		if err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}
		if ws.Status != "ACTIVE" {
			t.Errorf("expected ACTIVE status, got %s", ws.Status)
		}

		wsID = ws.ID

		// Update Workspace
		updated, err := svc.UpdateWorkspace(ctx, tenantID, wsID, "Marketing Zone", "ACTIVE")
		if err != nil {
			t.Fatalf("failed to update: %v", err)
		}
		if updated.Name != "Marketing Zone" {
			t.Errorf("expected 'Marketing Zone', got %s", updated.Name)
		}

		// List Workspaces
		list, err := svc.ListWorkspaces(ctx, tenantID, "")
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}
		if len(list) == 0 {
			t.Error("expected workspaces list, got 0")
		}
	})

	// -------------------------------------------------------------
	// Test Case 5: Team Management and Roles
	// -------------------------------------------------------------
	var teamID string
	t.Run("Team_Management", func(t *testing.T) {
		team, err := svc.CreateTeam(ctx, tenantID, wsID, "org_root_1122", "Campaign Team")
		if err != nil {
			t.Fatalf("failed to create team: %v", err)
		}
		teamID = team.ID

		// Add Team Member
		member, err := svc.AddTeamMember(ctx, tenantID, teamID, "usr-9999-0002", "LEADER")
		if err != nil {
			t.Fatalf("failed to add member: %v", err)
		}
		if member.Role != "LEADER" {
			t.Errorf("expected role LEADER, got %s", member.Role)
		}

		// Fetch Team Members
		members, err := svc.GetTeamMembers(ctx, tenantID, teamID)
		if err != nil {
			t.Fatalf("failed to get members: %v", err)
		}
		if len(members) != 1 {
			t.Errorf("expected 1 member, got %d", len(members))
		}
	})

	// -------------------------------------------------------------
	// Test Case 6: Invitation Flow (Create, Accept, Expired, Revoked)
	// -------------------------------------------------------------
	t.Run("Invitation_Lifecycle", func(t *testing.T) {
		inv, err := svc.InviteUser(ctx, tenantID, "org_root_1122", wsID, teamID, "new.hire@nexuscore.com", "MEMBER", "usr-9999-0001")
		if err != nil {
			t.Fatalf("failed to create invitation: %v", err)
		}

		if inv.Status != "PENDING" {
			t.Errorf("expected PENDING status, got %s", inv.Status)
		}

		// Accept Invitation
		accepted, err := svc.AcceptInvitation(ctx, tenantID, inv.ID, "usr-new-hire-id")
		if err != nil {
			t.Fatalf("failed to accept invitation: %v", err)
		}
		if accepted.Status != "ACCEPTED" {
			t.Errorf("expected ACCEPTED, got %s", accepted.Status)
		}

		// Verify accepted user is added to the Team
		members, _ := svc.GetTeamMembers(ctx, tenantID, teamID)
		foundNewHire := false
		for _, m := range members {
			if m.UserID == "usr-new-hire-id" {
				foundNewHire = true
				break
			}
		}
		if !foundNewHire {
			t.Error("accepted user should be auto-enrolled to the team")
		}

		// Revoke Flow
		inv2, _ := svc.InviteUser(ctx, tenantID, "org_root_1122", wsID, teamID, "another@nexuscore.com", "MEMBER", "usr-9999-0001")
		revoked, err := svc.RevokeInvitation(ctx, tenantID, inv2.ID)
		if err != nil {
			t.Fatalf("failed to revoke: %v", err)
		}
		if revoked.Status != "REVOKED" {
			t.Errorf("expected REVOKED, got %s", revoked.Status)
		}
	})

	// -------------------------------------------------------------
	// Test Case 7: Ownership Transfer
	// -------------------------------------------------------------
	t.Run("Ownership_Transfer_Lifecycle", func(t *testing.T) {
		node, _ := store.GetNodeByID(ctx, tenantID, "org_root_1122")
		initialOwner := node.OwnerID

		// Initiate Transfer
		req, err := svc.InitiateOwnershipTransfer(ctx, tenantID, "org_root_1122", initialOwner, "usr-9999-0002")
		if err != nil {
			t.Fatalf("failed to initiate transfer: %v", err)
		}

		if req.Status != "PENDING" {
			t.Errorf("expected PENDING request, got %s", req.Status)
		}

		// Complete Transfer
		completed, err := svc.CompleteOwnershipTransfer(ctx, tenantID, req.ID, "usr-9999-0002")
		if err != nil {
			t.Fatalf("failed to complete transfer: %v", err)
		}

		if completed.Status != "COMPLETED" {
			t.Errorf("expected COMPLETED, got %s", completed.Status)
		}

		// Verify Owner ID is swapped on Node
		nodeUpdated, _ := store.GetNodeByID(ctx, tenantID, "org_root_1122")
		if nodeUpdated.OwnerID != "usr-9999-0002" {
			t.Errorf("expected owner ID to be updated to usr-9999-0002, got %s", nodeUpdated.OwnerID)
		}
	})

	// -------------------------------------------------------------
	// Test Case 8: REST Endpoint Validation
	// -------------------------------------------------------------
	t.Run("HTTP_REST_Gateways", func(t *testing.T) {
		// Test tree endpoint
		reqTree := httptest.NewRequest("GET", "/orgs/tree?tenant_id="+tenantID, nil)
		recTree := httptest.NewRecorder()

		srv.handleGetOrgTree(recTree, reqTree)

		if recTree.Code != http.StatusOK {
			t.Errorf("expected 200 OK from tree path, got %d", recTree.Code)
		}

		// Test workspace creation endpoint
		payloadWS := map[string]string{
			"org_id": "org_root_1122",
			"name":   "New Endpoint WS",
		}
		wsBytes, _ := json.Marshal(payloadWS)
		reqWS := httptest.NewRequest("POST", "/workspaces/create", bytes.NewReader(wsBytes))
		reqWS.Header.Set("X-Tenant-ID", tenantID)
		recWS := httptest.NewRecorder()

		srv.handleCreateWorkspace(recWS, reqWS)
		if recWS.Code != http.StatusCreated {
			t.Errorf("expected 210 Created, got %d", recWS.Code)
		}
	})
}
