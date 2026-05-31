package handler

import (
	"fmt"
	"io"
	"os"

	"github.com/tatsu-t/claude-remote/internal/files"
	"github.com/tatsu-t/claude-remote/server/workspace"
)

// Pull sends a JSON header then a tar.gz of the instance workspace to w.
// Protocol:
//  1. JSON line: PullMeta{"type":"ok",...} or PullMeta{"type":"error",...}
//  2. If "ok": raw tar.gz bytes of the workspace directory
func Pull(mgr *workspace.Manager, instanceID string, r io.Reader, w io.Writer) error {
	if err := validateInstanceID(instanceID); err != nil {
		return files.WritePullHeader(w, files.PullMeta{
			Type:    "error",
			Message: err.Error(),
		})
	}

	inst, err := mgr.Get(instanceID)
	if err != nil {
		if os.IsNotExist(err) {
			return files.WritePullHeader(w, files.PullMeta{
				Type:    "error",
				Message: fmt.Sprintf("instance %q not found", instanceID),
			})
		}
		return err
	}

	if err := files.WritePullHeader(w, files.PullMeta{
		Type:       "ok",
		InstanceID: inst.ID,
		RepoName:   inst.RepoName,
	}); err != nil {
		return err
	}

	workspaceDir := mgr.WorkspaceDir(inst.ID)
	allFiles, err := gatherWorkspaceFiles(workspaceDir)
	if err != nil {
		return fmt.Errorf("gather workspace: %w", err)
	}

	return files.Pack(workspaceDir, allFiles, w)
}

// gatherWorkspaceFiles returns all regular files relative to root.
func gatherWorkspaceFiles(root string) ([]string, error) {
	var result []string
	err := walkDir(root, root, &result)
	return result, err
}

func walkDir(root, dir string, result *[]string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}
		path := dir + "/" + entry.Name()
		rel := path[len(root)+1:]
		if entry.IsDir() {
			if err := walkDir(root, path, result); err != nil {
				return err
			}
		} else {
			*result = append(*result, rel)
		}
	}
	return nil
}
