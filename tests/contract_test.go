package tests

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// ==============================================================================
// CONTRACT TESTS - CONSUMER-DRIVEN API SPEC VALIDATIONS (Pact Model)
// Validates structural schemas and required payload envelopes
// ==============================================================================

// Expected Schema Contract definition for the Auth Service Endpoint
var ExpectedAuthContract = map[string]string{
	"user_id":  "string",
	"username": "string",
	"status":   "string",
	"privileges": "array",
}

// ValidateJSONContract matches raw JSON strings against the expected blueprint
func ValidateJSONContract(schema map[string]string, jsonStr string) error {
	var rawData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &rawData); err != nil {
		return fmt.Errorf("invalid json format: %v", err)
	}

	for key, expectedType := range schema {
		val, exists := rawData[key]
		if !exists {
			return fmt.Errorf("contract violation: missing required field '%s'", key)
		}

		// Basic type matching
		switch expectedType {
		case "string":
			if _, ok := val.(string); !ok {
				return fmt.Errorf("contract violation: field '%s' must be of type string", key)
			}
		case "number":
			if _, ok := val.(float64); !ok {
				return fmt.Errorf("contract violation: field '%s' must be of type number", key)
			}
		case "array":
			if _, ok := val.([]interface{}); !ok {
				return fmt.Errorf("contract violation: field '%s' must be of type array", key)
			}
		}
	}
	return nil
}

func Test_AuthService_Contract_Compliance(t *testing.T) {
	t.Run("Valid JSON Response Contract", func(t *testing.T) {
		validResponsePayload := `{
			"user_id": "auth-821-xyz",
			"username": "infra_controller_admin",
			"status": "ACTIVE_SYSTEM",
			"privileges": ["manage:pods", "write:network_policy"]
		}`

		err := ValidateJSONContract(ExpectedAuthContract, validResponsePayload)
		if err != nil {
			t.Fatalf("Contract failed validation on accurate payload: %v", err)
		}
	})

	t.Run("Contract Violation on Missing Field", func(t *testing.T) {
		deficientPayload := `{
			"username": "broken_infra_controller",
			"status": "SUSPENDED",
			"privileges": []
		}`

		err := ValidateJSONContract(ExpectedAuthContract, deficientPayload)
		if err == nil {
			t.Fatal("Expected contract validation to fail due to missing 'user_id', but it passed")
		}

		if !strings.Contains(err.Error(), "missing required field 'user_id'") {
			t.Errorf("Unexpected contract violation output: %v", err)
		}
	})

	t.Run("Contract Violation on Wrong Type", func(t *testing.T) {
		corruptedTypePayload := `{
			"user_id": 1208923091,
			"username": "wrong_type_admin",
			"status": "ACTIVE",
			"privileges": ["write"]
		}`

		err := ValidateJSONContract(ExpectedAuthContract, corruptedTypePayload)
		if err == nil {
			t.Fatal("Expected contract validation to fail due to type mismatch on 'user_id', but it passed")
		}

		if !strings.Contains(err.Error(), "must be of type string") {
			t.Errorf("Unexpected contract error payload: %v", err)
		}
	})
}
