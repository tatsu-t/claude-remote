package handler

import (
	"encoding/json"
	"io"

	"github.com/tatsu-t/claude-remote/server/workspace"
)

// List writes a JSON array of all instances to w.
func List(mgr *workspace.Manager, w io.Writer) error {
	instances, err := mgr.List()
	if err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(instances)
}
