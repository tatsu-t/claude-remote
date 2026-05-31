package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/tatsu-t/claude-remote/internal/config"
	"github.com/tatsu-t/claude-remote/internal/instance"
	"github.com/tatsu-t/claude-remote/internal/remote"
)

// Attach reconnects to a remote instance and streams its output.
// If instanceName is empty, the user is shown a list of recent instances to choose from.
func Attach(ctx context.Context, cfg *config.Config, instanceName string) error {
	if cfg.Server.Host == "" {
		return fmt.Errorf("server host not configured; edit %s", config.Path())
	}

	store, err := instance.NewStore()
	if err != nil {
		return fmt.Errorf("open instance store: %w", err)
	}

	inst, err := selectInstance(store, instanceName)
	if err != nil {
		return err
	}
	if inst == nil {
		fmt.Println("Cancelled.")
		return nil
	}

	fmt.Printf("Attaching to %q...\n", inst.Name)
	if inst.IsTerminal() {
		fmt.Printf("Instance %q is not running.\nShowing the latest available output instead.\n", inst.Name)
	} else {
		fmt.Println("Press Ctrl+C to stop watching.")
	}

	client, err := remote.NewClient(remote.Config{
		Host:    cfg.Server.Host,
		Port:    cfg.Server.Port,
		User:    cfg.Server.User,
		KeyPath: cfg.Server.KeyPath,
	})
	if err != nil {
		return fmt.Errorf("SSH connect to %s: %w", cfg.Server.Host, err)
	}
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	stdout, err := sess.StdoutPipe()
	if err != nil {
		return err
	}

	if err := sess.Start(fmt.Sprintf("claude-remote-server attach %s", inst.ID)); err != nil {
		return fmt.Errorf("start remote attach: %w", err)
	}

	if err := remote.ReadMessages(stdout, func(msg remote.Message) error {
		switch msg.Type {
		case remote.MsgLog:
			var p remote.LogPayload
			if err := remote.DecodePayload(msg, &p); err == nil {
				fmt.Println(p.Text)
			}
		case remote.MsgError:
			var p remote.ErrorPayload
			if err := remote.DecodePayload(msg, &p); err == nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", p.Message)
			}
		case remote.MsgDone:
			return io.EOF
		}
		return nil
	}); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "stream ended: %v\n", err)
	}

	_ = sess.Wait()
	return nil
}

// selectInstance resolves the target instance:
// - if name is given, looks it up in the local store
// - if empty, presents a list to choose from
func selectInstance(store *instance.Store, nameOrID string) (*instance.Instance, error) {
	if nameOrID != "" {
		inst, err := store.Get(nameOrID)
		if err != nil {
			return nil, err
		}
		if inst == nil {
			return nil, fmt.Errorf("instance %q not found", nameOrID)
		}
		return inst, nil
	}

	all, err := store.List()
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("no remote instances found; run /upload first")
	}

	limit := len(all)
	if limit > 9 {
		limit = 9
	}
	recent := all[:limit]

	fmt.Println("Recent remote instances:")
	for i, inst := range recent {
		fmt.Printf("%d) %-24s %s\n", i+1, inst.Name, inst.State)
	}
	fmt.Printf("%d) Cancel\n> ", limit+1)

	choice := readChoice()
	cancelStr := fmt.Sprintf("%d", limit+1)
	if choice == cancelStr || choice == "" {
		return nil, nil
	}

	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	if idx < 1 || idx > limit {
		fmt.Println("Invalid choice.")
		return nil, nil
	}
	return recent[idx-1], nil
}
