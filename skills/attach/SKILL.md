---
name: attach
description: Reconnect to a running remote Claude Code instance and stream its logs
---

<Purpose>
Run `claude-remote attach` to reconnect to a running remote instance. Without arguments, shows an interactive list of recent instances to choose from. With an instance name or ID, connects directly.
</Purpose>

<Use_When>
- User runs `/attach` to reconnect to a remote session
- User says "reconnect to remote", "attach to instance", "check remote session", "what is the remote doing"
- User wants to see logs from a previously uploaded session
</Use_When>

<Execution>
Run the following command with any arguments the user provided:

```
claude-remote attach $ARGUMENTS
```

Without arguments, an interactive list of recent instances is shown (up to 9). The user selects one by number.

With an instance name or ID, connects directly to that instance and streams its log output. Press Ctrl+C to detach locally — the remote session continues running.

If `claude-remote` is not found, tell the user to run `/setup` first.
</Execution>
