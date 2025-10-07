#!/usr/bin/env bash
set -euo pipefail

# CONFIG
SESSION="${SESSION:-tmux_testing}"        # tmux session name
LOGDIR="${LOGDIR:-/tmp/tmux-logs}"         # directory where logs will be stored 
SSH_TARGET="${SSH_TARGET:-dory@xsrl8-emp-166.xsight.ent}" # change to your ssh target (user@host)
TMUX_BIN="${TMUX_BIN:-tmux}"                # path to tmux if needed

# create log dir
mkdir -p "$LOGDIR"

# make timestamped filenames (optional)
timestamp() { echo ""; } #date +"%Y%m%d-%H%M%S"; }
TS="$(timestamp)"
SSH_LOG="$LOGDIR/machine1${TS}.log"
SHELL_LOG="$LOGDIR/machine2${TS}.log"

# avoid clobbering an existing session
if $TMUX_BIN has-session -t "$SESSION" 2>/dev/null; then
    $TMUX_BIN kill-session -t "$SESSION" 2>/dev/null || true
#   echo "tmux session '$SESSION' already exists. Attach with: tmux attach -t $SESSION"
#   exit 1
fi

# 1️⃣ Create a session with a dummy shell window first
$TMUX_BIN new-session -d -s "$SESSION" "/bin/bash --norc"

# 2️⃣ Replace the first pane’s command with the SSH+logging command
$TMUX_BIN send-keys -t "$SESSION:1.0" \
  "exec script -q -f \"$SSH_LOG\" -c 'ssh $SSH_TARGET'" C-m

# 3️⃣ Split to make the second pane
$TMUX_BIN split-window -h -t "$SESSION:1" \
  "exec script -q -f \"$SHELL_LOG\" -c '/bin/bash --norc'"


# optional: set pane widths/initial focus. Here we select left pane (ssh) as active.
$TMUX_BIN select-pane -t "$SESSION:1.0"

# tmux send-keys -t $SESSION:1.0 'echo machine1(remote)' C-m
# tmux send-keys -t $SESSION:1.1 'echo machine2(local)' C-m

# attach to the session
# $TMUX_BIN attach-session -t "$SESSION"
echo "To attach to the tmux session, run: $TMUX_BIN attach-session -t $SESSION"
echo "Logs are being saved to: $SSH_LOG and $SHELL_LOG"
# done
