---
name: pull
description: Download workspace artifacts from a remote instance to the local machine
---

<Purpose>
Run `claude-remote pull` to download the workspace files from a remote instance back to the local machine. Creates a local directory at `./remote/{instance-name}/` with all workspace files.
</Purpose>

<Use_When>
- User runs `/pull` to retrieve results from a remote session
- User says "download from remote", "pull results", "get remote files", "sync back from server"
- User wants to continue work locally after a remote Claude Code session
</Use_When>

<Execution>
Run the following command with any arguments the user provided:

```
claude-remote pull $ARGUMENTS
```

Without arguments, shows running instances to choose from. The command will:
1. Connect to the remote server via SSH
2. Package the workspace as tar.gz
3. Download and extract to `./remote/{instance-name}/`

If a local directory already exists, the user is prompted to choose a new directory or overwrite.

If `claude-remote` is not found, tell the user to run `/setup` first.
</Execution>
