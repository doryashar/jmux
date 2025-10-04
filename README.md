# jmux - Tmux Session Sharing Made Easy

Enhanced tmux session sharing tool with support for named sessions, multiple invitees, private sessions, and real-time status display.

## GO!
mkdir -p ${HOME}/.local/bin/ && cd ${HOME}/.local/bin/ && curl --silent -L https://github.com/doryashar/jmux/releases/download/v1.1.1/dmux -o dmux && chmod +x ${HOME}/.local/bin/dmux && cd - && \
grep -q 'set path = ( $HOME/.local/bin $path )' $HOME/.tcshrc.user || echo '\nif ( "$path" !~ *"$HOME/.local/bin"* ) set path = ( $HOME/.local/bin $path )' >> $HOME/.tcshrc.user


## Features

- 🚀 **Easy sharing**: Share tmux sessions with simple commands
- 👥 **Multiple invitees**: Invite multiple users at once
- 🏷️ **Named sessions**: Create sessions with custom names
- 🔒 **Private sessions**: Restrict access to specific users
- 📊 **Status display**: Real-time tmux footer showing share status
- 🔍 **Auto-discovery**: Join sessions automatically from invitations
- 💬 **Message system**: Built-in invitation and notification system
- ⚡ **Real-time messaging**: Instant message notifications with non-intrusive display
- 🎯 **Smart notifications**: Priority-based messages with customizable display
- 🔧 **Auto-installation**: Automatically installs tmux AppImage if missing
- 🔗 **Self-setup**: Creates symlinks and PATH entries for easy access

## Directory Structure

```
jmux/
├── bin/                 # Compiled binaries
│   ├── jmux            # Main jmux bash script
│   ├── jmux-go         # Go version (recommended)
│   └── jcat-binary     # Static jcat binary
├── src/                 # Source code
│   ├── jcat/           # jcat Go source
│   └── jmux-go/        # jmux Go implementation
├── scripts/             # Build and utility scripts
├── tests/               # Test scripts
├── docs/                # Documentation
├── Makefile            # Build automation
└── README.md           # This file
```

## Building

```bash
# Build everything (Go version + jcat)
make build

# Build just the Go version
make build-go

# Build just jcat 
make build-jcat

# Clean build artifacts
make clean
```

## Versions

**jmux-go (Recommended)** - Pure Go implementation:
- ✅ **Built-in jcat** - No external dependencies
- ✅ **Live monitoring** - Real-time message overlays  
- ✅ **Static binary** - Single portable executable
- ✅ **Better performance** - Native Go networking
- ✅ **Rich CLI** - Modern command interface with help

**jmux (Bash)** - Original shell script:
- ✅ **Mature** - Stable and well-tested
- ✅ **Flexible** - Easy to modify and extend
- ⚠️ **Dependencies** - Requires socat/jcat binary

## Installation

```bash
# Install to system (installs Go version as default)
make install

# Manual installation
sudo cp bin/jmux-go /usr/local/bin/jmux      # Go version (recommended)
sudo cp bin/jmux /usr/local/bin/jmux-bash    # Bash version  
sudo cp bin/jcat-binary /usr/local/bin/jcat  # jcat binary
```

## Quick Start

### 1. Setup NFS Mount (Required)

First, mount the shared storage required for jmux:

```bash
# Check current mount status
sudo ./mount-projects-common.sh status

# Mount the NFS share
sudo ./mount-projects-common.sh mount

# Setup autofs for automatic mounting (recommended)
sudo ./mount-projects-common.sh autofs
```

### 2. Basic Usage

```bash
# Start regular tmux session (registers your IP)
./jmux

# Share your current/new session
./jmux share

# Join someone's shared session
./jmux join username
```

## Advanced Features

### Named Sessions
```bash
# Create session with custom name
./jmux share --name "code-review" alice bob
```

### Multiple Invitees
```bash
# Invite multiple users
./jmux share alice bob charlie dave
```

### Private Sessions
```bash
# Create private session (only invited users can join)
./jmux share --private alice bob
```

### Real-Time Messaging
```bash
# Send a regular message
./jmux msg alice "Can you help me with the deployment?"

# Send an urgent message (high priority)
./jmux msg bob urgent "Server is down!"

# Manage message watcher
./jmux watch status
./jmux watch start
./jmux watch stop
./jmux watch restart
```

#### Message Features:
- **📨 Instant delivery**: Messages appear immediately at bottom of terminal
- **🎨 Visual indicators**: Different icons for invites, urgent messages, etc.
- **⏰ Auto-hide**: Messages disappear after configurable duration
- **🔧 Non-intrusive**: Doesn't disrupt current work, saves cursor position
- **⚙️ Configurable**: Control display duration and enable/disable real-time

### Flexible Joining Options
```bash
# Join by username
./jmux join alice

# Join specific session by username  
./jmux join dory mysession

# Join by IP address (default port)
./jmux join 192.168.1.100

# Join by IP with specific port
./jmux join 192.168.1.100 12346

# Join by hostname
./jmux join server.example.com

# Join by hostname with port
./jmux join server.example.com 54321

# Auto-join from pending invitation
./jmux join
```

## Commands

| Command | Description |
|---------|-------------|
| `jmux` | Start regular tmux session |
| `jmux share [options] [users...]` | Share session with options |
| `jmux join [user\|hostname\|ip] [session-name\|port]` | Join a shared session |
| `jmux stop` | Stop sharing current session |
| `jmux status` | Show detailed status |
| `jmux sessions` | List all active shared sessions |
| `jmux users` | List connected users in current session |
| `jmux list-users` | List all registered users |
| `jmux messages` | Check for new messages |
| `jmux msg <user> [type] <message>` | Send message to user |
| `jmux watch {start\|stop\|status\|restart}` | Manage real-time message watcher |
| `jmux reset` | Reset terminal settings (fix mouse/keyboard issues) |

## Share Options

- `--name <name>`: Set custom session name
- `--private`: Make session private (only invited users can join)
- `[users...]`: Space-separated list of users to invite

## Status Display

When sharing a session, the tmux footer displays:
```
[SHARED] Join: jmux join username | Connections: 2
```

- Updates every 30 seconds
- Shows exact join command for others
- Displays current connection count
- Automatically clears when sharing stops

## Setup Requirements

### Dependencies
```bash
# Required: socat and NFS tools
# Ubuntu/Debian
sudo apt-get install socat nfs-common autofs inotify-tools

# RHEL/CentOS/Fedora
sudo yum install socat nfs-utils autofs inotify-tools

# Note: tmux auto-installs if missing (downloads AppImage)
# Note: inotify-tools is optional for real-time messaging
# Without it, messaging works with manual checking
```

### NFS Mount
The `/projects/common` NFS share must be mounted for shared storage:
- Server: `x-filer21-100.xsight.ent:/project_common`
- Mount point: `/projects/common`
- User ID: `87002638`, Group ID: `87001524`
- Use the provided `mount-projects-common.sh` script

### Environment Variables
```bash
export JMUX_PORT=12345                                    # Port for sharing
export JMUX_SHARED_DIR=/projects/common/work/dory/jmux    # Shared storage path
export JMUX_REALTIME=true                                 # Enable real-time messaging
export JMUX_NOTIFICATION_DURATION=5                       # Message display duration (seconds)
```

## File Structure

```
jmux/
├── jmux                      # Main script
├── mount-projects-common.sh  # NFS mount helper
├── test_jmux.sh             # Test suite
└── README.md                # This file
```

## Storage Layout

```
$JMUX_SHARED_DIR/jmux/
├── users.db          # User to IP mapping
├── messages/         # Message queue for invitations
└── sessions/         # Active session registry
```

## Testing

Run the test suite to verify functionality:
```bash
./test_jmux.sh
```

## Troubleshooting

### Common Issues

1. **"Shared storage not accessible"**
   - Ensure `/projects/common` is mounted
   - Run: `sudo ./mount-projects-common.sh status`

2. **"Port already in use"**
   - jmux will automatically find available ports
   - Set custom port: `export JMUX_PORT=54321`

3. **"User not found"**
   - Users must run `jmux` at least once to register
   - Check registered users: `jmux list-users`

4. **Permission issues**
   - Ensure write access to `$JMUX_SHARED_DIR/jmux/`
   - Check NFS mount permissions

5. **Terminal broken after session (mouse sends keystrokes)**
   - Run: `jmux reset`
   - Or manually: `stty sane && reset`
   - This fixes terminal state corruption from socat connections

6. **Real-time messages not working**
   - Install inotify-tools: `sudo apt-get install inotify-tools`
   - Check watcher status: `jmux watch status`
   - Restart watcher: `jmux watch restart`
   - Disable if needed: `export JMUX_REALTIME=false`

7. **Messages appearing in wrong location**
   - Terminal may not support cursor save/restore
   - Try resizing terminal or using different terminal emulator
   - Check `$TERM` environment variable

### Debug Mode
Enable debug output:
```bash
set -x
./jmux share alice
```

## Security Notes

- Sessions are shared over unencrypted connections
- Private sessions restrict access but don't encrypt data
- Use only on trusted networks
- Regular cleanup of old session files recommended

## License

This project is provided as-is for internal use.

# TODO: 
[ ]   need to validate the join-session permissions, how is it handled?
[ ]   add security (password / encryption / pem etc)
[ ]   use different version of socat to allow sigwitch? https://github.com/StalkR/misc/tree/master/pty
[ ]   add "ask for share" that is doing reverse sharing
[ ]   live monitor without inotify
[ ]   every share should also show (print to the host/listener session) the share command that the client should run to connect (if there are more than one sessions, the session name is preferred to be shown)
      if not using a shared directory - you should display the full ip:port in your join command.
[ ]   add request-for-share command which will start a reverse port listen so that if a client connects to that port, the client will share his session. 
      jcat should be enhanced for this

Assistant mode (now)
View only mode
Pair programming mode
Switch modes

Kick / Ban user
Password / ssh key / encryption

Can we pass all keystrokes but client special keys?