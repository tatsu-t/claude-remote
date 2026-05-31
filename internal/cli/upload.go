package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/tatsu-t/claude-remote/internal/config"
	"github.com/tatsu-t/claude-remote/internal/files"
	"github.com/tatsu-t/claude-remote/internal/instance"
	"github.com/tatsu-t/claude-remote/internal/remote"
)

// UploadOptions configures the upload command.
type UploadOptions struct {
	MaxUploadSize string
}

// Upload implements the /upload flow: gather → select → confirm → transfer → stream.
func Upload(ctx context.Context, cfg *config.Config, opts UploadOptions) error {
	if cfg.Server.Host == "" {
		return fmt.Errorf("server host not configured; edit %s", config.Path())
	}

	root, repoName, branch, err := getRepoInfo()
	if err != nil {
		return err
	}

	maxSize, err := parseSize(opts.MaxUploadSize)
	if err != nil {
		return err
	}

	fmt.Printf("Preparing remote handoff...\n")
	fmt.Printf("- Repo:   %s\n", repoName)
	fmt.Printf("- Branch: %s\n", branch)

	store, err := instance.NewStore()
	if err != nil {
		return fmt.Errorf("open instance store: %w", err)
	}

	// Existing instance check
	instanceID, instanceName, err := resolveInstance(store, repoName)
	if err != nil {
		return err
	}
	if instanceID == "" && instanceName == "" {
		return nil // user cancelled
	}

	// File gathering and selection
	fmt.Printf("\nScanning files for remote continuation...\n")
	candidates, err := files.Gather(root)
	if err != nil {
		return fmt.Errorf("gather files: %w", err)
	}

	safe, secrets := files.FilterSecrets(candidates)
	fmt.Printf("- Git candidates: %d\n", len(candidates))

	selected, err := files.SelectWithClaude(ctx, safe)
	if err != nil {
		return err
	}
	fmt.Printf("- Claude selected: %d\n", len(selected))
	if len(secrets) > 0 {
		fmt.Printf("- Secrets excluded: %d\n", len(secrets))
	}
	fmt.Printf("\nSelected %d files for upload.\n", len(selected))

	// Confirm selection loop
	selected, ok := confirmSelection(ctx, selected, safe)
	if !ok {
		fmt.Println("Cancelled.")
		return nil
	}

	totalSize := files.TotalSize(selected)

	// Size limit handling
	if maxSize > 0 && totalSize > maxSize {
		selected = handleSizeExceeded(ctx, selected, totalSize, maxSize, safe)
		if selected == nil {
			fmt.Println("Cancelled.")
			return nil
		}
		totalSize = files.TotalSize(selected)
	}
	if maxSize == 0 && totalSize > files.DefaultMaxUploadSize {
		fmt.Printf("\nWARNING: Upload size is %s (no limit set).\n", files.FormatSize(totalSize))
	}

	// Pack archive
	var archiveBuf bytes.Buffer
	bar := progressbar.DefaultBytes(-1, "Packing")
	if err := files.Pack(root, selectedPaths(selected), io.MultiWriter(&archiveBuf, bar)); err != nil {
		return fmt.Errorf("pack: %w", err)
	}
	fmt.Println()

	// Connect and upload
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
		return fmt.Errorf("open SSH session: %w", err)
	}
	defer sess.Close()

	// Build wire payload: header JSON line + tar.gz bytes
	var payload bytes.Buffer
	if err := files.WriteHeader(&payload, files.UploadMeta{
		RepoName:     repoName,
		Branch:       branch,
		InstanceID:   instanceID,
		InstanceName: instanceName,
		PayloadSize:  int64(archiveBuf.Len()),
	}); err != nil {
		return err
	}
	payload.Write(archiveBuf.Bytes())
	sess.Stdin = &payload

	stdout, err := sess.StdoutPipe()
	if err != nil {
		return err
	}

	fmt.Printf("\nUploading files...\n")
	uploadBar := progressbar.DefaultBytes(int64(payload.Len()), "Uploading")

	if err := sess.Start("claude-remote-server upload"); err != nil {
		return fmt.Errorf("start remote upload: %w", err)
	}

	// Persist instance locally (creating state until server confirms running)
	inst := &instance.Instance{
		ID:         instanceID,
		Name:       instanceName,
		RepoName:   repoName,
		Branch:     branch,
		State:      instance.StateCreating,
		ServerHost: cfg.Server.Host,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	_ = store.Save(inst)
	_ = uploadBar.Finish()

	// Stream server messages
	fmt.Println()
	if err := remote.ReadMessages(stdout, func(msg remote.Message) error {
		switch msg.Type {
		case remote.MsgURL:
			var p remote.URLPayload
			if err := remote.DecodePayload(msg, &p); err == nil {
				// Use the server-assigned ID (generated server-side for security).
				if p.InstanceID != "" && p.InstanceID != inst.ID {
					_ = store.Delete(inst.ID)
					inst.ID = p.InstanceID
				}
				inst.RemoteURL = p.URL
				inst.State = instance.StateRunning
				_ = store.Save(inst)

				fmt.Printf("\nRemote instance is ready.\n")
				fmt.Printf("- Instance:       %s\n", instanceName)
				fmt.Printf("- Server:         %s\n", cfg.Server.Host)
				fmt.Printf("- Remote Control: %s\n\n", p.URL)
				fmt.Println("Attaching to live output...")
				fmt.Println("Press Ctrl+C to stop watching. Remote work will continue.")
			}
		case remote.MsgPort:
			var p remote.PortPayload
			if err := remote.DecodePayload(msg, &p); err == nil && p.RemotePort > 0 {
				localPort, err := client.ForwardPort(ctx, p.RemotePort, p.RemotePort)
				if err != nil {
					// Fallback: try any free port
					localPort, err = client.ForwardPort(ctx, p.RemotePort, 0)
				}
				if err == nil {
					fmt.Printf("SSH tunnel: localhost:%d → remote:%d (Claude Code CLI sync)\n",
						localPort, p.RemotePort)
				} else {
					fmt.Fprintf(os.Stderr, "Warning: port forward failed: %v\n", err)
				}
			}
		case remote.MsgLog:
			var p remote.LogPayload
			if err := remote.DecodePayload(msg, &p); err == nil {
				fmt.Println(p.Text)
			}
		case remote.MsgError:
			var p remote.ErrorPayload
			if err := remote.DecodePayload(msg, &p); err == nil {
				fmt.Fprintf(os.Stderr, "\nError: %s\n", p.Message)
			}
		case remote.MsgDone:
			return io.EOF // stop reading
		}
		return nil
	}); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "stream ended: %v\n", err)
	}

	_ = sess.Wait()

	fmt.Printf("\nStopped watching remote output.\n")
	fmt.Printf("The remote instance is still running.\n\n")
	fmt.Printf("Next actions:\n")
	fmt.Printf("  Reconnect:    claude-remote attach %s\n", instanceName)
	fmt.Printf("  Pull results: claude-remote pull %s\n", instanceName)
	return nil
}

// resolveInstance handles the "existing instance" prompt and returns (id, name, err).
// Returns ("", "", nil) if the user cancels.
func resolveInstance(store *instance.Store, repoName string) (id, name string, err error) {
	existing, err := store.ListByRepo(repoName)
	if err != nil {
		return "", "", err
	}
	var active []*instance.Instance
	for _, i := range existing {
		if !i.IsTerminal() {
			active = append(active, i)
		}
	}

	if len(active) > 0 {
		fmt.Printf("\nA remote workspace for %q already exists.\n\n", repoName)
		fmt.Printf("Choose how to continue:\n")
		fmt.Printf("1) Reuse existing instance (%s)\n", active[0].Name)
		fmt.Printf("2) Create new instance\n")
		fmt.Printf("3) Cancel\n> ")

		switch readChoice() {
		case "1":
			return active[0].ID, active[0].Name, nil
		case "2":
			// fall through to generate new
		default:
			return "", "", nil
		}
	}

	fmt.Print("\nEnter instance name (leave empty to auto-generate):\n> ")
	name = readLine()
	if name == "" {
		name = generateInstanceName(repoName)
	}
	return generateID(), name, nil
}

// confirmSelection presents the "choose how to continue" prompt and returns
// the final selection. Returns (nil, false) if the user cancels.
func confirmSelection(ctx context.Context, selected []files.FileInfo, safe []files.FileInfo) ([]files.FileInfo, bool) {
	for {
		fmt.Printf("\nChoose how to continue:\n")
		fmt.Printf("1) Continue\n")
		fmt.Printf("2) Review full file list\n")
		fmt.Printf("3) Re-run selection\n")
		fmt.Printf("4) Cancel\n> ")

		switch readChoice() {
		case "1":
			return selected, true
		case "2":
			fmt.Println("\nSelected files:")
			for _, f := range selected {
				fmt.Printf("  %s (%s)\n", f.Path, files.FormatSize(f.Size))
			}
		case "3":
			reselected, err := files.SelectWithClaude(ctx, safe)
			if err == nil {
				selected = reselected
				fmt.Printf("Re-selected %d files (%s).\n",
					len(selected), files.FormatSize(files.TotalSize(selected)))
			}
		default:
			return nil, false
		}
	}
}

// handleSizeExceeded presents the size-exceeded menu.
// Returns nil if the user cancels.
func handleSizeExceeded(ctx context.Context, selected []files.FileInfo, totalSize, maxSize int64, safe []files.FileInfo) []files.FileInfo {
	largest := files.LargestFiles(selected, 5)

	fmt.Printf("\nUpload size is %s.\n", files.FormatSize(totalSize))
	fmt.Printf("Default limit is %s.\n\n", files.FormatSize(maxSize))
	fmt.Printf("Largest files:\n")
	for _, f := range largest {
		fmt.Printf("  %s (%s)\n", f.Path, files.FormatSize(f.Size))
	}

	fmt.Printf("\nChoose how to continue:\n")
	fmt.Printf("1) Continue once\n")
	fmt.Printf("2) Exclude large files and retry\n")
	fmt.Printf("3) Re-run Claude selection with size limit\n")
	fmt.Printf("4) Cancel\n> ")

	switch readChoice() {
	case "1":
		return selected
	case "2":
		const threshold = 10 * 1024 * 1024
		filtered := files.ExcludeLarge(selected, threshold)
		fmt.Printf("Excluded %d large files. New size: %s\n",
			len(selected)-len(filtered), files.FormatSize(files.TotalSize(filtered)))
		return filtered
	case "3":
		limited := files.ExcludeLarge(safe, maxSize/10)
		reselected, err := files.SelectWithClaude(ctx, limited)
		if err == nil {
			return reselected
		}
		return selected
	default:
		return nil
	}
}

func selectedPaths(fis []files.FileInfo) []string {
	paths := make([]string, len(fis))
	for i, f := range fis {
		paths[i] = f.Path
	}
	return paths
}
