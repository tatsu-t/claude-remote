---
name: upload
description: Transfer current workspace to a remote server and start a Claude Code session there
---

<Purpose>
Run `claude-remote upload` to scan the current git repository, select relevant files using Claude, transfer them to the configured remote server via SSH, and start a Claude Code session in server mode. Streams the Remote Control URL and logs back to the local terminal.
</Purpose>

<Use_When>
- User runs `/upload` or types `upload` to hand off work to a remote machine
- User says "send to remote", "run remotely", "upload to server", "hand off to server"
- User wants to continue a Claude Code session on a more powerful or always-on remote machine
</Use_When>

<Execution>
Run the following command with any arguments the user provided:

```
claude-remote upload $ARGUMENTS
```

The interactive flow will:
1. Scan the git repository for changed/untracked files
2. Use Claude Haiku to select files needed for continuation
3. Show a file list with sizes for confirmation
4. Transfer files as tar.gz over SSH
5. Start `claude --server-mode` on the remote server
6. Stream the Remote Control URL and live logs

If `claude-remote` is not found, tell the user to run `/setup` first.
</Execution>

<Options>
- `--max-upload-size=200MB` — Maximum upload size (0 = unlimited)
- `--server=HOST` — Override the configured server host
</Options>
