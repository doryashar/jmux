package security

import (
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

// SecurityConfig holds security configuration
type SecurityConfig struct {
	Enabled          bool                   `json:"enabled"`
	Method           string                 `json:"method"`
	GlobalPassword   string                 `json:"global_password,omitempty"`
	SessionPasswords map[string]string      `json:"session_passwords,omitempty"`
	Argon2Params     *Argon2Config         `json:"argon2_params,omitempty"`
}

// Argon2Config holds Argon2 key derivation parameters
type Argon2Config struct {
	Memory      uint32 `json:"memory"`       // Memory usage in KB (default: 65536 = 64MB)
	Iterations  uint32 `json:"iterations"`   // Number of iterations (default: 3)
	Parallelism uint8  `json:"parallelism"`  // Number of threads (default: 4)
	SaltLength  uint32 `json:"salt_length"`  // Salt length in bytes (default: 16)
	KeyLength   uint32 `json:"key_length"`   // Derived key length (default: 32)
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		Enabled:          false,
		Method:           "password",
		SessionPasswords: make(map[string]string),
		Argon2Params: &Argon2Config{
			Memory:      65536, // 64 MB
			Iterations:  3,
			Parallelism: 4,
			SaltLength:  16,
			KeyLength:   32,
		},
	}
}

// PasswordAuth implements password-based authentication with session encryption
type PasswordAuth struct {
	config *SecurityConfig
}

// NewPasswordAuth creates a new password authenticator
func NewPasswordAuth(config *SecurityConfig) *PasswordAuth {
	if config.Argon2Params == nil {
		config.Argon2Params = DefaultSecurityConfig().Argon2Params
	}
	return &PasswordAuth{config: config}
}

// GenerateNonce generates a cryptographically secure random nonce
func (p *PasswordAuth) GenerateNonce() ([]byte, error) {
	nonce := make([]byte, 32) // 32 bytes for both salt and session key derivation
	_, err := rand.Read(nonce)
	return nonce, err
}

// DeriveKey derives a key from password and salt using Argon2
func (p *PasswordAuth) DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey(
		[]byte(password),
		salt,
		p.config.Argon2Params.Iterations,
		p.config.Argon2Params.Memory,
		p.config.Argon2Params.Parallelism,
		p.config.Argon2Params.KeyLength,
	)
}

// GenerateAuthResponse generates HMAC response for authentication
func (p *PasswordAuth) GenerateAuthResponse(password string, nonce []byte) ([]byte, error) {
	if len(nonce) < 16 {
		return nil, fmt.Errorf("nonce too short: need at least 16 bytes")
	}
	
	// Use first 16 bytes of nonce as salt for auth key derivation
	authSalt := nonce[:16]
	authKey := p.DeriveKey(password, authSalt)
	
	// Generate HMAC of nonce using derived auth key
	mac := hmac.New(sha256.New, authKey)
	mac.Write(nonce)
	return mac.Sum(nil), nil
}

// VerifyAuthResponse verifies HMAC response for authentication
func (p *PasswordAuth) VerifyAuthResponse(password string, nonce []byte, response []byte) bool {
	expectedResponse, err := p.GenerateAuthResponse(password, nonce)
	if err != nil {
		return false
	}
	return hmac.Equal(expectedResponse, response)
}

// DeriveSessionKey derives session encryption key from password and nonce
func (p *PasswordAuth) DeriveSessionKey(password string, nonce []byte) ([32]byte, error) {
	if len(nonce) < 32 {
		return [32]byte{}, fmt.Errorf("nonce too short: need at least 32 bytes")
	}
	
	// Use last 16 bytes of nonce as salt for session key derivation
	sessionSalt := nonce[16:32]
	sessionKey := p.DeriveKey(password, sessionSalt)
	
	var key [32]byte
	copy(key[:], sessionKey)
	return key, nil
}

// GetPasswordForSession returns the password for a given session
func (p *PasswordAuth) GetPasswordForSession(sessionName string) string {
	// Check for session-specific password first
	if password, exists := p.config.SessionPasswords[sessionName]; exists {
		return password
	}
	// Fall back to global password
	return p.config.GlobalPassword
}

// GetPasswordConfig returns the security config (for server-side access)
func (p *PasswordAuth) GetPasswordConfig() *SecurityConfig {
	return p.config
}

// EncryptedConnection wraps a connection with ChaCha20-Poly1305 encryption
type EncryptedConnection struct {
	cipher       cipher.AEAD
	nonceCounter uint64
}

// NewEncryptedConnection creates a new encrypted connection wrapper
func NewEncryptedConnection(sessionKey [32]byte) (*EncryptedConnection, error) {
	cipher, err := chacha20poly1305.New(sessionKey[:])
	if err != nil {
		return nil, err
	}
	
	return &EncryptedConnection{
		cipher:       cipher,
		nonceCounter: 0,
	}, nil
}

// Encrypt encrypts data using ChaCha20-Poly1305
func (ec *EncryptedConnection) Encrypt(data []byte) ([]byte, error) {
	// Generate nonce from counter (96-bit nonce for ChaCha20-Poly1305)
	nonce := make([]byte, 12)
	// Use counter as nonce (in little-endian format)
	for i := 0; i < 8 && i < len(nonce); i++ {
		nonce[i] = byte(ec.nonceCounter >> (i * 8))
	}
	ec.nonceCounter++
	
	// Encrypt with AEAD
	ciphertext := ec.cipher.Seal(nil, nonce, data, nil)
	
	// Prepend nonce to ciphertext for transmission
	result := make([]byte, len(nonce)+len(ciphertext))
	copy(result[:len(nonce)], nonce)
	copy(result[len(nonce):], ciphertext)
	
	return result, nil
}

// Decrypt decrypts data using ChaCha20-Poly1305
func (ec *EncryptedConnection) Decrypt(data []byte) ([]byte, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("encrypted data too short")
	}
	
	// Extract nonce and ciphertext
	nonce := data[:12]
	ciphertext := data[12:]
	
	// Decrypt with AEAD
	plaintext, err := ec.cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %v", err)
	}
	
	return plaintext, nil
}

// Protocol constants for secure handshake
const (
	SecureHandshakeMsg = "JCAT/2.0.0+SEC\n"
	AuthMethodPassword = "password"
)

// ParseAuthMessage parses authentication message format "AUTH:method\n"
func ParseAuthMessage(msg string) (string, error) {
	msg = strings.TrimSpace(msg)
	if !strings.HasPrefix(msg, "AUTH:") {
		return "", fmt.Errorf("invalid auth message format")
	}
	
	method := strings.TrimPrefix(msg, "AUTH:")
	if method == "" {
		return "", fmt.Errorf("empty auth method")
	}
	
	return method, nil
}

// FormatChallengeMessage formats challenge message "CHALLENGE:base64-nonce\n"
func FormatChallengeMessage(nonce []byte) string {
	return fmt.Sprintf("CHALLENGE:%s\n", base64.StdEncoding.EncodeToString(nonce))
}

// ParseChallengeMessage parses challenge message format
func ParseChallengeMessage(msg string) ([]byte, error) {
	msg = strings.TrimSpace(msg)
	if !strings.HasPrefix(msg, "CHALLENGE:") {
		return nil, fmt.Errorf("invalid challenge message format")
	}
	
	nonceB64 := strings.TrimPrefix(msg, "CHALLENGE:")
	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 in challenge: %v", err)
	}
	
	return nonce, nil
}

// FormatResponseMessage formats response message "RESPONSE:base64-response\n"
func FormatResponseMessage(response []byte) string {
	return fmt.Sprintf("RESPONSE:%s\n", base64.StdEncoding.EncodeToString(response))
}

// ParseResponseMessage parses response message format
func ParseResponseMessage(msg string) ([]byte, error) {
	msg = strings.TrimSpace(msg)
	if !strings.HasPrefix(msg, "RESPONSE:") {
		return nil, fmt.Errorf("invalid response message format")
	}
	
	responseB64 := strings.TrimPrefix(msg, "RESPONSE:")
	response, err := base64.StdEncoding.DecodeString(responseB64)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 in response: %v", err)
	}
	
	return response, nil
}