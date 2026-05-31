package handler

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/tatsu-t/claude-remote/internal/files"
	"github.com/tatsu-t/claude-remote/internal/instance"
	"github.com/tatsu-t/claude-remote/internal/remote"
	"github.com/tatsu-t/claude-remote/server/workspace"
)

// Upload handles the upload command from a client:
//  1. Reads the JSON header line (UploadMeta) from r
//  2. Validates all client-supplied metadata fields
//  3. Generates a server-side instance ID (never trusts the client's ID)
//  4. Reads the tar.gz payload and extracts it to workspace/
//  5. Starts claude --server-mode and streams its output as JSON Lines to w
func Upload(mgr *workspace.Manager, r io.Reader, w io.Writer) error {
	br := bufio.NewReader(r)

	meta, err := files.ReadHeader(br)
	if err != nil {
		_ = remote.WriteMessage(w, remote.MsgError, remote.ErrorPayload{Message: fmt.Sprintf("read header: %v", err)})
		return err
	}

	// Validate all client-supplied fields used in file paths or stored metadata.
	if err := validateInstanceName(meta.InstanceName); err != nil {
		_ = remote.WriteMessage(w, remote.MsgError, remote.ErrorPayload{Message: err.Error()})
		return err
	}
	if err := validateRepoName(meta.RepoName); err != nil {
		_ = remote.WriteMessage(w, remote.MsgError, remote.ErrorPayload{Message: err.Error()})
		return err
	}
	if err := validateBranch(meta.Branch); err != nil {
		_ = remote.WriteMessage(w, remote.MsgError, remote.ErrorPayload{Message: err.Error()})
		return err
	}

	// Generate the instance ID server-side; never use the client-supplied ID in paths.
	serverID := serverGenerateID()

	inst := &instance.Instance{
		ID:        serverID,
		Name:      meta.InstanceName,
		RepoName:  meta.RepoName,
		Branch:    meta.Branch,
		State:     instance.StateCreating,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := mgr.Create(inst); err != nil {
		_ = remote.WriteMessage(w, remote.MsgError, remote.ErrorPayload{Message: fmt.Sprintf("create workspace: %v", err)})
		return err
	}

	_ = remote.WriteMessage(w, remote.MsgLog, remote.LogPayload{Text: "Extracting files..."})

	if err := files.Unpack(br, mgr.WorkspaceDir(inst.ID)); err != nil {
		_ = mgr.UpdateState(inst.ID, instance.StateFailed)
		_ = remote.WriteMessage(w, remote.MsgError, remote.ErrorPayload{Message: fmt.Sprintf("unpack: %v", err)})
		return err
	}

	_ = mgr.UpdateState(inst.ID, instance.StateRunning)
	_ = remote.WriteMessage(w, remote.MsgLog, remote.LogPayload{Text: "Starting Claude Code..."})

	session, err := workspace.StartClaude(mgr.WorkspaceDir(inst.ID))
	if err != nil {
		_ = mgr.UpdateState(inst.ID, instance.StateFailed)
		_ = remote.WriteMessage(w, remote.MsgError, remote.ErrorPayload{Message: fmt.Sprintf("start claude: %v", err)})
		return err
	}

	if session.URL != "" {
		_ = mgr.SetRemoteURL(inst.ID, session.URL)
		_ = remote.WriteMessage(w, remote.MsgURL, remote.URLPayload{
			URL:        session.URL,
			InstanceID: inst.ID,
		})
	}

	// Tee logs to artifacts/claude.log for future attach requests.
	logPath := filepath.Join(mgr.ArtifactsDir(inst.ID), "claude.log")
	logFile, logErr := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if logErr != nil {
		logFile = nil
	}
	defer func() {
		if logFile != nil {
			logFile.Close()
		}
	}()

	for {
		select {
		case line, ok := <-session.LogLines:
			if !ok {
				_ = mgr.UpdateState(inst.ID, instance.StateCompleted)
				_ = remote.WriteMessage(w, remote.MsgDone, nil)
				return nil
			}
			_ = remote.WriteMessage(w, remote.MsgLog, remote.LogPayload{Text: line})
			if logFile != nil {
				fmt.Fprintln(logFile, line)
			}
		case err := <-session.Done:
			state := instance.StateCompleted
			if err != nil {
				state = instance.StateFailed
			}
			_ = mgr.UpdateState(inst.ID, state)
			_ = remote.WriteMessage(w, remote.MsgDone, nil)
			return err
		}
	}
}
