package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/tatsu-t/claude-remote/server/handler"
	"github.com/tatsu-t/claude-remote/server/workspace"
)

func main() {
	args := resolveArgs()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: claude-remote-server <upload|attach|pull|list> [instance-id]")
		os.Exit(1)
	}

	dataDir := os.Getenv("CLAUDE_REMOTE_DATA_DIR")
	mgr := workspace.NewManager(dataDir)

	if err := dispatch(mgr, args); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// resolveArgs returns os.Args[1:] when called directly, or parses
// SSH_ORIGINAL_COMMAND when invoked as a forced command.
func resolveArgs() []string {
	if len(os.Args) > 1 {
		return os.Args[1:]
	}
	if cmd := os.Getenv("SSH_ORIGINAL_COMMAND"); cmd != "" {
		return strings.Fields(cmd)
	}
	return nil
}

func dispatch(mgr *workspace.Manager, args []string) error {
	switch args[0] {
	case "upload":
		return handler.Upload(mgr, os.Stdin, os.Stdout)

	case "attach":
		if len(args) < 2 {
			return fmt.Errorf("attach requires an instance ID")
		}
		return handler.Attach(mgr, args[1], os.Stdin, os.Stdout)

	case "pull":
		if len(args) < 2 {
			return fmt.Errorf("pull requires an instance ID")
		}
		return handler.Pull(mgr, args[1], os.Stdin, os.Stdout)

	case "list":
		return handler.List(mgr, os.Stdout)

	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
