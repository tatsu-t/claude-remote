---
name: setup
description: Install and configure the claude-remote CLI for SSH-based remote handoff
---

<Purpose>
Build and install the `claude-remote` binary from source, then guide the user through initial configuration of `~/.claude-remote/config.toml` with their remote server details.
</Purpose>

<Use_When>
- User runs `/setup` for the first time after installing this plugin
- User says "install claude-remote", "setup remote", "configure remote server"
- `claude-remote` binary is not found in PATH
</Use_When>

<Execution>
1. Check if Go is installed: run `go version`. If not found, tell the user to install Go 1.22+ from https://go.dev/dl/ and stop.
2. Check if `claude-remote` is already installed: run `claude-remote --version 2>/dev/null`. If found, skip to step 4.
3. Build and install from the plugin directory:
   - Identify the plugin installation directory (where this SKILL.md lives, two levels up)
   - Run: `go install github.com/tatsu-t/claude-remote/cmd/claude-remote@latest`
   - Or if working from source: `cd <plugin-dir> && go build -o claude-remote ./cmd/claude-remote && mv claude-remote ~/bin/` (adjust PATH as needed)
4. Check current config: run `claude-remote config show`
5. If not configured, guide the user to edit `~/.claude-remote/config.toml`:
   ```toml
   [server]
   host = "your-server.example.com"
   port = 2222
   user = "claude-remote"
   key  = "~/.ssh/id_ed25519"
   ```
6. For Docker deployment, show:
   ```
   cd <plugin-dir>/deploy
   docker compose up --build -d
   ssh-copy-id -i ~/.ssh/id_ed25519 -p 2222 claude-remote@localhost
   ```
7. Confirm setup: run `claude-remote config show` and display the result.
</Execution>

<Notes>
- The server requires `claude-remote-server` binary on PATH and `ANTHROPIC_API_KEY` set.
- Docker deployment in `deploy/` is the recommended way to run the server.
- SSH key authentication only; password auth is disabled.
</Notes>
