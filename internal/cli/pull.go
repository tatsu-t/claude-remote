package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/tatsu-t/claude-remote/internal/config"
	"github.com/tatsu-t/claude-remote/internal/files"
	"github.com/tatsu-t/claude-remote/internal/instance"
	"github.com/tatsu-t/claude-remote/internal/remote"
)

// Pull downloads workspace artifacts from a remote instance to ./remote/{name}/.
func Pull(ctx context.Context, cfg *config.Config, instanceName string) error {
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

	fmt.Printf("Preparing to pull %q...\n", inst.Name)
	fmt.Printf("Checking remote workspace...\n")

	// Running instance: offer to pull current state or attach instead
	if inst.IsRunning() {
		fmt.Printf("\nInstance %q is still running.\n\n", inst.Name)
		fmt.Printf("Choose how to continue:\n")
		fmt.Printf("1) Pull current state\n")
		fmt.Printf("2) Attach instead\n")
		fmt.Printf("3) Cancel\n> ")

		switch readChoice() {
		case "1":
			// proceed to pull
		case "2":
			return Attach(ctx, cfg, inst.Name)
		default:
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Resolve local destination
	destDir, ok := resolveDestDir(inst.Name)
	if !ok {
		fmt.Println("Cancelled.")
		return nil
	}

	// Connect and pull
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

	if err := sess.Start(fmt.Sprintf("claude-remote-server pull %s", inst.ID)); err != nil {
		return fmt.Errorf("start remote pull: %w", err)
	}

	// Protocol: JSON header line, then raw tar.gz
	br := bufio.NewReader(stdout)
	meta, err := files.ReadPullHeader(br)
	if err != nil {
		return fmt.Errorf("read pull header: %w", err)
	}
	if meta.Type == "error" {
		return fmt.Errorf("server error: %s", meta.Message)
	}

	fmt.Printf("Restoring to %s...\n", destDir)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}

	if err := files.Unpack(br, destDir); err != nil {
		return fmt.Errorf("unpack: %w", err)
	}

	_ = sess.Wait()

	fmt.Printf("\nPull complete.\n")
	fmt.Printf("- Instance:   %s\n", inst.Name)
	fmt.Printf("- Restored to: %s\n\n", destDir)
	fmt.Printf("Next actions:\n")
	fmt.Printf("  Review output: claude-remote attach %s\n", inst.Name)
	return nil
}

// resolveDestDir determines where to restore, handling conflicts.
// Returns ("", false) if the user cancels.
func resolveDestDir(instanceName string) (string, bool) {
	base := filepath.Join(".", "remote", instanceName)

	if _, err := os.Stat(base); os.IsNotExist(err) {
		return base, true
	}

	fmt.Printf("\nA local directory for this repo already exists.\n\n")
	fmt.Printf("Choose how to continue:\n")
	fmt.Printf("1) Restore into a new directory\n")
	fmt.Printf("2) Overwrite the existing restore directory\n")
	fmt.Printf("3) Cancel\n> ")

	switch readChoice() {
	case "1":
		// Append timestamp suffix
		suffix := fmt.Sprintf("%d", nowUnix())
		return base + "-" + suffix, true
	case "2":
		return base, true
	default:
		return "", false
	}
}

// nowUnix returns the current Unix timestamp (abstracted for testability).
var nowUnix = func() int64 {
	f, _ := os.Stat(".")
	if f != nil {
		return f.ModTime().Unix()
	}
	return 0
}

// Ensure io is used (for future pipe reads in streaming pull)
var _ io.Reader = (*bufio.Reader)(nil)
