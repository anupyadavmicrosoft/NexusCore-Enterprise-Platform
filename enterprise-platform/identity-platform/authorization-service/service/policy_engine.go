package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ABACContext represents the attributes used for dynamic policy evaluations
type ABACContext struct {
	SubjectDept      string    `json:"subject_dept"`
	SubjectRiskScore int       `json:"subject_risk_score"`
	ClearanceLevel   int       `json:"clearance_level"` // 1: Low, 2: Medium, 3: High/Confidential
	NetworkZone      string    `json:"network_zone"`    // e.g., "internal", "external"
	IPAddress        string    `json:"ip_address"`
	CurrentTime      time.Time `json:"current_time"`
	ResourceOwnerID  string    `json:"resource_owner_id"`
	Classification   string    `json:"classification"` // e.g., "public", "internal", "highly-confidential"
}

// OPAPolicyRule models a declaratively compiled policy rule similar to OPA/Rego constraints
type OPAPolicyRule struct {
	ID             string   `json:"id"`
	Description    string   `json:"description"`
	Effect         string   `json:"effect"` // "ALLOW" or "DENY"
	RequiredDept   string   `json:"required_dept,omitempty"`
	MaxRiskScore   int      `json:"max_risk_score,omitempty"`
	MinClearance   int      `json:"min_clearance,omitempty"`
	AllowedZones   []string `json:"allowed_zones,omitempty"`
	AllowedHours   string   `json:"allowed_hours,omitempty"` // e.g. "09:00-18:00"
	Classification string   `json:"classification,omitempty"`
}

type PolicyEngine struct {
	rules []OPAPolicyRule
}

func NewPolicyEngine() *PolicyEngine {
	pe := &PolicyEngine{
		rules: make([]OPAPolicyRule, 0),
	}
	pe.loadDefaultOPAPolicies()
	return pe
}

func (pe *PolicyEngine) loadDefaultOPAPolicies() {
	pe.rules = []OPAPolicyRule{
		{
			ID:             "rule-high-confidentiality",
			Description:    "Requires High Clearance and low risk score for highly-confidential resources",
			Effect:         "ALLOW",
			RequiredDept:   "finance",
			MinClearance:   3,
			MaxRiskScore:   30,
			Classification: "highly-confidential",
		},
		{
			ID:           "rule-internal-zone-only",
			Description:  "Restricts administrative mutations from outside the secure internal corporate network zone",
			Effect:       "DENY",
			AllowedZones: []string{"internal"},
		},
		{
			ID:           "rule-business-hours-enforcement",
			Description:  "Limits bulk financial transactions to standard GMT working hours",
			Effect:       "ALLOW",
			AllowedHours: "08:00-18:00",
			RequiredDept: "finance",
		},
	}
}

// AddRule registers a declarative rule dynamically in the engine
func (pe *PolicyEngine) AddRule(rule OPAPolicyRule) {
	pe.rules = append(pe.rules, rule)
}

// EvaluateABACAndOPA matches attributes against dynamic rules, simulating compile/evaluate sequence of OPA
func (pe *PolicyEngine) EvaluateABACAndOPA(ctx context.Context, abac ABACContext, resource, action string) (bool, string, error) {
	if abac.CurrentTime.IsZero() {
		abac.CurrentTime = time.Now()
	}

	for _, rule := range pe.rules {
		// 1. Evaluate Rule Match criteria
		match := false

		// If classification matches or is highly confidential
		if rule.Classification != "" && abac.Classification == rule.Classification {
			match = true
		}

		// Deny-rules usually evaluate universally or on action type
		if len(rule.AllowedZones) > 0 {
			// Matches if action is administrative or writing
			if action == "write" || action == "delete" {
				match = true
			}
		}

		if rule.AllowedHours != "" {
			if strings.Contains(resource, "billing") || action == "payout" {
				match = true
			}
		}

		if !match {
			continue
		}

		// 2. Evaluate Rule Constraints
		if rule.Effect == "DENY" {
			// If rule is DENY, any violation triggers absolute rejection (explicit deny takes precedence)
			if len(rule.AllowedZones) > 0 {
				zoneAllowed := false
				for _, z := range rule.AllowedZones {
					if strings.ToLower(abac.NetworkZone) == strings.ToLower(z) {
						zoneAllowed = true
						break
					}
				}
				if !zoneAllowed {
					return false, fmt.Sprintf("Access Denied: OPA Policy %s violation (untrusted network zone %s)", rule.ID, abac.NetworkZone), nil
				}
			}
		}

		if rule.Effect == "ALLOW" {
			// If rule is ALLOW, all specified attributes must match
			if rule.RequiredDept != "" && abac.SubjectDept != rule.RequiredDept {
				return false, fmt.Sprintf("Access Denied: OPA Policy %s violation (Department %s mismatch)", rule.ID, abac.SubjectDept), nil
			}

			if rule.MinClearance > 0 && abac.ClearanceLevel < rule.MinClearance {
				return false, fmt.Sprintf("Access Denied: OPA Policy %s violation (Clearance %d below required %d)", rule.ID, abac.ClearanceLevel, rule.MinClearance), nil
			}

			if rule.MaxRiskScore > 0 && abac.SubjectRiskScore > rule.MaxRiskScore {
				return false, fmt.Sprintf("Access Denied: OPA Policy %s violation (Risk score %d exceeds maximum %d)", rule.ID, abac.SubjectRiskScore, rule.MaxRiskScore), nil
			}

			if rule.AllowedHours != "" {
				inHours, err := evaluateWorkingHours(abac.CurrentTime, rule.AllowedHours)
				if err != nil || !inHours {
					return false, fmt.Sprintf("Access Denied: OPA Policy %s violation (Outside permitted execution hours %s)", rule.ID, rule.AllowedHours), nil
				}
			}

			// All allowed rules passed!
			return true, fmt.Sprintf("Access Granted: OPA Policy %s satisfied", rule.ID), nil
		}
	}

	// Default Fallback behavior - If no matching rule was hit, allow by default if classification is low/public
	if abac.Classification == "highly-confidential" {
		return false, "Access Denied: Highly confidential resource requires explicit rule match", nil
	}

	return true, "Access Granted: Default policy", nil
}

// Helper to check if current time is within bounds like "08:00-18:00"
func evaluateWorkingHours(currTime time.Time, bounds string) (bool, error) {
	parts := strings.Split(bounds, "-")
	if len(parts) != 2 {
		return false, errors.New("invalid hours format")
	}

	currHour, currMin, _ := currTime.Clock()
	currMinOfDay := currHour*60 + currMin

	parseMinOfDay := func(s string) (int, error) {
		var h, m int
		if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
			return 0, err
		}
		return h*60 + m, nil
	}

	startMin, err := parseMinOfDay(parts[0])
	if err != nil {
		return false, err
	}

	endMin, err := parseMinOfDay(parts[1])
	if err != nil {
		return false, err
	}

	return currMinOfDay >= startMin && currMinOfDay <= endMin, nil
}
