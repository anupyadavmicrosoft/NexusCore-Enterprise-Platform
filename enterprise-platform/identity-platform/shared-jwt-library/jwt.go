package jwt

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Standard JWT Error Definitions
var (
	ErrMalformedToken   = errors.New("the token is malformed or invalidly structured")
	ErrExpiredToken     = errors.New("the security token has expired")
	ErrInvalidSignature = errors.New("the cryptographic signature validation failed")
	ErrInvalidKey       = errors.New("the provided cryptographic key is invalid")
)

// Claims represents standard OpenID Connect and custom multi-tenant fields
type Claims struct {
	Issuer      string   `json:"iss"`
	Subject     string   `json:"sub"`
	Audience    string   `json:"aud"`
	Expiry      int64    `json:"exp"`
	NotBefore   int64    `json:"nbf"`
	IssuedAt    int64    `json:"iat"`
	JWTID       string   `json:"jti"`
	TenantID    string   `json:"tenant_id,omitempty"`
	OrgID       string   `json:"org_id,omitempty"`
	Role        string   `json:"role,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

// GenerateRSAKeyPair creates a new 4096-bit RSA key pair for cryptographic token signing.
func GenerateRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA private key: %w", err)
	}
	return privateKey, &privateKey.PublicKey, nil
}

// ParsePrivateKeyPEM parses a PEM-encoded PKCS#1 or PKCS#8 private key block.
func ParsePrivateKeyPEM(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, ErrInvalidKey
	}

	// Try PKCS#1 structure
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	// Try PKCS#8 structure
	keyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		if privKey, ok := keyInterface.(*rsa.PrivateKey); ok {
			return privKey, nil
		}
	}

	// Try generic parsing
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if privKey, ok := key.(*rsa.PrivateKey); ok {
			return privKey, nil
		}
	}

	return nil, ErrInvalidKey
}

// SignTokenRS256 issues a cryptographically signed RS256 JWT using an RSA private key.
func SignTokenRS256(claims Claims, privateKey *rsa.PrivateKey, kid string) (string, error) {
	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": kid,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	unsignedToken := fmt.Sprintf("%s.%s", headerB64, claimsB64)

	// Compute SHA-256 hash of header and claims
	hasher := sha256.New()
	hasher.Write([]byte(unsignedToken))
	hashed := hasher.Sum(nil)

	// Sign the computed hash with the RSA private key
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)
	return fmt.Sprintf("%s.%s", unsignedToken, signatureB64), nil
}

// VerifyTokenRS256 validates a token's structure, expiration, and signature using an RSA public key.
func VerifyTokenRS256(tokenString string, publicKey *rsa.PublicKey) (*Claims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, ErrMalformedToken
	}

	unsignedToken := fmt.Sprintf("%s.%s", parts[0], parts[1])
	signatureBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrMalformedToken
	}

	// Cryptographically verify signature
	hasher := sha256.New()
	hasher.Write([]byte(unsignedToken))
	hashed := hasher.Sum(nil)

	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed, signatureBytes)
	if err != nil {
		return nil, ErrInvalidSignature
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrMalformedToken
	}

	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, err
	}

	// Validate Expiration Time (exp)
	currentTime := time.Now().Unix()
	if claims.Expiry > 0 && currentTime > claims.Expiry {
		return nil, ErrExpiredToken
	}

	// Validate Not Before Time (nbf)
	if claims.NotBefore > 0 && currentTime < claims.NotBefore {
		return nil, ErrMalformedToken
	}

	return &claims, nil
}
