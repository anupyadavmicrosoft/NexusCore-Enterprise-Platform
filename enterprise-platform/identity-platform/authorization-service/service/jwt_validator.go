package service

import (
	"crypto/rsa"
	"errors"
	"fmt"

	"github.com/nexuscore/identity-platform/shared-jwt-library"
)

type JWTValidator struct {
	publicKey *rsa.PublicKey
}

func NewJWTValidator(pubKey *rsa.PublicKey) *JWTValidator {
	return &JWTValidator{
		publicKey: pubKey,
	}
}

func (j *JWTValidator) ValidateToken(tokenString string) (*jwt.Claims, error) {
	if j.publicKey == nil {
		return nil, errors.New("cryptographic validation engine not initialized: missing public key")
	}

	claims, err := jwt.VerifyTokenRS256(tokenString, j.publicKey)
	if err != nil {
		return nil, fmt.Errorf("JWT verification failed: %w", err)
	}

	return claims, nil
}
