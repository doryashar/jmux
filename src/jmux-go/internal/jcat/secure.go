package jcat

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"jmux/internal/security"
)

// SecureServer wraps Server with security capabilities
type SecureServer struct {
	*Server
	auth *security.PasswordAuth
}

// SecureClient wraps Client with security capabilities  
type SecureClient struct {
	*Client
	auth *security.PasswordAuth
}

// NewSecureServer creates a new secure jcat server
func NewSecureServer(listenAddr, rcfile string, securityConfig *security.SecurityConfig) *SecureServer {
	return &SecureServer{
		Server: NewServer(listenAddr, rcfile),
		auth:   security.NewPasswordAuth(securityConfig),
	}
}

// NewSecureClient creates a new secure jcat client
func NewSecureClient(connectAddr string, securityConfig *security.SecurityConfig) *SecureClient {
	return &SecureClient{
		Client: NewClient(connectAddr),
		auth:   security.NewPasswordAuth(securityConfig),
	}
}

// NewSecureClientWithMode creates a new secure jcat client with specified mode
func NewSecureClientWithMode(connectAddr, mode string, securityConfig *security.SecurityConfig) *SecureClient {
	return &SecureClient{
		Client: NewClientWithMode(connectAddr, mode),
		auth:   security.NewPasswordAuth(securityConfig),
	}
}

// Start starts the secure jcat server
func (s *SecureServer) Start() error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	log.Printf("secure jcat server listening on %s", s.listenAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go s.handleSecure(conn)
	}
}

// Connect connects the secure jcat client
func (c *SecureClient) Connect(sessionName, password string) error {
	conn, err := net.Dial("tcp", c.connectAddr)
	if err != nil {
		return err
	}

	// Perform secure handshake
	encryptedConn, err := c.performClientHandshake(conn, sessionName, password)
	if err != nil {
		conn.Close()
		return fmt.Errorf("secure handshake failed: %v", err)
	}

	// Continue with existing yamux protocol over encrypted connection
	return c.continueWithEncryptedConnection(encryptedConn)
}

// performClientHandshake handles the client side of secure authentication
func (c *SecureClient) performClientHandshake(conn net.Conn, sessionName, password string) (*EncryptedConn, error) {
	reader := bufio.NewReader(conn)
	
	// Read secure handshake message
	handshake, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read handshake: %v", err)
	}

	if strings.TrimSpace(handshake) != strings.TrimSpace(security.SecureHandshakeMsg) {
		return nil, fmt.Errorf("invalid secure handshake: %s", handshake)
	}

	log.Printf("Connected to secure jcat server")

	// Send authentication method
	authMsg := fmt.Sprintf("AUTH:%s\n", security.AuthMethodPassword)
	_, err = conn.Write([]byte(authMsg))
	if err != nil {
		return nil, fmt.Errorf("failed to send auth method: %v", err)
	}

	// Read challenge
	challengeMsg, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read challenge: %v", err)
	}

	nonce, err := security.ParseChallengeMessage(challengeMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse challenge: %v", err)
	}

	// Use session-specific password if not provided
	if password == "" {
		password = c.auth.GetPasswordForSession(sessionName)
	}
	
	if password == "" {
		return nil, fmt.Errorf("no password configured for session")
	}

	// Generate authentication response
	response, err := c.auth.GenerateAuthResponse(password, nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth response: %v", err)
	}

	// Send response
	responseMsg := security.FormatResponseMessage(response)
	_, err = conn.Write([]byte(responseMsg))
	if err != nil {
		return nil, fmt.Errorf("failed to send response: %v", err)
	}

	// Read authentication result
	authResult, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read auth result: %v", err)
	}

	authResult = strings.TrimSpace(authResult)
	if authResult != "AUTH_OK" {
		return nil, fmt.Errorf("authentication failed: %s", authResult)
	}

	log.Printf("Authentication successful")

	// Derive session key
	sessionKey, err := c.auth.DeriveSessionKey(password, nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to derive session key: %v", err)
	}

	// Create encrypted connection wrapper
	encryptedConn, err := NewEncryptedConn(conn, sessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create encrypted connection: %v", err)
	}

	// Send mode information over encrypted channel
	modeMsg := fmt.Sprintf("MODE:%s\n", c.mode)
	_, err = encryptedConn.Write([]byte(modeMsg))
	if err != nil {
		return nil, fmt.Errorf("failed to send mode: %v", err)
	}

	return encryptedConn, nil
}

// handleSecure handles a secure server connection
func (s *SecureServer) handleSecure(conn net.Conn) {
	remote := conn.RemoteAddr().String()
	
	// Perform secure handshake
	encryptedConn, sessionName, clientMode, err := s.performServerHandshake(conn)
	if err != nil {
		log.Printf("[%s] secure handshake failed: %v", remote, err)
		conn.Close()
		return
	}

	log.Printf("[%s] client authenticated for session '%s' in %s mode", remote, sessionName, clientMode)

	// Continue with existing server logic using encrypted connection
	s.continueWithEncryptedConnection(encryptedConn, clientMode)
}

// performServerHandshake handles the server side of secure authentication
func (s *SecureServer) performServerHandshake(conn net.Conn) (*EncryptedConn, string, string, error) {
	remote := conn.RemoteAddr().String()
	reader := bufio.NewReader(conn)

	// Send secure handshake message
	_, err := conn.Write([]byte(security.SecureHandshakeMsg))
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to send handshake: %v", err)
	}

	// Read authentication method
	authMsg, err := reader.ReadString('\n')
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to read auth method: %v", err)
	}

	method, err := security.ParseAuthMessage(authMsg)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to parse auth method: %v", err)
	}

	if method != security.AuthMethodPassword {
		return nil, "", "", fmt.Errorf("unsupported auth method: %s", method)
	}

	// Generate challenge nonce
	nonce, err := s.auth.GenerateNonce()
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate nonce: %v", err)
	}

	// Send challenge
	challengeMsg := security.FormatChallengeMessage(nonce)
	_, err = conn.Write([]byte(challengeMsg))
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to send challenge: %v", err)
	}

	// Read response
	responseMsg, err := reader.ReadString('\n')
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to read response: %v", err)
	}

	response, err := security.ParseResponseMessage(responseMsg)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to parse response: %v", err)
	}

	// Try to authenticate with session passwords
	// For now, we try all configured passwords since we don't know the session name yet
	var sessionKey [32]byte
	authenticated := false
	var _ string // usedPassword - remove if not needed

	// Try global password first
	if s.auth.GetPasswordForSession("") != "" {
		globalPassword := s.auth.GetPasswordForSession("")
		if s.auth.VerifyAuthResponse(globalPassword, nonce, response) {
			sessionKey, err = s.auth.DeriveSessionKey(globalPassword, nonce)
			if err == nil {
				authenticated = true
			}
		}
	}

	// If global password didn't work, try all session-specific passwords
	if !authenticated {
		for sessionName, password := range s.auth.GetPasswordConfig().SessionPasswords {
			if s.auth.VerifyAuthResponse(password, nonce, response) {
				sessionKey, err = s.auth.DeriveSessionKey(password, nonce)
				if err == nil {
					authenticated = true
					log.Printf("[%s] authenticated with session-specific password for '%s'", remote, sessionName)
					break
				}
			}
		}
	}

	if !authenticated {
		conn.Write([]byte("AUTH_FAIL\n"))
		return nil, "", "", fmt.Errorf("authentication failed")
	}

	// Send success
	_, err = conn.Write([]byte("AUTH_OK\n"))
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to send auth ok: %v", err)
	}

	// Create encrypted connection wrapper
	encryptedConn, err := NewEncryptedConn(conn, sessionKey)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to create encrypted connection: %v", err)
	}

	// Read mode information over encrypted channel
	modeBuffer := make([]byte, 256)
	n, err := encryptedConn.Read(modeBuffer)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to read mode: %v", err)
	}

	// Parse mode from "MODE:value\n" format
	modeStr := string(modeBuffer[:n])
	clientMode := "pair" // default
	if strings.HasPrefix(modeStr, "MODE:") && strings.Contains(modeStr, "\n") {
		clientMode = strings.TrimSpace(strings.Split(modeStr, ":")[1])
		clientMode = strings.TrimSpace(strings.Split(clientMode, "\n")[0])
	}

	// We don't know the exact session name at this point, so we return a placeholder
	// The session name will be determined later from port mapping or environment
	sessionName := "authenticated-session"

	return encryptedConn, sessionName, clientMode, nil
}


// EncryptedConn wraps a net.Conn with encryption
type EncryptedConn struct {
	conn      net.Conn
	encryptor *security.EncryptedConnection
	readBuf   []byte
}

// NewEncryptedConn creates a new encrypted connection wrapper
func NewEncryptedConn(conn net.Conn, sessionKey [32]byte) (*EncryptedConn, error) {
	encryptor, err := security.NewEncryptedConnection(sessionKey)
	if err != nil {
		return nil, err
	}

	return &EncryptedConn{
		conn:      conn,
		encryptor: encryptor,
		readBuf:   make([]byte, 0),
	}, nil
}

// Read reads encrypted data from the connection
func (ec *EncryptedConn) Read(p []byte) (n int, err error) {
	// If we have buffered data, use it first
	if len(ec.readBuf) > 0 {
		n = copy(p, ec.readBuf)
		ec.readBuf = ec.readBuf[n:]
		return n, nil
	}

	// Read encrypted data from underlying connection
	// First read the length prefix (4 bytes)
	lengthBuf := make([]byte, 4)
	_, err = io.ReadFull(ec.conn, lengthBuf)
	if err != nil {
		return 0, err
	}

	// Decode length (big-endian)
	length := int(lengthBuf[0])<<24 | int(lengthBuf[1])<<16 | int(lengthBuf[2])<<8 | int(lengthBuf[3])
	if length <= 0 || length > 64*1024 { // Sanity check: max 64KB
		return 0, fmt.Errorf("invalid encrypted message length: %d", length)
	}

	// Read encrypted data
	encryptedData := make([]byte, length)
	_, err = io.ReadFull(ec.conn, encryptedData)
	if err != nil {
		return 0, err
	}

	// Decrypt data
	decryptedData, err := ec.encryptor.Decrypt(encryptedData)
	if err != nil {
		return 0, err
	}

	// Copy to output buffer
	n = copy(p, decryptedData)
	
	// Buffer any remaining data
	if len(decryptedData) > n {
		ec.readBuf = append(ec.readBuf, decryptedData[n:]...)
	}

	return n, nil
}

// Write writes encrypted data to the connection
func (ec *EncryptedConn) Write(p []byte) (n int, err error) {
	// Encrypt data
	encryptedData, err := ec.encryptor.Encrypt(p)
	if err != nil {
		return 0, err
	}

	// Write length prefix (4 bytes, big-endian)
	length := len(encryptedData)
	lengthBuf := []byte{
		byte(length >> 24),
		byte(length >> 16),
		byte(length >> 8),
		byte(length),
	}

	// Write length prefix
	_, err = ec.conn.Write(lengthBuf)
	if err != nil {
		return 0, err
	}

	// Write encrypted data
	_, err = ec.conn.Write(encryptedData)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

// Close closes the encrypted connection
func (ec *EncryptedConn) Close() error {
	return ec.conn.Close()
}

// LocalAddr returns the local network address
func (ec *EncryptedConn) LocalAddr() net.Addr {
	return ec.conn.LocalAddr()
}

// RemoteAddr returns the remote network address
func (ec *EncryptedConn) RemoteAddr() net.Addr {
	return ec.conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines
func (ec *EncryptedConn) SetDeadline(t time.Time) error {
	return ec.conn.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
func (ec *EncryptedConn) SetReadDeadline(t time.Time) error {
	return ec.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
func (ec *EncryptedConn) SetWriteDeadline(t time.Time) error {
	return ec.conn.SetWriteDeadline(t)
}

// Helper methods for backward compatibility

// continueWithEncryptedConnection continues client connection with encrypted channel
func (c *SecureClient) continueWithEncryptedConnection(encConn *EncryptedConn) error {
	// This would be similar to the existing Connect() method but using encConn
	// For now, return an error indicating this needs implementation
	return fmt.Errorf("encrypted client connection continuation not yet implemented")
}

// continueWithEncryptedConnection continues server connection with encrypted channel  
func (s *SecureServer) continueWithEncryptedConnection(encConn *EncryptedConn, clientMode string) {
	// This would be similar to the existing handle() method but using encConn
	// For now, just log that we need to implement this
	log.Printf("encrypted server connection continuation not yet implemented for mode: %s", clientMode)
}