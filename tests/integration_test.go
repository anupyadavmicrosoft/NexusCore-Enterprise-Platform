package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ==============================================================================
// INTEGRATION TESTS - MULTI-SERVICE ROUTING FLUIDITY
// Orchestrates multi-node state synchronization under strict HTTP contracts
// ==============================================================================

type AuthResponse struct {
	Authorized bool   `json:"authorized"`
	UserID     string `json:"user_id"`
	Scope      string `json:"scope"`
}

type LedgerTransaction struct {
	ID        string  `json:"tx_id"`
	AccountID string  `json:"account_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
}

func Test_EndToEnd_Ingress_Auth_Ledger_Flow(t *testing.T) {
	// 1. Setup Mock Authorization Microservice
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token != "Bearer nexus-token-secure-390" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(AuthResponse{Authorized: false})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AuthResponse{
			Authorized: true,
			UserID:     "user-enterprise-abc",
			Scope:      "write:transactions",
		})
	}))
	defer authServer.Close()

	// 2. Setup Mock Ledger Ledger Database Microservice
	ledgerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var tx LedgerTransaction
		if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if tx.Amount <= 0 {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(`{"error":"negative_balance_write_rejected"}`))
			return
		}

		tx.ID = "tx-auto-908123"
		tx.Status = "COMMITTED"

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(tx)
	}))
	defer ledgerServer.Close()

	// 3. Orchestrate API Gateway Logic combining Auth & Ledger
	apiGatewayHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Step A: Inbound Authorization check
		token := r.Header.Get("Authorization")
		req, _ := http.NewRequest("GET", authServer.URL, nil)
		req.Header.Set("Authorization", token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"upstream_authentication_denied"}`))
			return
		}

		var authResp AuthResponse
		json.NewDecoder(resp.Body).Decode(&authResp)

		// Step B: Write Transaction to ledger database
		var inboundTx LedgerTransaction
		if err := json.NewDecoder(r.Body).Decode(&inboundTx); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Marshal payload for ledger API
		ledgerBytes, _ := json.Marshal(inboundTx)
		ledgerReq, _ := http.NewRequest("POST", ledgerServer.URL, bytesNewReader(ledgerBytes))
		ledgerReq.Header.Set("Content-Type", "application/json")
		ledgerReq.Header.Set("X-Consumer-ID", authResp.UserID)

		ledgerResp, err := client.Do(ledgerReq)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"ledger_service_unreachable"}`))
			return
		}
		defer ledgerResp.Body.Close()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(ledgerResp.StatusCode)
		
		// Proxy back response
		var finalTx LedgerTransaction
		json.NewDecoder(ledgerResp.Body).Decode(&finalTx)
		json.NewEncoder(w).Encode(finalTx)
	})

	// 4. Run Test Requests against API Gateway Mock Ingress Router
	t.Run("Authorized Transaction Write Success", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		payload := `{"account_id":"acc-9921", "amount": 25000.50}`
		req := httptest.NewRequest("POST", "/api/v2/transactions", bytesNewReader([]byte(payload)))
		req.Header.Set("Authorization", "Bearer nexus-token-secure-390")
		req.Header.Set("Content-Type", "application/json")

		apiGatewayHandler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusCreated {
			t.Errorf("Expected HTTP 201 Created status, got: %d", recorder.Code)
		}

		var resTx LedgerTransaction
		json.Unmarshal(recorder.Body.Bytes(), &resTx)

		if resTx.ID != "tx-auto-908123" || resTx.Status != "COMMITTED" {
			t.Errorf("Ledger transaction commit state misaligned, got: %+v", resTx)
		}
	})

	t.Run("Unauthorized Access Rejection", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		payload := `{"account_id":"acc-9921", "amount": 500.00}`
		req := httptest.NewRequest("POST", "/api/v2/transactions", bytesNewReader([]byte(payload)))
		req.Header.Set("Authorization", "Bearer evil-forged-token-xyz")
		req.Header.Set("Content-Type", "application/json")

		apiGatewayHandler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusUnauthorized {
			t.Errorf("Expected HTTP 401 Unauthorized status, got: %d", recorder.Code)
		}
	})
}

// Utility reader mimic
type dummyReader struct {
	content []byte
	offset  int
}

func bytesNewReader(b []byte) *dummyReader {
	return &dummyReader{content: b}
}

func (r *dummyReader) Read(p []byte) (n int, err error) {
	if r.offset >= len(r.content) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(p, r.content[r.offset:])
	r.offset += n
	return n, nil
}

func (r *dummyReader) Close() error {
	return nil
}
