package instance

import "time"

type State string

const (
	StateCreating  State = "creating"
	StateRunning   State = "running"
	StateAttached  State = "attached"
	StateCompleted State = "completed"
	StateFailed    State = "failed"
	StateArchived  State = "archived"
)

type Instance struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	RepoName   string    `json:"repo_name"`
	Branch     string    `json:"branch"`
	State      State     `json:"state"`
	RemoteURL  string    `json:"remote_url,omitempty"`
	ServerHost string    `json:"server_host"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (i *Instance) IsRunning() bool {
	return i.State == StateRunning || i.State == StateAttached
}

func (i *Instance) IsTerminal() bool {
	return i.State == StateCompleted || i.State == StateFailed || i.State == StateArchived
}
