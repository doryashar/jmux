# dmux Sharing Modes

dmux now supports three different sharing modes, similar to wemux, that provide different levels of control and interaction for joining users.

## Sharing Modes

### 1. Pair Mode (Default)
**Command**: `dmux share` or `dmux share --name session-name`

- **Description**: Standard shared session where all users have full control
- **Behavior**: All participants can type, navigate, and control the session equally
- **Use case**: Collaborative work, pair programming, shared debugging

```bash
# Start a pair session
dmux share collaborative-work

# Join a pair session (uses session's configured mode)
dmux join alice collaborative-work
```

### 2. View-Only Mode
**Command**: `dmux share --view` or `dmux share --view --name session-name`

- **Description**: Host has full control, joining users can only observe (read-only)
- **Behavior**: Joining users see everything but cannot type or control the session
- **Use case**: Presentations, demonstrations, teaching, code reviews

```bash
# Share in view-only mode
dmux share --view presentation

# Join in view-only mode (even if session allows more)
dmux join alice presentation --view
```

### 3. Rogue Mode
**Command**: `dmux share --rogue` or `dmux share --rogue --name session-name`

- **Description**: Independent control within the same tmux server
- **Behavior**: Each user gets their own session but shares the same tmux server and windows
- **Use case**: Independent work on the same project, individual exploration

```bash
# Share in rogue mode
dmux share --rogue development

# Join in rogue mode (even if session has different mode)
dmux join bob development --rogue
```

## Mode Override on Join

When joining a session, you can override the session's configured mode:

```bash
# Force view-only mode regardless of session mode
dmux join alice session --view

# Force rogue mode regardless of session mode  
dmux join alice session --rogue

# Use session's configured mode (default)
dmux join alice session
```

## Technical Details

### tmux Commands Used

- **Pair Mode**: `tmux attach-session -t session-name`
- **View Mode**: `tmux attach-session -t session-name -r`
- **Rogue Mode**: `tmux new-session -t session-name`

### Session Information

Use `dmux sessions` to see all active sessions with their modes:

```
Active Shared Sessions
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

User: alice
  Session: presentation
  Port: 12345
  Started: 14:30:00 (5m ago)
  Public session
  Mode: View-only (read-only)
  To join: dmux join alice

User: bob  
  Session: development
  Port: 12346
  Started: 14:25:00 (10m ago)
  Public session
  Mode: Rogue (independent control)
  To join: dmux join bob
```

## Compatibility

- **Backward Compatibility**: Existing sessions without a mode automatically use pair mode
- **Network Sessions**: Remote sessions (via jcat) currently use the session's configured mode
- **Local Sessions**: Direct tmux connection for same-user sessions supports all modes

## Examples

### Teaching/Presentation Scenario
```bash
# Teacher shares screen in view-only mode
dmux share --view programming-lesson

# Students join to watch
dmux join teacher programming-lesson
# They automatically get read-only access
```

### Code Review Scenario  
```bash
# Developer shares code for review
dmux share --view code-review

# Reviewers can watch and discuss
dmux join developer code-review --view
```

### Independent Development
```bash
# Team lead starts shared development environment
dmux share --rogue team-project

# Team members join with independent control
dmux join lead team-project
# Each gets their own session in the same server
```

### Collaborative Debugging
```bash
# Start collaborative session (pair mode)
dmux share debug-session

# Multiple developers join with shared control
dmux join alice debug-session
dmux join bob debug-session
```