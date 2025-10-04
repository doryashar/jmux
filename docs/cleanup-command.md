# dmux cleanup Command

The `dmux cleanup` command provides comprehensive system maintenance for dmux installations, including terminal restoration, process cleanup, and session file management.

## Overview

The cleanup command addresses common issues that can occur when using dmux:
- **Terminal corruption** from interrupted sessions or binary data
- **Orphaned processes** from improperly terminated dmux sessions
- **Stale session files** from dead or disconnected sessions

## Usage

### Basic Usage
```bash
# Clean up everything (recommended)
dmux cleanup

# Fix terminal issues only
dmux cleanup --terminal

# Kill orphaned processes only  
dmux cleanup --processes

# Clean session files only
dmux cleanup --sessions
```

## Cleanup Types

### 🔧 Terminal Cleanup (`--terminal`)

Fixes terminal corruption issues that can occur after:
- Interrupted tmux sessions
- Binary data displayed in terminal
- Corrupted terminal state

**What it does:**
- Saves current terminal settings
- Runs `stty sane` to restore sane terminal settings
- Runs `reset` to completely reset the terminal
- Restores normal terminal functionality

**Example:**
```bash
# Terminal corrupted after cat'ing a binary file
$ cat /bin/ls
[terminal shows garbage characters]

# Fix it with dmux cleanup
$ dmux cleanup --terminal
🔧 Fixing terminal settings...
🔄 Resetting terminal...
✓ Applied sane terminal settings
✓ Terminal reset complete
```

### 🧹 Process Cleanup (`--processes`)

Removes orphaned dmux-related processes that may be consuming resources.

**Targets these process types:**
- `_internal_jcat_server` - dmux jcat server processes
- `jcat.*client` - jcat client processes
- `socat.*dmux` - socat processes related to dmux

**What it does:**
- Scans for orphaned dmux processes using `pgrep`
- Terminates processes using `kill`
- Reports the number of processes cleaned up

**Example:**
```bash
$ dmux cleanup --processes
🧹 Cleaning up orphaned processes...
🔫 Killing orphaned jcat server process (PID: 12345)
🔫 Killing orphaned socat process (PID: 12346)
✓ Killed 2 orphaned process(es)
```

### 📁 Session Cleanup (`--sessions`)

Removes stale session files for sessions that are no longer active.

**What it does:**
- Scans session files in the shared directory
- Checks if associated processes are still running
- Verifies if ports are still in use
- Removes files for dead sessions

**Example:**
```bash
$ dmux cleanup --sessions
📁 Cleaning up stale session files...
✓ Cleaned up 3 stale session(s)
```

## Default Behavior

When run without flags, `dmux cleanup` performs all cleanup types:

```bash
$ dmux cleanup
🔧 Fixing terminal settings...
✓ Applied sane terminal settings
✓ Terminal reset complete
🧹 Cleaning up orphaned processes...
✓ No orphaned processes found
📁 Cleaning up stale session files...
✓ Cleaned up 1 stale session(s)

🎉 Cleanup complete: 1 items cleaned
```

## Common Use Cases

### 1. Terminal Corruption Fix
When your terminal becomes unusable (garbled text, no response to input):
```bash
dmux cleanup --terminal
```

### 2. Resource Cleanup
When you notice high CPU/memory usage from orphaned dmux processes:
```bash
dmux cleanup --processes
```

### 3. Storage Cleanup
When you want to remove old session files:
```bash
dmux cleanup --sessions
```

### 4. Complete System Maintenance
Regular maintenance to keep dmux running smoothly:
```bash
dmux cleanup
```

## Exit Codes

- `0` - Cleanup completed successfully
- `1` - Error during cleanup process

## Safety

The cleanup command is designed to be safe:

- **Terminal cleanup**: Only applies standard terminal reset commands
- **Process cleanup**: Only targets dmux-related processes based on specific patterns
- **Session cleanup**: Only removes files for verifiably dead sessions
- **Non-destructive**: Won't affect active sessions or unrelated processes

## Automation

You can safely run cleanup in scripts or automation:

```bash
# Add to cron for regular maintenance
0 2 * * * /usr/local/bin/dmux cleanup --sessions

# Add to shell profile for terminal fixes
alias fix='dmux cleanup --terminal'

# Post-session cleanup script
dmux stop
dmux cleanup
```

## Troubleshooting

### Terminal Still Corrupted
If `--terminal` doesn't fix terminal issues:
```bash
# Try these additional commands
export TERM=xterm
source ~/.bashrc

# Or completely restart your terminal session
```

### Processes Not Cleaned
If orphaned processes persist:
```bash
# Manual process investigation
ps aux | grep dmux
ps aux | grep jcat

# Force kill if necessary (use with caution)
pkill -f dmux
```

### Permission Issues
If cleanup fails with permission errors:
```bash
# Check file permissions
ls -la ~/.jmux/shared/sessions/

# Fix permissions if needed
chmod 644 ~/.jmux/shared/sessions/*.session
```

## Integration with Other Commands

The cleanup functionality is integrated into other dmux commands:

- `dmux status` - Automatically cleans stale sessions before showing status
- `dmux stop` - Could be enhanced to run cleanup after stopping sessions
- `dmux sessions` - Benefits from automatic stale session removal

## Performance Impact

The cleanup command is lightweight:
- **Terminal cleanup**: Near-instant
- **Process cleanup**: Scales with number of processes (typically < 1 second)
- **Session cleanup**: Scales with number of session files (typically < 1 second)

Total runtime is usually under 2 seconds even on busy systems.