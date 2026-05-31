# claude-remote

SSH-based remote handoff for Claude Code sessions.

Upload your local workspace to a remote server, start Claude Code there, and pull results back — all from slash commands inside Claude Code.

## Install as oh-my-claudecode plugin

```
/plugin marketplace add https://github.com/tatsu-t/claude-remote
```

Then run `/setup` to build the binary and configure your server.

## Commands

| Slash command | What it does |
|---------------|-------------|
| `/setup`  | Build & install the `claude-remote` binary, configure `~/.claude-remote/config.toml` |
| `/upload` | Transfer current workspace to remote server, start Claude Code in server mode |
| `/attach` | Reconnect to a running remote instance and stream its logs |
| `/pull`   | Download workspace artifacts from a remote instance |

## Requirements

- Go 1.22+ (client build)
- SSH key pair (`~/.ssh/id_ed25519` recommended)
- Remote server running `claude-remote-server` with `ANTHROPIC_API_KEY` set

## Server setup (Docker)

```bash
cd deploy
cp /dev/stdin authorized_keys <<< "$(cat ~/.ssh/id_ed25519.pub)"
docker compose up --build -d
```

Then in `~/.claude-remote/config.toml`:

```toml
[server]
host = "your-server.example.com"
port = 2222
user = "claude-remote"
key  = "~/.ssh/id_ed25519"
```

## Architecture

```
Local (Claude Code CLI)
  /upload ─── SSH ──→ claude-remote-server upload
  │                     └── extract tar.gz → start claude --server-mode → stream logs
  /attach ─── SSH ──→ claude-remote-server attach <id>
  │                     └── stream stored log or live output
  /pull   ─── SSH ──→ claude-remote-server pull <id>
                        └── JSON header + tar.gz of workspace
```

## License

MIT
