package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tatsu-t/claude-remote/internal/instance"
)

const DefaultDataDir = "/var/lib/claude-remote"

// Manager handles the instances/ directory layout on the server.
// Each instance lives at {dataDir}/instances/{id}/ with subdirs:
//   - repo/      — git bare clone of the original repo
//   - workspace/ — git worktree (active work directory)
//   - artifacts/ — output logs and produced files
type Manager struct {
	dataDir string
}

func NewManager(dataDir string) *Manager {
	if dataDir == "" {
		dataDir = DefaultDataDir
	}
	return &Manager{dataDir: dataDir}
}

func (m *Manager) InstanceDir(id string) string {
	return filepath.Join(m.dataDir, "instances", id)
}

func (m *Manager) RepoDir(id string) string {
	return filepath.Join(m.InstanceDir(id), "repo")
}

func (m *Manager) WorkspaceDir(id string) string {
	return filepath.Join(m.InstanceDir(id), "workspace")
}

func (m *Manager) ArtifactsDir(id string) string {
	return filepath.Join(m.InstanceDir(id), "artifacts")
}

func (m *Manager) MetaPath(id string) string {
	return filepath.Join(m.InstanceDir(id), "meta.json")
}

// Create initialises the directory tree and persists the instance metadata.
func (m *Manager) Create(inst *instance.Instance) error {
	for _, sub := range []string{"repo", "workspace", "artifacts"} {
		dir := filepath.Join(m.InstanceDir(inst.ID), sub)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", sub, err)
		}
	}
	return m.saveMeta(inst)
}

func (m *Manager) Get(id string) (*instance.Instance, error) {
	data, err := os.ReadFile(m.MetaPath(id))
	if err != nil {
		return nil, err
	}
	var inst instance.Instance
	if err := json.Unmarshal(data, &inst); err != nil {
		return nil, err
	}
	return &inst, nil
}

func (m *Manager) UpdateState(id string, state instance.State) error {
	inst, err := m.Get(id)
	if err != nil {
		return err
	}
	inst.State = state
	inst.UpdatedAt = time.Now()
	return m.saveMeta(inst)
}

func (m *Manager) SetRemoteURL(id string, url string) error {
	inst, err := m.Get(id)
	if err != nil {
		return err
	}
	inst.RemoteURL = url
	inst.UpdatedAt = time.Now()
	return m.saveMeta(inst)
}

func (m *Manager) List() ([]*instance.Instance, error) {
	instancesDir := filepath.Join(m.dataDir, "instances")
	entries, err := os.ReadDir(instancesDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var result []*instance.Instance
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		inst, err := m.Get(entry.Name())
		if err != nil {
			continue // skip corrupted entries
		}
		result = append(result, inst)
	}
	return result, nil
}

func (m *Manager) FindByRepo(repoName string) ([]*instance.Instance, error) {
	all, err := m.List()
	if err != nil {
		return nil, err
	}
	var result []*instance.Instance
	for _, inst := range all {
		if inst.RepoName == repoName {
			result = append(result, inst)
		}
	}
	return result, nil
}

func (m *Manager) saveMeta(inst *instance.Instance) error {
	data, err := json.MarshalIndent(inst, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.MetaPath(inst.ID), data, 0644)
}
