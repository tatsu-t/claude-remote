Guide the user through interactive setup of claude-remote. Follow the steps below exactly.

> Note: Claude Code auto mode blocks shell execution. When you need to run a command, show it to the user prefixed with `!` so they can paste it directly (e.g. `! go version`). Do not use the Bash tool directly.

## Step 1: Check Go

Ask the user to run: `! go version`

If Go is not installed, tell the user to install Go 1.22+ from https://go.dev/dl/ and stop.

## Step 2: Build & install binary

Ask the user to run: `! claude-remote --version`

If the command is not found:

If NOT_FOUND, run:
```
go install github.com/tatsu-t/claude-remote/cmd/claude-remote@latest
```
Confirm success with `claude-remote --version`.

## Step 3: Show menu

Ask the user (use AskUserQuestion with multiSelect: true):

**"What would you like to set up?"**
- **Copy commands globally** — Copy /upload, /attach, /pull, /install to ~/.claude/commands/ so they work in any project
- **Configure client** — Set up ~/.claude-remote/config.toml with your server connection details
- **Set up server (SSH)** — SSH into your remote server and deploy claude-remote-server via Docker
- **Set up server (local Docker)** — Deploy the server locally with Docker for testing

Run only the sections the user selected.

---

## Section A: Copy commands globally

Copy the 4 command files to `~/.claude/commands/`:

```bash
mkdir -p ~/.claude/commands
cp "$(dirname $0)/../../.claude/commands/upload.md" ~/.claude/commands/upload.md
cp "$(dirname $0)/../../.claude/commands/attach.md" ~/.claude/commands/attach.md
cp "$(dirname $0)/../../.claude/commands/pull.md"   ~/.claude/commands/pull.md
cp "$(dirname $0)/../../.claude/commands/install.md" ~/.claude/commands/install.md
```

If the source path is unclear, find the plugin directory from the marketplace:
`ls ~/.claude/plugins/marketplaces/claude-remote/.claude/commands/`

Then copy from there. Confirm each file was copied. Tell the user: "/upload, /attach, /pull, /install are now available in all projects."

---

## Section B: Configure client

1. Run `claude-remote config show` and display the output.
2. Ask the user for each missing value (server host, port, SSH user, SSH key path).
   - Default port: 2222
   - Default user: claude-remote
   - Default key: ~/.ssh/id_ed25519
3. Write the config by running:
```bash
claude-remote config set server.host <HOST>
claude-remote config set server.port <PORT>
claude-remote config set server.user <USER>
claude-remote config set server.key  <KEY_PATH>
```
4. Run `claude-remote config show` again to confirm.

---

## Section C: Set up server via SSH

**Safety rules:**
- Never overwrite authorized_keys — only append
- Never echo or log the API key
- Ask for confirmation before each step

1. Ask for: server hostname, SSH port (default 22), SSH user (default root or ubuntu), SSH key path.
2. Test connectivity: `ssh -p <PORT> -i <KEY> <USER>@<HOST> 'echo OK'`. If it fails, show the error and stop.
3. Ask: "The server needs ANTHROPIC_API_KEY. Please enter it now (it will not be stored):" — read it as a variable, never print it.
4. Ask for confirmation before each of the following steps:

**a) Add your SSH public key to the server's authorized_keys** (so claude-remote-server can accept connections):
```bash
# Ask which local key to add (default ~/.ssh/id_ed25519.pub)
ssh -p <PORT> -i <KEY> <USER>@<HOST> 'mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys' < ~/.ssh/id_ed25519.pub
```

**b) Install Docker on the server** (skip if already installed):
```bash
ssh -p <PORT> -i <KEY> <USER>@<HOST> 'which docker || (curl -fsSL https://get.docker.com | sh)'
```

**c) Clone the repository on the server**:
```bash
ssh -p <PORT> -i <KEY> <USER>@<HOST> 'git clone https://github.com/tatsu-t/claude-remote.git ~/claude-remote 2>/dev/null || (cd ~/claude-remote && git pull)'
```

**d) Start the server**:
```bash
ssh -p <PORT> -i <KEY> <USER>@<HOST> "cd ~/claude-remote/deploy && ANTHROPIC_API_KEY=<API_KEY> docker compose up -d --build"
```
Pass the API key via the command but do NOT display it in output.

**e) Verify the server is running**:
```bash
ssh -p <PORT> -i <KEY> <USER>@<HOST> 'docker ps | grep claude-remote'
```

5. If all steps succeeded, show a summary with the server address and suggested config.toml values.

---

## Section D: Set up server locally with Docker

1. Check Docker is running: `docker info 2>/dev/null | head -5`. If not, ask user to start Docker Desktop.
2. Find the plugin/repo directory (look in `~/.claude/plugins/marketplaces/claude-remote/` or current working directory).
3. Ask for ANTHROPIC_API_KEY (read securely, do not log).
4. Run:
```bash
cd <repo-dir>/deploy
ANTHROPIC_API_KEY=<API_KEY> docker compose up -d --build
```
5. Confirm with: `docker ps | grep claude-remote`
6. Show: "Server running on localhost:2222. Add your SSH key: `ssh-copy-id -i ~/.ssh/id_ed25519 -p 2222 claude-remote@localhost`"

---

## Final step

Show a summary of what was completed and what the user should do next.
