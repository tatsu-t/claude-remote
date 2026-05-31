Run `claude-remote upload $ARGUMENTS` to transfer the current workspace to the remote server and start a Claude Code session there.

This command:
1. Scans your git repository for relevant files
2. Uses Claude to select the files needed for continuation
3. Transfers them to the configured remote server via SSH
4. Starts Claude Code in server mode on the remote machine
5. Streams the remote output back so you can watch progress

Configuration: edit `~/.claude-remote/config.toml` with your server details.
Run `claude-remote config show` to check the current configuration.

Options:
  --max-upload-size=200MB   Maximum upload size (use 0 for unlimited)
  --server=HOST             Override the server host from config
