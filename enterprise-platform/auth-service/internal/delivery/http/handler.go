package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nexuscore/auth-service/internal/domain"
	"go.opentelemetry.io/otel"
)

type GinIdentityHandler struct {
	usecase domain.IdentityUsecase
}

func NewGinIdentityHandler(uc domain.IdentityUsecase) *GinIdentityHandler {
	return &GinIdentityHandler{usecase: uc}
}

// RegisterRoutes registers endpoints into the Gin dynamic multiplexer engine
func (h *GinIdentityHandler) RegisterRoutes(router *gin.Engine) {
	// 1. Core Platform Health Monitor
	router.GET("/healthz", h.HealthCheck)
	router.GET("/metrics", h.PrometheusMetrics)

	// 2. OpenID Connect Discovery Metadata Endpoint (OIDC Compliance)
	router.GET("/.well-known/openid-configuration", h.GetOIDCConfiguration)
	router.GET("/.well-known/jwks.json", h.GetJWKS)

	// 3. Multi-Tenant Onboarding API Router
	api := router.Group("/api/v1")
	{
		// Tenant Service
		api.POST("/tenants", h.CreateTenant)
		api.GET("/tenants/:id", h.GetTenant)

		// Organization Service
		api.POST("/organizations", h.CreateOrganization)

		// User Service
		api.POST("/auth/register", h.RegisterUser)
		api.GET("/users/:id", h.GetUser)

		// Authentication Service (OAuth2 flow)
		api.POST("/auth/login", h.Login)
		api.POST("/auth/token", h.Login) // OAuth2 Client Credential / JWT exchange fallback

		// Authorization Service (RBAC & ABAC Policy Checks)
		api.POST("/auth/authorize", h.AuthorizeCheck)
		api.POST("/auth/policies/abac", h.CreateABACPolicy)
	}
}

func (h *GinIdentityHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "UP",
		"service":   "auth-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *GinIdentityHandler) PrometheusMetrics(c *gin.Context) {
	// Exposes simulated standard scraping payloads
	metrics := `# HELP identity_auth_requests_total Total user credentials evaluations
# TYPE identity_auth_requests_total counter
identity_auth_requests_total{status="success"} 142
identity_auth_requests_total{status="failure"} 3

# HELP identity_token_verification_latency_seconds Duration metrics of JWT verifications
# TYPE identity_token_verification_latency_seconds histogram
identity_token_verification_latency_seconds_bucket{le="0.005"} 124
identity_token_verification_latency_seconds_bucket{le="0.01"} 140
identity_token_verification_latency_seconds_bucket{le="0.05"} 145
identity_token_verification_latency_seconds_sum 0.1245
identity_token_verification_latency_seconds_count 145

# HELP zero_trust_policy_evaluations_total Counter of context-aware RBAC/ABAC audits
# TYPE zero_trust_policy_evaluations_total counter
zero_trust_policy_evaluations_total{decision="allow"} 89
zero_trust_policy_evaluations_total{decision="deny"} 5
`
	c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(metrics))
}

// -----------------------------------------------------------------
// TENANT CONTROLLERS
// -----------------------------------------------------------------

func (h *GinIdentityHandler) CreateTenant(c *gin.Context) {
	tr := otel.Tracer("http-delivery")
	ctx, span := tr.Start(c.Request.Context(), "POST /api/v1/tenants")
	defer span.End()

	type tenantReq struct {
		Name   string `json:"name" binding:"required"`
		Domain string `json:"domain" binding:"required"`
	}

	var req tenantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "malformed parameters body: " + err.Error()})
		return
	}

	res, err := h.usecase.CreateTenant(ctx, req.Name, req.Domain)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, res)
}

func (h *GinIdentityHandler) GetTenant(c *gin.Context) {
	tr := otel.Tracer("http-delivery")
	ctx, span := tr.Start(c.Request.Context(), "GET /api/v1/tenants/:id")
	defer span.End()

	id := c.Param("id")
	res, err := h.usecase.GetTenant(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// -----------------------------------------------------------------
// ORGANIZATION CONTROLLERS
// -----------------------------------------------------------------

func (h *GinIdentityHandler) CreateOrganization(c *gin.Context) {
	tr := otel.Tracer("http-delivery")
	ctx, span := tr.Start(c.Request.Context(), "POST /api/v1/organizations")
	defer span.End()

	type orgReq struct {
		TenantID string `json:"tenant_id" binding:"required"`
		Name     string `json:"name" binding:"required"`
	}

	var req orgReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "malformed parameters body: " + err.Error()})
		return
	}

	res, err := h.usecase.CreateOrganization(ctx, req.TenantID, req.Name)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, res)
}

// -----------------------------------------------------------------
// USER CONTROLLERS
// -----------------------------------------------------------------

func (h *GinIdentityHandler) RegisterUser(c *gin.Context) {
	tr := otel.Tracer("http-delivery")
	ctx, span := tr.Start(c.Request.Context(), "POST /api/v1/auth/register")
	defer span.End()

	type regReq struct {
		TenantID       string `json:"tenant_id" binding:"required"`
		OrganizationID string `json:"organization_id" binding:"required"`
		Email          string `json:"email" binding:"required,email"`
		Password       string `json:"password" binding:"required"`
		FullName       string `json:"full_name" binding:"required"`
		Role           string `json:"role"`
	}

	var req regReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userEntity := domain.User{
		TenantID:       req.TenantID,
		OrganizationID: req.OrganizationID,
		Email:          req.Email,
		FullName:       req.FullName,
		Role:           req.Role,
	}

	res, err := h.usecase.RegisterUser(ctx, userEntity, req.Password)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, res)
}

func (h *GinIdentityHandler) GetUser(c *gin.Context) {
	tr := otel.Tracer("http-delivery")
	ctx, span := tr.Start(c.Request.Context(), "GET /api/v1/users/:id")
	defer span.End()

	id := c.Param("id")
	res, err := h.usecase.GetUser(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// -----------------------------------------------------------------
// AUTHENTICATION (LOGIN & JWT ISSUANCE)
// -----------------------------------------------------------------

func (h *GinIdentityHandler) Login(c *gin.Context) {
	tr := otel.Tracer("http-delivery")
	ctx, span := tr.Start(c.Request.Context(), "POST /api/v1/auth/login")
	defer span.End()

	type loginReq struct {
		TenantID string `json:"tenant_id" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.usecase.Authenticate(ctx, req.TenantID, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// -----------------------------------------------------------------
// ZERO-TRUST AUTHORIZATION CONTROLLER
// -----------------------------------------------------------------

func (h *GinIdentityHandler) AuthorizeCheck(c *gin.Context) {
	tr := otel.Tracer("http-delivery")
	ctx, span := tr.Start(c.Request.Context(), "POST /api/v1/auth/authorize")
	defer span.End()

	var req domain.AuthzCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.usecase.EvaluateAuthorization(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !res.Allowed {
		c.JSON(http.StatusForbidden, res)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *GinIdentityHandler) CreateABACPolicy(c *gin.Context) {
	tr := otel.Tracer("http-delivery")
	ctx, span := tr.Start(c.Request.Context(), "POST /api/v1/auth/policies/abac")
	defer span.End()

	var req domain.ABACPolicy
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.usecase.CreateABACPolicy(ctx, req)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "POLICY_COMMITTED"})
}

// -----------------------------------------------------------------
// OIDC / OAUTH2 META SERVICES
// -----------------------------------------------------------------

func (h *GinIdentityHandler) GetOIDCConfiguration(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"issuer":                                "https://auth.nexuscore.io",
		"authorization_endpoint":                "https://auth.nexuscore.io/oauth/authorize",
		"token_endpoint":                        "https://auth.nexuscore.io/api/v1/auth/token",
		"userinfo_endpoint":                     "https://auth.nexuscore.io/api/v1/users/me",
		"jwks_uri":                              "https://auth.nexuscore.io/.well-known/jwks.json",
		"scopes_supported":                      []string{"openid", "profile", "email", "offline_access"},
		"response_types_supported":              []string{"code", "token", "id_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
	})
}

func (h *GinIdentityHandler) GetJWKS(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"keys": []gin.H{
			{
				"kty": "RSA",
				"use": "sig",
				"kid": "nexuscore_jwks_key_id_v1",
				"alg": "RS256",
				"n":   "u1W_M6e_M-lS5tG6...", // simulated public key modulus
				"e":   "AQAB",
			},
		},
	})
}
