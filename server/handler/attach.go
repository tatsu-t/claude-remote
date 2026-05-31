package handler

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/tatsu-t/claude-remote/internal/instance"
	"github.com/tatsu-t/claude-remote/internal/remote"
	"github.com/tatsu-t/claude-remote/server/workspace"
)

// Attach streams output for the given instance to w.
// For completed/failed instances it replays the stored log.
// For running instances it also replays the stored log (live attach via
// process management is a future enhancement).
func Attach(mgr *workspace.Manager, instanceID string, r io.Reader, w io.Writer) error {
	if err := validateInstanceID(instanceID); err != nil {
		return remote.WriteMessage(w, remote.MsgError, remote.ErrorPayload{Message: err.Error()})
	}

	inst, err := mgr.Get(instanceID)
	if err != nil {
		if os.IsNotExist(err) {
			return remote.WriteMessage(w, remote.MsgError, remote.ErrorPayload{
				Message: fmt.Sprintf("instance %q not found", instanceID),
			})
		}
		return err
	}

	// Send instance state as first message so the client can show context.
	_ = remote.WriteMessage(w, remote.MsgLog, remote.LogPayload{
		Text: fmt.Sprintf("[State: %s | Remote Control: %s]", inst.State, inst.RemoteURL),
	})

	if inst.IsTerminal() {
		_ = remote.WriteMessage(w, remote.MsgLog, remote.LogPayload{
			Text: fmt.Sprintf("Instance %q is not running. Showing the latest available output instead.", inst.Name),
		})
	}

	if err := streamStoredLog(mgr, inst, w); err != nil {
		return err
	}

	if inst.State == instance.StateRunning || inst.State == instance.StateAttached {
		_ = mgr.UpdateState(instanceID, instance.StateAttached)
		_ = remote.WriteMessage(w, remote.MsgLog, remote.LogPayload{
			Text: "Press Ctrl+C to stop watching. Remote work will continue.",
		})
	}

	return remote.WriteMessage(w, remote.MsgDone, nil)
}

func streamStoredLog(mgr *workspace.Manager, inst *instance.Instance, w io.Writer) error {
	logPath := filepath.Join(mgr.ArtifactsDir(inst.ID), "claude.log")
	data, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return remote.WriteMessage(w, remote.MsgLog, remote.LogPayload{Text: "(no output recorded yet)"})
	}
	if err != nil {
		return err
	}
	return remote.WriteMessage(w, remote.MsgLog, remote.LogPayload{Text: string(data)})
}
