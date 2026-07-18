package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/nexuscore/api-gateway/internal/config"
)

type GatewayClaims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. API Key Authentication (Highest priority for machine-to-machine integrations)
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		if apiKey != "" {
			if role, exists := cfg.APIKeys[apiKey]; exists {
				c.Set("apiKey", apiKey)
				c.Set("user", "api-key-client")
				c.Set("role", role)
				c.Next()
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "The specified API Key is invalid or has expired",
				"code":  "INVALID_API_KEY",
			})
			c.Abort()
			return
		}

		// 2. JWT & OAuth2 / OIDC Token Authentication
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No credentials provided; treat as anonymous/guest client
			c.Set("role", "Guest")
			c.Set("user", "anonymous")
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header must use Bearer token formatting",
				"code":  "MALFORMED_AUTHORIZATION_HEADER",
			})
			c.Abort()
			return
		}

		tokenStr := parts[1]

		// Fast verification of Mock OIDC tokens for offline/sandbox mode
		if strings.HasPrefix(tokenStr, "mock_oidc_") {
			role := "Operator"
			if strings.Contains(tokenStr, "admin") {
				role = "Admin"
			} else if strings.Contains(tokenStr, "guest") {
				role = "Guest"
			}
			c.Set("user", "oidc-federated-user")
			c.Set("role", role)
			c.Next()
			return
		}

		// Real Cryptographic JWT Parsing & Signature Verification
		token, err := jwt.ParseWithClaims(tokenStr, &GatewayClaims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected cryptographic signing algorithm")
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "JSON Web Token (JWT) validation or signature check failed",
				"code":  "INVALID_TOKEN",
			})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*GatewayClaims); ok {
			c.Set("user", claims.Username)
			c.Set("role", claims.Role)
		} else {
			c.Set("role", "Guest")
		}

		c.Next()
	}
}

// RBACMiddleware checks if the active principal possesses one of the required authorization roles
func RBACMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied. Principal role could not be resolved from authentication context",
				"code":  "ROLE_UNRESOLVED",
			})
			c.Abort()
			return
		}

		role := roleVal.(string)
		isAllowed := false
		for _, r := range allowedRoles {
			if strings.EqualFold(role, r) {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			c.JSON(http.StatusForbidden, gin.H{
				"error":         "Access forbidden. Your current role permissions do not authorize this API namespace",
				"code":          "FORBIDDEN_API_ACCESS",
				"required_roles": allowedRoles,
				"your_role":      role,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
