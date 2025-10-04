# SetSize Script in dmux

The setsize script is a critical component of dmux (and jmux) that handles terminal size synchronization and tmux session attachment when clients connect to shared sessions.

## Overview

The setsize script (`~/.config/jmux/setsize.sh`) is automatically created and maintained by dmux. It serves as the entry point for connecting clients and handles:

1. **Session Name Detection**: Determines which tmux session to connect to based on port mapping
2. **Terminal Size Synchronization**: Ensures proper terminal dimensions for remote connections
3. **tmux Path Resolution**: Finds tmux executable in various common locations
4. **Environment Setup**: Sources jmux profile for proper PATH and tool availability

## How It Works

### 1. Script Creation

The setsize script is automatically created during dmux initialization when:
- The script doesn't exist in `~/.config/jmux/setsize.sh`
- The script exists but is outdated (missing profile sourcing functionality)

### 2. Session Name Resolution

When a client connects, the script determines the session name through:

```bash
# Priority 1: Port mapping from SOCAT_SOCKPORT environment variable
if [[ -n "${SOCAT_SOCKPORT:-}" ]]; then
    PORT_MAP_FILE="${JMUX_SHARED_DIR:-/projects/common/work/dory/jmux}/port_sessions.db"
    SESSION_NAME=$(grep "^${SOCAT_SOCKPORT}:" "$PORT_MAP_FILE" | head -1 | cut -d: -f3)
fi

# Priority 2: Hostname fallback
if [[ -z "$SESSION_NAME" ]]; then
    SESSION_NAME="${HOSTNAME:-$(hostname 2>/dev/null || echo "jmux-session")}"
fi

# Priority 3: Safe fallback
if [[ -z "$SESSION_NAME" ]]; then
    SESSION_NAME="jmux-fallback-session"
fi
```

### 3. tmux Executable Discovery

The script tries multiple locations for tmux:
1. `command -v tmux` (in PATH)
2. `/bin/tmux`
3. `/usr/bin/tmux`
4. `$HOME/.local/bin/tmux`

### 4. Integration with Socket Servers

The setsize script is used by:

#### jcat Server (Primary)
```bash
JCAT_SETSIZE_SCRIPT="$setsize_script" jcat -server -listen ":$port"
```

#### socat Fallback
```bash
socat TCP-LISTEN:${port},fork EXEC:"bash --rcfile ${setsize_script}",pty,stderr,setsid,sigint,sane
```

## Environment Variables

The script respects these environment variables:

- **SOCAT_SOCKPORT**: Port number used to look up session name in port mapping
- **SOCAT_PEERADDR**: Client IP address (set by socat/jcat)
- **SOCAT_PEERPORT**: Client port (set by socat/jcat)
- **JMUX_SHARED_DIR**: Path to shared storage directory
- **HOSTNAME**: System hostname for session naming

## Port Mapping Database

The script reads from `port_sessions.db` with format:
```
PORT:USER:SESSION_NAME
12345:alice:alice-session
12346:bob:project-dory
```

## Profile Integration

The script sources `~/.config/jmux/profile.sh` which:
- Adds `~/.local/bin` to PATH
- Makes jmux available if installed locally
- Sets up environment for tmux sessions

## Script Updates

The script is automatically updated when:
- dmux detects the script is missing the profile sourcing comment
- The script becomes corrupted or invalid
- Major version updates require script changes

## Security Considerations

- Script is created with 755 permissions (executable by owner, readable by all)
- Only sources trusted profile script from user's config directory
- Uses safe fallbacks for all environment variables
- Validates session names before use

## Troubleshooting

### Script Not Found
If the setsize script is missing, run any dmux command to recreate it:
```bash
dmux sessions
```

### Session Name Issues
Check the port mapping database:
```bash
cat /projects/common/work/dory/jmux/port_sessions.db
```

### tmux Not Found
The script will show detailed error with paths tried:
```
Error: tmux not found in any common location
Available paths:
  PATH: /usr/local/bin:/usr/bin:/bin
Tried:
  - tmux (in PATH)
  - /bin/tmux
  - /usr/bin/tmux
  - /home/user/.local/bin/tmux
```

### Profile Issues
Check that profile script exists and is readable:
```bash
ls -la ~/.config/jmux/profile.sh
cat ~/.config/jmux/profile.sh
```

## Implementation Details

### Go Implementation

The setsize script creation is handled in `internal/config/config.go`:

```go
// EnsureSetSizeScript creates or updates the setsize script if needed
func (c *Config) EnsureSetSizeScript() error {
    // Check if script needs update
    // Create setsize.sh and profile.sh
    // Set proper permissions
}
```

### Template Content

The script template is embedded in the Go code and includes:
- Shebang line
- Profile sourcing
- Session name resolution logic
- tmux path discovery
- Error handling

## Related Files

- `~/.config/jmux/setsize.sh` - Main setsize script
- `~/.config/jmux/profile.sh` - Profile script for environment setup
- `${JMUX_SHARED_DIR}/port_sessions.db` - Port to session mapping
- `internal/config/config.go` - Go implementation for script creation
- `internal/jcat/jcat.go` - Integration with jcat server

## Compatibility

The setsize script maintains compatibility with:
- Original bash jmux implementation
- socat-based sharing (fallback mode)
- jcat-based sharing (primary mode)
- Multiple tmux installation locations
- Various Linux distributions and shell environments