package instance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// Store persists instance metadata locally at ~/.claude-remote/instances.json.
type Store struct {
	path string
}

func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".claude-remote")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &Store{path: filepath.Join(dir, "instances.json")}, nil
}

func (s *Store) List() ([]*Instance, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var instances []*Instance
	if err := json.Unmarshal(data, &instances); err != nil {
		return nil, err
	}
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].UpdatedAt.After(instances[j].UpdatedAt)
	})
	return instances, nil
}

func (s *Store) Get(nameOrID string) (*Instance, error) {
	instances, err := s.List()
	if err != nil {
		return nil, err
	}
	for _, inst := range instances {
		if inst.ID == nameOrID || inst.Name == nameOrID {
			return inst, nil
		}
	}
	return nil, nil
}

func (s *Store) Save(inst *Instance) error {
	instances, err := s.List()
	if err != nil {
		return err
	}
	found := false
	for i, existing := range instances {
		if existing.ID == inst.ID {
			instances[i] = inst
			found = true
			break
		}
	}
	if !found {
		instances = append(instances, inst)
	}
	data, err := json.MarshalIndent(instances, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func (s *Store) Delete(id string) error {
	instances, err := s.List()
	if err != nil {
		return err
	}
	filtered := instances[:0]
	for _, inst := range instances {
		if inst.ID != id {
			filtered = append(filtered, inst)
		}
	}
	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func (s *Store) ListByRepo(repoName string) ([]*Instance, error) {
	all, err := s.List()
	if err != nil {
		return nil, err
	}
	var result []*Instance
	for _, inst := range all {
		if inst.RepoName == repoName {
			result = append(result, inst)
		}
	}
	return result, nil
}
