Run `claude-remote config show` to check the current configuration. If not configured, guide the user through initial setup:

1. Check if Go is installed: run `go version`. If not found, tell the user to install Go 1.22+ from https://go.dev/dl/ and stop.
2. Check if `claude-remote` is already installed: run `claude-remote --version 2>/dev/null`. If found, skip to step 4.
3. Build and install from source: run `go install github.com/tatsu-t/claude-remote/cmd/claude-remote@latest`
4. Show the current config: run `claude-remote config show`
5. If the server host is empty, guide the user to edit `~/.claude-remote/config.toml`:
   ```toml
   [server]
   host = "your-server.example.com"
   port = 2222
   user = "claude-remote"
   key  = "~/.ssh/id_ed25519"
   ```
6. For Docker server deployment:
   ```
   cd <plugin-dir>/deploy
   docker compose up --build -d
   ssh-copy-id -i ~/.ssh/id_ed25519 -p 2222 claude-remote@localhost
   ```
