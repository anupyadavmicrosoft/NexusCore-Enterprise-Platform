package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// Standard OAuth Error definitions
var (
	ErrInvalidPKCECode = errors.New("the PKCE code verifier does not match the registered challenge")
)

// PKCEPair contains the code verifier and corresponding S256 code challenge.
type PKCEPair struct {
	Verifier  string
	Challenge string
	Method    string
}

// GeneratePKCEPair creates a cryptographically high-entropy PKCE pair conforming to RFC 7636.
func GeneratePKCEPair() (*PKCEPair, error) {
	// Generate random 32 bytes (256 bits) of entropy for the verifier
	verifierBytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, verifierBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random entropy: %w", err)
	}

	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// S256 code challenge calculation: BASE64URL-ENCODE(SHA256(ASCII(code_verifier)))
	hasher := sha256.New()
	hasher.Write([]byte(verifier))
	hashed := hasher.Sum(nil)

	challenge := base64.RawURLEncoding.EncodeToString(hashed)

	return &PKCEPair{
		Verifier:  verifier,
		Challenge: challenge,
		Method:    "S256",
	}, nil
}

// VerifyPKCE validates an incoming code verifier against a registered S256 challenge.
func VerifyPKCE(verifier, challenge string) (bool, error) {
	hasher := sha256.New()
	hasher.Write([]byte(verifier))
	hashed := hasher.Sum(nil)

	computedChallenge := base64.RawURLEncoding.EncodeToString(hashed)

	// Constant-time comparative validation to shield against timing-based enumeration side-channel attacks
	if subtle.ConstantTimeCompare([]byte(computedChallenge), []byte(challenge)) == 1 {
		return true, nil
	}

	return false, ErrInvalidPKCECode
}

// AuthorizationCodeRequest represents standard OAuth 2.0 parameters
type AuthorizationCodeRequest struct {
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri"`
	Scope               string `json:"scope"`
	State               string `json:"state"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
}

// TokenRequest represents standard parameters for /oauth2/token exchange
type TokenRequest struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
	ClientID     string `json:"client_id"`
	CodeVerifier string `json:"code_verifier"`
}
