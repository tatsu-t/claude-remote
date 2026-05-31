Run `claude-remote attach $ARGUMENTS` to reconnect to a running remote instance.

Usage:
  /attach                   — Show a list of recent instances to choose from
  /attach <instance-name>   — Directly attach to the named instance

Behaviour:
- If the instance is running, streams live output until Ctrl+C
- If the instance has completed, shows the most recent output log
- The remote session continues after you detach

The instance name is shown after /upload completes (e.g. "my-app-1234").
