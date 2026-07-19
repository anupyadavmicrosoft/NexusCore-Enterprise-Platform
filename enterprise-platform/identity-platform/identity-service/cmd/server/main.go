package main

import (
	"crypto/rsa"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nexuscore/identity-platform/shared-jwt-library"
	"github.com/nexuscore/identity-platform/shared-oauth-library"
)

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
)

func init() {
	// Dynamically generate a transient keypair for demonstration runtime.
	// In production, this is loaded from Kubernetes Secret Volume.
	var err error
	privateKey, publicKey, err = jwt.GenerateRSAKeyPair()
	if err != nil {
		slog.Error("CRITICAL: Failed to generate cryptographic master RSA keys", "error", err)
		os.Exit(1)
	}
}

func main() {
	// 1. Structured JSON Logger Setup
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	slog.Info("Initializing NexusCore Central OIDC Identity Provider (identity-service)...")

	// 2. Map Route Endpoints
	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/openid-configuration", handleOIDCConfiguration)
	mux.HandleFunc("/oauth2/v1/certs", handleJWKS)
	mux.HandleFunc("/oauth2/v1/authorize", handleAuthorize)
	mux.HandleFunc("/oauth2/v1/token", handleTokenExchange)
	mux.HandleFunc("/healthz", handleHealthz)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// 3. Launch Server Async
	go func() {
		slog.Info("Central OIDC Identity Service listening", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Identity Server crashed unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	// 4. Trap Signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	slog.Info("Shutting down identity-service gracefully...")
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("Inbound OIDC Request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start).String(),
			"client_ip", r.RemoteAddr,
		)
	})
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "HEALTHY", "service": "identity-service"})
}

func handleOIDCConfiguration(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"issuer":                                "https://identity.nexuscore.com",
		"authorization_endpoint":                "https://identity.nexuscore.com/oauth2/v1/authorize",
		"token_endpoint":                        "https://identity.nexuscore.com/oauth2/v1/token",
		"jwks_uri":                              "https://identity.nexuscore.com/oauth2/v1/certs",
		"response_types_supported":              []string{"code"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email", "offline_access"},
	})
}

func handleJWKS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Expose current Public JWKS keys (Base64url encoded modules and exponents)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": "key_rotation_v2_2026_q3",
				"n":   "u1_transient_runtime_modulus_string",
				"e":   "AQAB",
			},
		},
	})
}

func handleAuthorize(w http.ResponseWriter, r *http.Request) {
	// Parse authorize PKCE structures and redirect
	clientID := r.URL.Query().Get("client_id")
	challenge := r.URL.Query().Get("code_challenge")
	state := r.URL.Query().Get("state")

	if clientID == "" || challenge == "" {
		http.Error(w, "missing mandatory client_id and code_challenge fields", http.StatusBadRequest)
		return
	}

	slog.Info("OAuth2 PKCE authorization grant evaluated", "client", clientID, "state", state)

	// Redirect back with auth code (mocking redirect callback)
	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI != "" {
		http.Redirect(w, r, redirectURI+"?code=auth_code_99a8b7c6&state="+state, http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "AUTHORIZED",
		"code":   "auth_code_99a8b7c6",
		"state":  state,
	})
}

func handleTokenExchange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req oauth.TokenRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "malformed exchange payload", http.StatusBadRequest)
		return
	}

	// 1. Validate Auth Code & Verify PKCE (Mock validation for structure)
	if req.Code == "" {
		http.Error(w, "invalid authorization code value", http.StatusUnauthorized)
		return
	}

	// 2. Generate RS256 token
	expiry := time.Now().Add(15 * time.Minute).Unix()
	claims := jwt.Claims{
		Issuer:      "https://identity.nexuscore.com",
		Subject:     "usr_99a8b7c6_e5f4_3d2c_1b0a",
		Audience:    "https://api.nexuscore.com",
		Expiry:      expiry,
		TenantID:    "ten_master_isolated",
		Role:        "SYSTEM_ADMIN",
		Permissions: []string{"*"},
	}

	accessToken, err := jwt.SignTokenRS256(claims, privateKey, "key_rotation_v2_2026_q3")
	if err != nil {
		slog.Error("Failed to sign access token during exchange", "error", err)
		http.Error(w, "cryptographic error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    900,
		"refresh_token": "nx_refresh_998877665544332211",
		"id_token":      "id_token_jwt_rs256_mock_string",
	})
}
