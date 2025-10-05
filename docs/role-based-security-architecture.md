# Role-Based Security Architecture for dmux

## Overview
This document outlines multiple approaches for implementing secure role-based commands in dmux, focusing on preventing clients from accessing host-only functionality.

## Security Challenge
The primary concern is preventing clients from executing host commands like:
```bash
dmux host-menu
dmux kick user
dmux ban user
```

## Approach 1: Multi-Layer Client-Side Verification

### Architecture
```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ Tmux Key Binding│───▶│ Security Layers  │───▶│ Command Execution│
│ Ctrl+A + H      │    │ (5 Verifications)│    │ Host Menu       │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Security Layers
1. **Session Ownership Verification**
2. **Process Origin Validation** 
3. **Tmux Context Verification**
4. **Time-Based Command Validation**
5. **Token-Based Authentication**

### Implementation
```go
type SecurityManager struct {
    sessionMgr   *session.Manager
    tokenStore   map[string]SessionToken
    rateLimiter  *RateLimiter
    auditLogger  *AuditLogger
}

func (s *SecurityManager) VerifyHostAccess(sessionName string) error {
    // Layer 1: Ownership check
    if !s.isSessionOwner(sessionName) {
        return ErrNotOwner
    }
    
    // Layer 2: Process origin
    if !s.verifyTmuxOrigin() {
        return ErrInvalidOrigin
    }
    
    // Layer 3: Context verification
    if !s.verifyTmuxContext(sessionName) {
        return ErrContextMismatch
    }
    
    // Layer 4: Time validation
    if !s.validateCommandTiming() {
        return ErrCommandTimeout
    }
    
    // Layer 5: Token validation
    if !s.validateToken(sessionName) {
        return ErrInvalidToken
    }
    
    return nil
}
```

### Pros
- No server modifications required
- Multiple security layers
- Comprehensive audit logging

### Cons
- Can be bypassed with sufficient effort
- Relies on client-side enforcement
- Complex verification logic

---

## Approach 2: Server-Side Command Filtering

### Architecture
```
┌─────────────┐    ┌─────────────────┐    ┌──────────────┐
│ Client Input│───▶│ jcat Server     │───▶│ Command Filter│
│ Ctrl+` + H  │    │ Key Interceptor │    │ Role-Based   │
└─────────────┘    └─────────────────┘    └──────────────┘
                            │
                            ▼
                   ┌─────────────────┐
                   │ Host Menu       │
                   │ (Server-Side)   │
                   └─────────────────┘
```

### Key Interception Protocol
```go
// Special key sequences intercepted by server
const (
    HOST_MENU_KEY    = "\x1b[H"  // Ctrl+` + H
    CLIENT_MENU_KEY  = "\x1b[C"  // Ctrl+` + C
    GLOBAL_MENU_KEY  = "\x1b[M"  // Ctrl+` + M
)

type KeyInterceptor struct {
    sessionRoles map[string]string
    menuHandlers map[string]MenuHandler
}

func (k *KeyInterceptor) ProcessInput(data []byte, clientID string) []byte {
    keySeq := string(data)
    
    switch keySeq {
    case HOST_MENU_KEY:
        if k.isHost(clientID) {
            k.showHostMenu(clientID)
            return nil // Don't forward to tmux
        }
        // Ignore for non-hosts
        return nil
        
    case CLIENT_MENU_KEY:
        k.showClientMenu(clientID)
        return nil
        
    default:
        return data // Forward to tmux
    }
}
```

### Server-Side Menu System
```go
type ServerMenu struct {
    title    string
    options  []MenuOption
    handler  func(option string, clientID string) error
}

func (s *Server) ShowHostMenu(clientID string) {
    menu := ServerMenu{
        title: "Host Management Menu",
        options: []MenuOption{
            {Key: "1", Label: "Kick User", Handler: s.kickUser},
            {Key: "2", Label: "Ban User", Handler: s.banUser},
            {Key: "3", Label: "List Users", Handler: s.listUsers},
            {Key: "4", Label: "Change Mode", Handler: s.changeMode},
            {Key: "q", Label: "Quit", Handler: nil},
        },
    }
    
    s.renderMenu(clientID, menu)
}
```

### Pros
- Server-side enforcement (more secure)
- No client can bypass restrictions
- Clean key sequence handling
- Real-time menu updates

### Cons
- Requires jcat protocol modifications
- More complex server implementation
- Custom key sequences may conflict

---

## Approach 3: Capability-Based Authentication

### Architecture
```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ Session Start   │───▶│ Capability Token │───▶│ Command Execution│
│ Role Assignment │    │ Generation       │    │ Capability Check│
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Capability System
```go
type Capability struct {
    Action      string    `json:"action"`
    Resource    string    `json:"resource"`
    Scope       string    `json:"scope"`
    ExpiresAt   int64     `json:"expires_at"`
    Signature   string    `json:"signature"`
}

type UserCapabilities struct {
    UserID       string       `json:"user_id"`
    SessionID    string       `json:"session_id"`
    Capabilities []Capability `json:"capabilities"`
    IssuedAt     int64        `json:"issued_at"`
}

// Host capabilities
var HostCapabilities = []Capability{
    {Action: "kick", Resource: "user", Scope: "session"},
    {Action: "ban", Resource: "user", Scope: "session"},
    {Action: "modify", Resource: "session", Scope: "owned"},
    {Action: "transfer", Resource: "ownership", Scope: "session"},
}

// Client capabilities  
var ClientCapabilities = []Capability{
    {Action: "read", Resource: "session", Scope: "joined"},
    {Action: "modify", Resource: "self", Scope: "profile"},
    {Action: "leave", Resource: "session", Scope: "joined"},
}
```

### Implementation
```go
func (c *CapabilityManager) GrantCapabilities(userID, sessionID, role string) (*UserCapabilities, error) {
    var caps []Capability
    
    switch role {
    case "host":
        caps = HostCapabilities
    case "client":
        caps = ClientCapabilities
    default:
        return nil, ErrInvalidRole
    }
    
    // Sign capabilities with session key
    for i := range caps {
        caps[i].ExpiresAt = time.Now().Add(24 * time.Hour).Unix()
        caps[i].Signature = c.signCapability(caps[i], sessionID)
    }
    
    return &UserCapabilities{
        UserID:       userID,
        SessionID:    sessionID,
        Capabilities: caps,
        IssuedAt:     time.Now().Unix(),
    }, nil
}

func (c *CapabilityManager) CheckCapability(userCaps *UserCapabilities, action, resource string) bool {
    for _, cap := range userCaps.Capabilities {
        if cap.Action == action && cap.Resource == resource {
            if time.Now().Unix() < cap.ExpiresAt {
                return c.verifySignature(cap, userCaps.SessionID)
            }
        }
    }
    return false
}
```

### Pros
- Fine-grained permission control
- Cryptographically secure
- Extensible capability system
- Clear audit trail

### Cons
- Complex implementation
- Overhead of capability management
- Key distribution challenges

---

## Approach 4: Hardware Security Module (HSM) Integration

### Architecture
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ User Authentication│───▶│ HSM Token       │───▶│ Secure Command  │
│ (SSH Key/Cert)  │    │ Generation      │    │ Execution       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Implementation
```go
type HSMManager struct {
    keyStore    map[string]*rsa.PublicKey
    tokenStore  map[string]SecureToken
}

type SecureToken struct {
    UserID      string    `json:"user_id"`
    SessionID   string    `json:"session_id"`
    Role        string    `json:"role"`
    IssuedAt    int64     `json:"issued_at"`
    ExpiresAt   int64     `json:"expires_at"`
    HMACDigest  string    `json:"hmac_digest"`
}

func (h *HSMManager) AuthenticateUser(userID, sessionID string, sshKeyPath string) (*SecureToken, error) {
    // Verify SSH key authenticity
    pubKey, err := h.loadSSHPublicKey(sshKeyPath)
    if err != nil {
        return nil, err
    }
    
    // Check if user is authorized for this session
    role, err := h.determineUserRole(userID, sessionID, pubKey)
    if err != nil {
        return nil, err
    }
    
    // Generate secure token with HSM
    token := SecureToken{
        UserID:    userID,
        SessionID: sessionID,
        Role:      role,
        IssuedAt:  time.Now().Unix(),
        ExpiresAt: time.Now().Add(time.Hour).Unix(),
    }
    
    // Sign with HSM
    token.HMACDigest = h.generateHMAC(token)
    
    return &token, nil
}
```

### Pros
- Hardware-level security
- Non-repudiation
- Strong authentication
- Enterprise-grade security

### Cons
- Complex setup requirements
- Hardware dependencies
- Significant overhead
- May be overkill for most use cases

---

## Approach 5: Blockchain-Based Consensus

### Architecture
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Command Request │───▶│ Network Vote    │───▶│ Consensus Check │
│ (Host Action)   │    │ (All Clients)   │    │ Command Execute │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Implementation
```go
type CommandProposal struct {
    ID          string    `json:"id"`
    Proposer    string    `json:"proposer"`
    Command     string    `json:"command"`
    Arguments   []string  `json:"arguments"`
    Timestamp   int64     `json:"timestamp"`
    Signatures  map[string]string `json:"signatures"`
}

type ConsensusManager struct {
    proposals   map[string]*CommandProposal
    validators  map[string]*ValidatorInfo
    threshold   float64 // e.g., 0.67 for 2/3 majority
}

func (c *ConsensusManager) ProposeCommand(proposer, command string, args []string) (*CommandProposal, error) {
    proposal := &CommandProposal{
        ID:        generateID(),
        Proposer:  proposer,
        Command:   command,
        Arguments: args,
        Timestamp: time.Now().Unix(),
        Signatures: make(map[string]string),
    }
    
    // Host automatically votes for their own proposals
    if c.isHost(proposer) {
        proposal.Signatures[proposer] = c.signProposal(proposal, proposer)
    }
    
    c.proposals[proposal.ID] = proposal
    c.broadcastProposal(proposal)
    
    return proposal, nil
}

func (c *ConsensusManager) VoteOnProposal(proposalID, voterID string, approve bool) error {
    proposal, exists := c.proposals[proposalID]
    if !exists {
        return ErrProposalNotFound
    }
    
    if approve {
        proposal.Signatures[voterID] = c.signProposal(proposal, voterID)
    }
    
    // Check if consensus reached
    if c.hasConsensus(proposal) {
        return c.executeCommand(proposal)
    }
    
    return nil
}
```

### Pros
- Democratic decision making
- Transparent process
- Tamper-resistant
- Innovative approach

### Cons
- Extremely complex
- Performance overhead
- May not be suitable for real-time commands
- Consensus delays

---

## Approach 6: Hybrid Server-Client Validation

### Architecture
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Client Request  │───▶│ Server Validation│───▶│ Client Execution│
│ (With Proof)    │    │ (Role Check)    │    │ (If Authorized) │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                               │
                               ▼
                      ┌─────────────────┐
                      │ Audit & Logging │
                      └─────────────────┘
```

### Protocol Implementation
```go
type CommandRequest struct {
    Command     string                 `json:"command"`
    Arguments   []string               `json:"arguments"`
    UserID      string                 `json:"user_id"`
    SessionID   string                 `json:"session_id"`
    Timestamp   int64                  `json:"timestamp"`
    Proof       AuthenticationProof    `json:"proof"`
}

type AuthenticationProof struct {
    Type        string `json:"type"`        // "tmux_context", "process_tree", etc.
    Evidence    string `json:"evidence"`    // Actual proof data
    Signature   string `json:"signature"`   // Client signature
}

type CommandResponse struct {
    Authorized  bool   `json:"authorized"`
    Reason      string `json:"reason,omitempty"`
    Challenge   string `json:"challenge,omitempty"`
    Token       string `json:"token,omitempty"`
}

func (s *Server) ProcessCommandRequest(req *CommandRequest) *CommandResponse {
    // 1. Validate basic request structure
    if err := s.validateRequest(req); err != nil {
        return &CommandResponse{
            Authorized: false,
            Reason: fmt.Sprintf("Invalid request: %v", err),
        }
    }
    
    // 2. Check user role in session
    role, err := s.getUserRole(req.UserID, req.SessionID)
    if err != nil {
        return &CommandResponse{
            Authorized: false,
            Reason: "Cannot determine user role",
        }
    }
    
    // 3. Verify command permissions
    if !s.commandAllowed(req.Command, role) {
        s.logSecurityViolation(req.UserID, req.Command, "Insufficient permissions")
        return &CommandResponse{
            Authorized: false,
            Reason: "Insufficient permissions",
        }
    }
    
    // 4. Validate proof of context
    if !s.validateProof(req.Proof, req.UserID, req.SessionID) {
        s.logSecurityViolation(req.UserID, req.Command, "Invalid authentication proof")
        return &CommandResponse{
            Authorized: false,
            Reason: "Authentication proof invalid",
        }
    }
    
    // 5. Generate execution token
    token, err := s.generateExecutionToken(req)
    if err != nil {
        return &CommandResponse{
            Authorized: false,
            Reason: "Token generation failed",
        }
    }
    
    s.logCommandAuthorization(req.UserID, req.Command, true)
    
    return &CommandResponse{
        Authorized: true,
        Token:      token,
    }
}
```

### Client-Side Implementation
```go
func (c *Client) ExecuteSecureCommand(command string, args []string) error {
    // 1. Gather authentication proof
    proof, err := c.gatherAuthProof()
    if err != nil {
        return fmt.Errorf("failed to gather auth proof: %v", err)
    }
    
    // 2. Create command request
    req := &CommandRequest{
        Command:   command,
        Arguments: args,
        UserID:    c.userID,
        SessionID: c.sessionID,
        Timestamp: time.Now().Unix(),
        Proof:     proof,
    }
    
    // 3. Request authorization from server
    resp, err := c.requestAuthorization(req)
    if err != nil {
        return fmt.Errorf("authorization request failed: %v", err)
    }
    
    if !resp.Authorized {
        return fmt.Errorf("command not authorized: %s", resp.Reason)
    }
    
    // 4. Execute command with token
    return c.executeWithToken(command, args, resp.Token)
}

func (c *Client) gatherAuthProof() (AuthenticationProof, error) {
    // Gather multiple pieces of evidence
    evidence := map[string]string{
        "tmux_session": c.getCurrentTmuxSession(),
        "process_tree": c.getProcessTree(),
        "environment":  c.getEnvironmentHash(),
        "timing":       fmt.Sprintf("%d", time.Now().Unix()),
    }
    
    evidenceJSON, _ := json.Marshal(evidence)
    signature := c.signData(evidenceJSON)
    
    return AuthenticationProof{
        Type:      "multi_factor",
        Evidence:  string(evidenceJSON),
        Signature: signature,
    }, nil
}
```

### Pros
- Server validates all commands
- Client provides proof of legitimacy
- Comprehensive audit trail
- Flexible proof mechanisms

### Cons
- Network round-trip for each command
- Complex proof generation
- Potential performance impact

---

## Approach 7: Hardware Token Integration

### Architecture
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Hardware Token  │───▶│ OTP Generation  │───▶│ Command         │
│ (YubiKey/etc)   │    │ & Validation    │    │ Authorization   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Implementation
```go
type HardwareTokenManager struct {
    tokenSerial  string
    secretKey    []byte
    lastCounter  uint64
}

func (h *HardwareTokenManager) GenerateOTP() (string, error) {
    // Generate HOTP (HMAC-based OTP)
    counter := h.getNextCounter()
    hash := hmac.New(sha1.New, h.secretKey)
    
    counterBytes := make([]byte, 8)
    binary.BigEndian.PutUint64(counterBytes, counter)
    hash.Write(counterBytes)
    
    hmacResult := hash.Sum(nil)
    offset := hmacResult[19] & 0x0f
    
    truncated := binary.BigEndian.Uint32(hmacResult[offset:offset+4]) & 0x7fffffff
    otp := truncated % 1000000
    
    return fmt.Sprintf("%06d", otp), nil
}

func (s *Server) ValidateHardwareToken(userID, sessionID, otp string) bool {
    tokenInfo, exists := s.userTokens[userID]
    if !exists {
        return false
    }
    
    // Validate OTP with time window
    for i := -2; i <= 2; i++ {
        expectedOTP := tokenInfo.generateOTPForCounter(tokenInfo.lastCounter + uint64(i))
        if otp == expectedOTP {
            tokenInfo.lastCounter = tokenInfo.lastCounter + uint64(i)
            return true
        }
    }
    
    return false
}

// Host menu with hardware token
func showHostMenuWithToken() error {
    // Request OTP from user
    fmt.Print("Enter hardware token OTP: ")
    otp, _ := bufio.NewReader(os.Stdin).ReadString('\n')
    otp = strings.TrimSpace(otp)
    
    // Validate with server
    if !validateOTP(getCurrentUser(), getCurrentSession(), otp) {
        return fmt.Errorf("invalid hardware token")
    }
    
    // Show menu
    return showHostMenu()
}
```

### Pros
- Hardware-based security
- Difficult to bypass
- Industry standard
- Two-factor authentication

### Cons
- Requires physical tokens
- Additional hardware cost
- User convenience impact
- Setup complexity

---

## Recommendation Matrix

| Approach | Security Level | Implementation Complexity | Performance Impact | User Experience | Cost |
|----------|---------------|---------------------------|-------------------|------------------|------|
| Multi-Layer Client | Medium | Medium | Low | Good | Low |
| Server-Side Filtering | High | High | Medium | Excellent | Low |
| Capability-Based | High | High | Medium | Good | Low |
| HSM Integration | Very High | Very High | High | Medium | High |
| Blockchain Consensus | High | Very High | Very High | Poor | Medium |
| Hybrid Validation | High | High | Medium | Good | Low |
| Hardware Token | Very High | Medium | Low | Medium | Medium |

## Recommended Implementation Plan

### Phase 1: Server-Side Key Filtering (Approach 2)
- Implement special key sequence interception in jcat server
- Add server-side menu system
- Role-based command filtering

### Phase 2: Multi-Layer Validation (Approach 1)
- Add comprehensive client-side verification
- Implement audit logging
- Rate limiting and monitoring

### Phase 3: Capability Enhancement (Approach 3)
- Implement fine-grained capability system
- Cryptographic token validation
- Advanced permission management

### Future Considerations
- Hardware token integration for high-security environments
- HSM integration for enterprise deployments
- Hybrid validation for additional security layers

## Conclusion

The recommended approach combines **Server-Side Key Filtering** with **Multi-Layer Client Validation** to provide both strong security and good user experience. This hybrid approach ensures commands are filtered at the protocol level while maintaining comprehensive verification on the client side.

The implementation should start with server-side filtering as the primary security mechanism, then add additional validation layers for defense in depth.