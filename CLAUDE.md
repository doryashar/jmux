## Development Workflow
- After each bug fix or feature implementation, update the version of the tool, commit, and push to GitHub, then bump to next dev version.

## Build Considerations
- Always build a static portable version to avoid library compatibility issues (e.g., GLIBC version mismatches)
- Ensure the binary can run on different Linux distributions without dependency problems

## Project Structure
- Always keep tests in tests directory

## Testing Setup
- Use `./scripts/tmux-test-setup.sh` to run a tmux session called `tmux_testing` with 2 panes (remote and local machines)
- Panes are monitored using screen command into log files: `/tmp/tmux-logs/machine1` and `/tmp/tmux-logs/machine2`
- Both machines have access to `/projects/common/work/dory/jmux`
- After building static binary, copy to `/projects/common/work/dory/jmux/bin`
- Send commands to specific panes using tmux send-keys:
  * Example: `tmux send-keys -t tmux_testing:1.0 'echo machine1(remote)' C-m`
  * Example: `tmux send-keys -t tmux_testing:1.1 'echo machine2(local)' C-m`
- machine1 (remote) should always listen while machine2 (local) should always connect