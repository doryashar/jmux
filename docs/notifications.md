# dmux Notification System

dmux provides multiple notification methods to display messages and invitations across different desktop environments.

## Notification Methods

### 1. Zenity (Default)
**GTK-based visual dialogs** - Best user experience for desktop users

- **Interactive dialogs** with proper buttons and actions
- **Message-specific types**: Question, Warning, Info
- **Auto-close timeouts** to prevent desktop clutter
- **Terminal integration** for invitation acceptance

**Features:**
- Invitation dialogs with "Join Session" / "Dismiss" buttons
- Urgent messages use warning style with 10-second timeout
- Regular messages use info style with 8-second timeout
- Clicking "Join Session" automatically opens terminal to join

### 2. KDialog (KDE)
**KDE desktop integration** - Native KDE notifications

- Modal dialogs with KDE styling
- Window attachment for proper focus
- Geometry positioning for visibility
- Error/question dialog types for different message priorities

### 3. Notify-send (Freedesktop)
**System tray notifications** - Lightweight and unobtrusive

- Standard freedesktop notifications
- Urgency levels (normal, critical)
- 5-second display duration
- Mail icon for message identification

### 4. Tmux (In-session)
**Terminal-based messages** - For tmux session users

- `tmux display-message` integration
- Shows messages directly in tmux status
- 5-second display duration
- Works within existing tmux sessions

### 5. Terminal (Fallback)
**Direct terminal output** - Always available fallback

- Colored text output with message formatting
- Real-time display with timestamps
- Works in any terminal environment
- Last resort when no GUI available

## Auto-Detection Priority

dmux automatically detects available notification methods in this order:

1. **zenity** → Interactive GTK dialogs (preferred)
2. **kdialog** → KDE desktop integration
3. **notify-send** → System tray notifications
4. **tmux** → In-session terminal messages
5. **terminal** → Direct text output (fallback)

## Configuration

### Environment Variables

```bash
# Force specific notification method
export DMUX_MESSAGE_DISPLAY=zenity
export DMUX_MESSAGE_DISPLAY=kdialog
export DMUX_MESSAGE_DISPLAY=notify
export DMUX_MESSAGE_DISPLAY=tmux
export DMUX_MESSAGE_DISPLAY=terminal

# Auto-detect best method (default)
export DMUX_MESSAGE_DISPLAY=auto

# Enable debug information
export DMUX_DEBUG=1
```

### Message Types

**Invitations**
- Zenity: Question dialog with Join/Dismiss buttons
- KDialog: Yes/No dialog with terminal launch
- Notify-send: High priority notification
- Tmux: Highlighted session invitation
- Terminal: Colored invitation text

**Urgent Messages**
- Zenity: Warning dialog with 10s timeout
- KDialog: Error dialog with attention
- Notify-send: Critical urgency notification
- Tmux: Urgent-style display message
- Terminal: Red urgent text

**Regular Messages**
- Zenity: Info dialog with 8s timeout
- KDialog: Standard message box
- Notify-send: Normal priority notification
- Tmux: Regular display message
- Terminal: Standard colored text

## Usage Examples

### Explicit Configuration
```bash
# Use zenity for all notifications
DMUX_MESSAGE_DISPLAY=zenity dmux monitor start

# Use notify-send for lightweight notifications
DMUX_MESSAGE_DISPLAY=notify dmux messages

# Debug notification method selection
DMUX_DEBUG=1 DMUX_MESSAGE_DISPLAY=auto dmux messages
```

### Testing Notifications
```bash
# Test zenity notifications
./tests/test_zenity_notifications.sh

# Test all notification methods
./tests/test_notification_methods.sh
```

## Desktop Integration

### Zenity Features
- **Visual Appeal**: Native GTK styling matches desktop theme
- **Interaction**: Click buttons to take action on invitations
- **Smart Timeouts**: Auto-close to prevent accumulation
- **Terminal Launch**: Automatic terminal opening for session joining

### Session Invitation Workflow
1. User receives invitation notification
2. Zenity shows question dialog with session details
3. User clicks "Join Session" button
4. dmux automatically opens terminal with `dmux join <user>`
5. User is connected to the shared session

## Troubleshooting

### Zenity Not Working
```bash
# Check if zenity is installed
which zenity

# Install zenity (Ubuntu/Debian)
sudo apt install zenity

# Install zenity (Fedora)
sudo dnf install zenity
```

### Debug Mode
```bash
# Enable debug output to see notification method selection
export DMUX_DEBUG=1
dmux messages
```

### Fallback Testing
```bash
# Test what happens when specific methods aren't available
PATH=/usr/bin dmux messages  # Remove zenity from PATH
```

## Best Practices

1. **Use auto-detection** - Let dmux choose the best method
2. **Install zenity** - Provides the best user experience
3. **Test notifications** - Ensure they work in your environment
4. **Configure timeouts** - Adjust for your workflow needs
5. **Enable debug mode** - When troubleshooting notification issues

## Installation Requirements

**For Zenity (Recommended)**
```bash
# Ubuntu/Debian
sudo apt install zenity

# Fedora/RHEL
sudo dnf install zenity

# Arch Linux
sudo pacman -S zenity
```

**For KDialog (KDE)**
```bash
# Usually included with KDE desktop
sudo apt install kdialog  # Ubuntu/Debian
```

**For Notify-send**
```bash
# Usually included with desktop environments
sudo apt install libnotify-bin  # Ubuntu/Debian
```

The notification system automatically adapts to your desktop environment, providing the best available user experience while maintaining compatibility across different systems.