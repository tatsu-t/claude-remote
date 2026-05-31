Run `claude-remote pull $ARGUMENTS` to download artifacts from a remote instance to your local machine.

Usage:
  /pull                     — Show a list of recent instances to choose from
  /pull <instance-name>     — Pull the named instance directly

Behaviour:
- If the instance is still running, asks whether to pull current state or attach instead
- Restores files to ./remote/<instance-name>/ by default
- If that directory already exists, offers to create a new directory or overwrite

The downloaded directory contains the full remote workspace at the time of the pull.
