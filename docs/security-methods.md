# dmux Security Methods

This document outlines various security approaches for protecting dmux session communications between server and client connections.

## Current Architecture

The jcat protocol currently uses:
- Plain TCP connections over configurable ports (default 12345+)
- Simple handshake: `"JCAT/2.0.0\n"`
- Mode exchange: `"MODE:value\n"` (pair/view/rogue)
- yamux multiplexing for control/data channels
- No encryption or authentication

## Security Threat Model

**Threats to Address:**
- **Eavesdropping**: Network sniffers can read all session data
- **Unauthorized Access**: Anyone can connect to shared sessions
- **Man-in-the-Middle**: Attackers can intercept and modify traffic
- **Session Hijacking**: Malicious users can take over sessions
- **Replay Attacks**: Captured authentication can be reused

**Assets to Protect:**
- Terminal I/O data (commands, output, sensitive information)
- Session control (window resizing, mode changes)
- User authentication credentials
- Session metadata (user names, session names)

## Security Method Comparison

### 🔐 Method 1: TLS with Client Certificates

**Overview:**
Implements mutual TLS authentication using X.509 certificates. Server and clients each have certificates, providing strong cryptographic authentication and encryption.

**Technical Implementation:**
```go
// Server side
cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
config := &tls.Config{
    Certificates: []tls.Certificate{cert},
    ClientAuth:   tls.RequireAndVerifyClientCert,
    ClientCAs:    caCertPool,
}
listener, err := tls.Listen("tcp", addr, config)

// Client side
cert, err := tls.LoadX509KeyPair("client.crt", "client.key") 
config := &tls.Config{
    Certificates: []tls.Certificate{cert},
    RootCAs:      caCertPool,
}
conn, err := tls.Dial("tcp", addr, config)
```

**Protocol Flow:**
1. TCP connection established
2. TLS handshake with mutual certificate verification
3. Encrypted channel established
4. Standard jcat protocol over TLS
5. yamux multiplexing over encrypted connection

**Configuration:**
```yaml
security:
  method: "tls"
  server_cert: "/path/to/server.crt"
  server_key: "/path/to/server.key"
  ca_cert: "/path/to/ca.crt"
  client_cert: "/path/to/client.crt"
  client_key: "/path/to/client.key"
```

**Pros:**
- ✅ **Industry Standard**: Battle-tested TLS 1.3 encryption
- ✅ **Strong Authentication**: Cryptographic certificate-based auth
- ✅ **Perfect Forward Secrecy**: Session keys are ephemeral
- ✅ **Mutual Authentication**: Both server and client verify identity
- ✅ **Go Standard Library**: Excellent `crypto/tls` support
- ✅ **Scalability**: Different certificates per user/role
- ✅ **Audit Trail**: Certificate-based identity tracking
- ✅ **Key Management**: Standard PKI practices apply

**Cons:**
- ❌ **Complex Setup**: Certificate generation, signing, distribution
- ❌ **CA Infrastructure**: Need certificate authority or self-signed certs
- ❌ **Client Configuration**: Each client needs certificate setup
- ❌ **Certificate Management**: Expiration, revocation, renewal
- ❌ **Overkill for Simple Use**: Too complex for basic personal use
- ❌ **Trust on First Use**: Self-signed certificates require manual verification

**Use Cases:**
- Enterprise environments with existing PKI
- High-security requirements
- Multiple users/teams with different access levels
- Compliance requirements (SOC2, HIPAA, etc.)

**Security Level:** ⭐⭐⭐⭐⭐ (Excellent)

---

### 🔑 Method 2: Pre-Shared Key (PSK) with ChaCha20-Poly1305

**Overview:**
Uses a single shared secret for all connections, with strong symmetric encryption. Keys are derived per-session but authentication uses the same PSK.

**Technical Implementation:**
```go
type PSKAuth struct {
    key [32]byte // 256-bit pre-shared key
}

func (p *PSKAuth) deriveSessionKey(nonce []byte) [32]byte {
    return hkdf.Expand(sha256.New(), p.key[:], nonce, 32)
}

func (p *PSKAuth) encrypt(data []byte, sessionKey [32]byte) []byte {
    cipher, _ := chacha20poly1305.New(sessionKey[:])
    nonce := make([]byte, cipher.NonceSize())
    rand.Read(nonce)
    return cipher.Seal(nonce, nonce, data, nil)
}
```

**Protocol Flow:**
1. TCP connection established
2. Server sends random nonce (32 bytes)
3. Client derives session key: HKDF(PSK, nonce)
4. Client sends HMAC(nonce, PSK) for authentication
5. Server verifies HMAC using same PSK
6. Both sides use session key for ChaCha20-Poly1305 encryption
7. All subsequent traffic encrypted

**Configuration:**
```yaml
security:
  method: "psk"
  key: "base64-encoded-256-bit-key"
  # OR
  key_file: "/path/to/psk.key"
```

**Pros:**
- ✅ **Simple Configuration**: Single key for all users
- ✅ **Strong Encryption**: ChaCha20-Poly1305 AEAD cipher
- ✅ **Good Performance**: Symmetric crypto is fast
- ✅ **Per-Session Keys**: Different key per connection via HKDF
- ✅ **No Certificate Management**: Avoid PKI complexity
- ✅ **Easy Key Rotation**: Update single key file
- ✅ **Replay Protection**: Nonce prevents replay attacks

**Cons:**
- ❌ **Single Point of Failure**: One key compromises all sessions
- ❌ **No User Identification**: Can't distinguish between users
- ❌ **Key Distribution**: How to securely share the PSK
- ❌ **Revocation Issues**: Can't revoke access for individual users
- ❌ **Key Storage**: PSK must be stored securely on all systems

**Use Cases:**
- Small teams with shared infrastructure
- Personal use with trusted users
- Environments where PKI is overkill
- Quick security implementation

**Security Level:** ⭐⭐⭐⭐ (Good)

---

### 🎯 Method 3: Password Authentication + Session Encryption (RECOMMENDED)

**Overview:**
Combines user-friendly password authentication with strong per-session encryption. Uses modern key derivation (Argon2) and authenticated encryption.

**Technical Implementation:**
```go
type PasswordAuth struct {
    argon2Params Argon2Params
}

type Argon2Params struct {
    Memory      uint32  // 64 MB
    Iterations  uint32  // 3
    Parallelism uint8   // 4
    SaltLength  uint32  // 16
    KeyLength   uint32  // 32
}

func (p *PasswordAuth) deriveKey(password string, salt []byte) []byte {
    return argon2.IDKey([]byte(password), salt, 
        p.argon2Params.Iterations,
        p.argon2Params.Memory,
        p.argon2Params.Parallelism,
        p.argon2Params.KeyLength)
}

func (p *PasswordAuth) authenticateAndDeriveSessionKey(password string, nonce []byte) ([32]byte, error) {
    // Derive authentication key
    authKey := p.deriveKey(password, nonce[:16])
    
    // Derive session encryption key  
    sessionKey := p.deriveKey(password, nonce[16:])
    
    return [32]byte(sessionKey), nil
}
```

**Protocol Flow:**
1. TCP connection established
2. Server: `"JCAT/2.0.0+SEC\n"`
3. Client: `"AUTH:password\n"`
4. Server: `"CHALLENGE:" + base64(32-byte-nonce) + "\n"`
5. Client derives response: `HMAC-SHA256(Argon2(password, nonce), nonce)`
6. Client: `"RESPONSE:" + base64(response) + "\n"`
7. Server verifies response using same derivation
8. Server: `"AUTH_OK\n"` or `"AUTH_FAIL\n"`
9. If OK: Both derive session key using Argon2(password, nonce)
10. All traffic encrypted with ChaCha20-Poly1305(session_key)

**Configuration:**
```yaml
security:
  method: "password"
  password: "session-password"  # Global default
  # OR per-session passwords
  session_passwords:
    session1: "password1"
    session2: "password2"
  argon2:
    memory: 65536      # 64 MB
    iterations: 3
    parallelism: 4
    salt_length: 16
```

**Pros:**
- ✅ **User-Friendly**: Familiar password-based authentication
- ✅ **Per-Session Security**: Different passwords per session possible
- ✅ **Strong Key Derivation**: Argon2 resists brute force attacks
- ✅ **Session-Specific Keys**: Each connection gets unique encryption key
- ✅ **Replay Protection**: Challenge-response prevents replay attacks
- ✅ **Good Balance**: Security vs usability sweet spot
- ✅ **Password Rotation**: Easy to change passwords
- ✅ **Flexible Configuration**: Global or per-session passwords

**Cons:**
- ❌ **Password Management**: Users must remember/store passwords
- ❌ **Brute Force Risk**: Vulnerable to password guessing attacks
- ❌ **Human Factor**: Weak passwords reduce security
- ❌ **No Forward Secrecy**: Compromised password affects old sessions
- ❌ **Dictionary Attacks**: Common passwords easily broken

**Use Cases:**
- General purpose secure sharing
- Teams with moderate security requirements
- Personal use with sensitive data
- Development environments with security needs

**Security Level:** ⭐⭐⭐⭐ (Good)

---

### 🚀 Method 4: SSH-Style Host Key + User Auth

**Overview:**
Mimics SSH's security model with server host keys for server authentication and flexible user authentication methods.

**Technical Implementation:**
```go
type SSHStyleAuth struct {
    hostKey     ed25519.PrivateKey  // Server's host key
    knownHosts  map[string][]byte   // Client's known hosts
    userDB      UserDatabase        // User authentication
}

type UserDatabase interface {
    AuthenticateUser(username, password string) bool
    GetUserPublicKey(username string) (ed25519.PublicKey, bool)
}

func (s *SSHStyleAuth) serverHandshake(conn net.Conn) error {
    // Send host key
    hostKeyBytes := s.hostKey.Public().(ed25519.PublicKey)
    conn.Write(append([]byte("HOSTKEY:"), hostKeyBytes...))
    
    // Receive user auth
    authData := make([]byte, 1024)
    n, _ := conn.Read(authData)
    
    return s.authenticateUser(authData[:n])
}
```

**Protocol Flow:**
1. TCP connection established
2. Server: `"JCAT/2.0.0+SSH\n"`
3. Server: `"HOSTKEY:" + base64(ed25519_public_key) + "\n"`
4. Client verifies host key against known_hosts file
5. Client: `"USER:" + username + "\n"`
6. Server: `"AUTH_METHOD:password\n"` or `"AUTH_METHOD:publickey\n"`
7. Password auth: Client sends `"PASSWORD:" + password + "\n"`
8. OR Public key auth: Challenge-response with user's key
9. Server: `"AUTH_OK\n"` or `"AUTH_FAIL\n"`
10. Derive session keys from shared secret established during auth
11. Encrypted communication using derived keys

**Configuration:**
```yaml
security:
  method: "ssh_style"
  host_key_file: "/path/to/host_key"
  known_hosts_file: "/path/to/known_hosts"
  user_auth: "password"  # or "publickey" or "both"
  user_database: "/path/to/users.db"
```

**Pros:**
- ✅ **Familiar Model**: SSH-like experience for users
- ✅ **Host Verification**: Prevents man-in-the-middle attacks
- ✅ **Flexible User Auth**: Password or public key authentication
- ✅ **Trust on First Use**: Standard known_hosts workflow
- ✅ **Strong Host Identity**: Ed25519 host keys
- ✅ **User Accountability**: Per-user authentication and logging
- ✅ **Key Management**: Standard SSH key workflows

**Cons:**
- ❌ **Implementation Complexity**: More complex than password-only
- ❌ **Key Management**: Host key distribution and verification
- ❌ **User Experience**: Host key verification prompts
- ❌ **Storage Requirements**: Known hosts, user keys, etc.
- ❌ **Initial Setup**: Generate host keys, set up user database

**Use Cases:**
- SSH-familiar environments
- Need for strong host authentication
- User accountability requirements
- Mixed authentication methods (password + keys)

**Security Level:** ⭐⭐⭐⭐⭐ (Excellent)

---

### ⚡ Method 5: Simple Password (No Encryption)

**Overview:**
Adds basic password authentication to existing protocol without encryption. Provides access control but no confidentiality.

**Technical Implementation:**
```go
func simplePasswordAuth(conn net.Conn, expectedPassword string) error {
    conn.Write([]byte("PASSWORD_REQUIRED\n"))
    
    buffer := make([]byte, 256)
    n, err := conn.Read(buffer)
    if err != nil {
        return err
    }
    
    providedPassword := strings.TrimSpace(string(buffer[:n]))
    if providedPassword != expectedPassword {
        conn.Write([]byte("AUTH_FAIL\n"))
        return errors.New("authentication failed")
    }
    
    conn.Write([]byte("AUTH_OK\n"))
    return nil
}
```

**Protocol Flow:**
1. TCP connection established
2. Server: `"JCAT/2.0.0+AUTH\n"`
3. Server: `"PASSWORD_REQUIRED\n"`
4. Client: `password + "\n"`
5. Server: `"AUTH_OK\n"` or `"AUTH_FAIL\n"`
6. If OK: Continue with standard unencrypted jcat protocol

**Configuration:**
```yaml
security:
  method: "simple_password"
  password: "session-password"
```

**Pros:**
- ✅ **Very Simple**: Minimal code changes required
- ✅ **Fast Implementation**: Can be added quickly
- ✅ **Low Overhead**: No encryption performance impact
- ✅ **Easy Configuration**: Single password setting
- ✅ **Debugging Friendly**: Traffic remains readable

**Cons:**
- ❌ **No Encryption**: All data transmitted in plaintext
- ❌ **Password Exposure**: Password sent over network
- ❌ **Eavesdropping**: Network sniffers can read everything
- ❌ **Man-in-the-Middle**: No protection against MITM attacks
- ❌ **Not Production Ready**: Suitable only for testing/development

**Use Cases:**
- Development and testing environments
- Local network with trusted infrastructure
- Quick prototyping of authentication
- Educational purposes

**Security Level:** ⭐⭐ (Basic)

## Implementation Priority

### Phase 1: Method 3 (Password Authentication + Session Encryption)
- Best balance of security and usability
- Good foundation for future enhancements
- Addresses main security threats

### Phase 2: Method 1 (TLS with Client Certificates)  
- For enterprise environments
- Builds on Phase 1 architecture
- Highest security level

### Phase 3: Method 2 (Pre-Shared Key)
- For simple team environments
- Alternative to password-based auth
- Good performance characteristics

### Future Considerations:
- Method 4: If SSH-style workflow is desired
- Method 5: Only for development/testing

## Configuration Framework

All methods will share a common configuration structure:

```go
type SecurityConfig struct {
    Enabled         bool                   `json:"enabled"`
    Method          string                 `json:"method"`
    GlobalPassword  string                 `json:"global_password,omitempty"`
    SessionPasswords map[string]string     `json:"session_passwords,omitempty"`
    PSK             string                 `json:"psk,omitempty"`
    TLSConfig       *TLSConfig            `json:"tls_config,omitempty"`
    Argon2Params    *Argon2Config         `json:"argon2_params,omitempty"`
}
```

This allows switching between methods and future extensibility.