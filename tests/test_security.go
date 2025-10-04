package main

import (
	"fmt"
	"testing"
	"crypto/rand"
	
	"../src/jmux-go/internal/security"
)

func TestPasswordAuth(t *testing.T) {
	// Create security config
	config := &security.SecurityConfig{
		Enabled:        true,
		Method:         "password",
		GlobalPassword: "test123",
		SessionPasswords: map[string]string{
			"session1": "pass1",
			"session2": "pass2",
		},
		Argon2Params: &security.Argon2Config{
			Memory:      65536,
			Iterations:  3,
			Parallelism: 4,
			SaltLength:  16,
			KeyLength:   32,
		},
	}

	// Create password authenticator
	auth := security.NewPasswordAuth(config)

	// Test password retrieval
	if auth.GetPasswordForSession("session1") != "pass1" {
		t.Errorf("Expected session1 password to be 'pass1'")
	}
	
	if auth.GetPasswordForSession("nonexistent") != "test123" {
		t.Errorf("Expected nonexistent session to use global password")
	}

	// Test nonce generation
	nonce, err := auth.GenerateNonce()
	if err != nil {
		t.Fatalf("Failed to generate nonce: %v", err)
	}
	
	if len(nonce) != 32 {
		t.Errorf("Expected nonce length 32, got %d", len(nonce))
	}

	// Test authentication flow
	password := "test123"
	
	// Generate auth response
	response, err := auth.GenerateAuthResponse(password, nonce)
	if err != nil {
		t.Fatalf("Failed to generate auth response: %v", err)
	}

	// Verify auth response
	if !auth.VerifyAuthResponse(password, nonce, response) {
		t.Errorf("Auth response verification failed")
	}

	// Test with wrong password
	if auth.VerifyAuthResponse("wrongpass", nonce, response) {
		t.Errorf("Auth response should not verify with wrong password")
	}

	// Test session key derivation
	sessionKey, err := auth.DeriveSessionKey(password, nonce)
	if err != nil {
		t.Fatalf("Failed to derive session key: %v", err)
	}

	if len(sessionKey) != 32 {
		t.Errorf("Expected session key length 32, got %d", len(sessionKey))
	}

	fmt.Println("✓ Password authentication tests passed")
}

func TestEncryptedConnection(t *testing.T) {
	// Generate session key
	var sessionKey [32]byte
	_, err := rand.Read(sessionKey[:])
	if err != nil {
		t.Fatalf("Failed to generate session key: %v", err)
	}

	// Create encrypted connection
	encConn, err := security.NewEncryptedConnection(sessionKey)
	if err != nil {
		t.Fatalf("Failed to create encrypted connection: %v", err)
	}

	// Test data
	testData := []byte("Hello, secure world!")

	// Encrypt data
	encrypted, err := encConn.Encrypt(testData)
	if err != nil {
		t.Fatalf("Failed to encrypt data: %v", err)
	}

	if len(encrypted) <= len(testData) {
		t.Errorf("Encrypted data should be longer than plaintext (includes nonce and auth tag)")
	}

	// Decrypt data
	decrypted, err := encConn.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt data: %v", err)
	}

	if string(decrypted) != string(testData) {
		t.Errorf("Decrypted data doesn't match original. Expected: %s, Got: %s", 
			string(testData), string(decrypted))
	}

	// Test with tampered data
	tamperedData := make([]byte, len(encrypted))
	copy(tamperedData, encrypted)
	if len(tamperedData) > 0 {
		tamperedData[len(tamperedData)-1] ^= 0x01 // Flip a bit in the last byte
	}

	_, err = encConn.Decrypt(tamperedData)
	if err == nil {
		t.Errorf("Decryption should fail with tampered data")
	}

	fmt.Println("✓ Encrypted connection tests passed")
}

func TestSecurityProtocol(t *testing.T) {
	// Test protocol message parsing
	authMsg := "AUTH:password\n"
	method, err := security.ParseAuthMessage(authMsg)
	if err != nil {
		t.Fatalf("Failed to parse auth message: %v", err)
	}
	
	if method != "password" {
		t.Errorf("Expected method 'password', got '%s'", method)
	}

	// Test challenge message formatting
	nonce := []byte("testnonce1234567")
	challengeMsg := security.FormatChallengeMessage(nonce)
	
	parsedNonce, err := security.ParseChallengeMessage(challengeMsg)
	if err != nil {
		t.Fatalf("Failed to parse challenge message: %v", err)
	}

	if string(parsedNonce) != string(nonce) {
		t.Errorf("Parsed nonce doesn't match original")
	}

	// Test response message formatting
	response := []byte("testresponse1234")
	responseMsg := security.FormatResponseMessage(response)
	
	parsedResponse, err := security.ParseResponseMessage(responseMsg)
	if err != nil {
		t.Fatalf("Failed to parse response message: %v", err)
	}

	if string(parsedResponse) != string(response) {
		t.Errorf("Parsed response doesn't match original")
	}

	fmt.Println("✓ Security protocol tests passed")
}

func main() {
	fmt.Println("Running dmux security tests...")
	
	// Create a simple test runner
	tests := []struct{
		name string
		fn   func(*testing.T)
	}{
		{"TestPasswordAuth", TestPasswordAuth},
		{"TestEncryptedConnection", TestEncryptedConnection}, 
		{"TestSecurityProtocol", TestSecurityProtocol},
	}

	for _, test := range tests {
		t := &testing.T{}
		fmt.Printf("Running %s...\n", test.name)
		test.fn(t)
		if t.Failed() {
			fmt.Printf("❌ %s failed\n", test.name)
		} else {
			fmt.Printf("✅ %s passed\n", test.name)
		}
	}

	fmt.Println("\n🎉 All security tests completed!")
}