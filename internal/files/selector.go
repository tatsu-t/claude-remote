package files

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

const DefaultMaxUploadSize = 200 * 1024 * 1024 // 200 MB

// FileInfo holds the relative path and size of a candidate file.
type FileInfo struct {
	Path string
	Size int64
}

// Gather collects upload candidates from a git repository:
// changed files, untracked files, and common config/context files.
func Gather(root string) ([]FileInfo, error) {
	out, err := exec.Command("git", "-C", root, "status", "--porcelain").Output()
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}

	seen := make(map[string]bool)
	var candidates []FileInfo

	for _, line := range strings.Split(string(out), "\n") {
		if len(line) < 4 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		// Handle rename: "old -> new"
		if idx := strings.Index(path, " -> "); idx >= 0 {
			path = path[idx+4:]
		}
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		full := filepath.Join(root, path)
		info, err := os.Lstat(full)
		if err != nil || info.IsDir() {
			continue
		}
		candidates = append(candidates, FileInfo{Path: path, Size: info.Size()})
	}

	// Always include common context files when present.
	for _, name := range []string{
		"go.mod", "go.sum", "package.json", "package-lock.json",
		"yarn.lock", "Cargo.toml", "requirements.txt", "pyproject.toml",
		"Makefile", "README.md", "README.txt", ".env.example",
	} {
		if seen[name] {
			continue
		}
		full := filepath.Join(root, name)
		info, err := os.Lstat(full)
		if err != nil || info.IsDir() {
			continue
		}
		seen[name] = true
		candidates = append(candidates, FileInfo{Path: name, Size: info.Size()})
	}

	return candidates, nil
}

var secretPatterns = []string{
	".env", "_secret", ".key", ".pem", ".p12", ".pfx",
	"credentials.json", "serviceaccount.json", ".aws/credentials",
}

// FilterSecrets removes files that look like secrets from candidates.
func FilterSecrets(candidates []FileInfo) (safe []FileInfo, excluded []string) {
	for _, f := range candidates {
		lower := strings.ToLower(filepath.Base(f.Path))
		secret := false
		for _, pat := range secretPatterns {
			if strings.Contains(lower, pat) {
				secret = true
				break
			}
		}
		if secret {
			excluded = append(excluded, f.Path)
		} else {
			safe = append(safe, f)
		}
	}
	return
}

// SelectWithClaude asks Claude Haiku to choose the files needed for continuation.
// Falls back to all candidates if the API call fails.
func SelectWithClaude(ctx context.Context, candidates []FileInfo) ([]FileInfo, error) {
	if len(candidates) == 0 {
		return candidates, nil
	}

	var list strings.Builder
	for i, f := range candidates {
		fmt.Fprintf(&list, "%d. %s (%s)\n", i+1, f.Path, FormatSize(f.Size))
	}

	prompt := fmt.Sprintf(`You are selecting files to transfer for a remote Claude Code session continuation.

Candidate files:
%s
Select only the files truly needed to continue work remotely. Prefer:
- Source files currently being modified
- Config files that define the project structure
- Context files like README

Exclude large build artifacts, generated files, and files unlikely to affect continuity.

Respond with a JSON array of file paths exactly as listed above. Example:
["src/main.go", "go.mod"]`, list.String())

	client := anthropic.NewClient()
	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_5HaikuLatest,
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		// Fallback: return all candidates so upload can still proceed.
		fmt.Fprintf(os.Stderr, "Claude file selection unavailable (%v); using all candidates.\n", err)
		return candidates, nil
	}

	if len(msg.Content) == 0 {
		return candidates, nil
	}

	text := msg.Content[0].Text
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start < 0 || end <= start {
		return candidates, nil
	}

	var selected []string
	if err := json.Unmarshal([]byte(text[start:end+1]), &selected); err != nil {
		return candidates, nil
	}

	selectedSet := make(map[string]bool, len(selected))
	for _, s := range selected {
		selectedSet[s] = true
	}

	var result []FileInfo
	for _, f := range candidates {
		if selectedSet[f.Path] {
			result = append(result, f)
		}
	}
	if len(result) == 0 {
		return candidates, nil
	}
	return result, nil
}

// TotalSize returns the sum of all file sizes.
func TotalSize(files []FileInfo) int64 {
	var total int64
	for _, f := range files {
		total += f.Size
	}
	return total
}

// LargestFiles returns the n largest files (descending by size).
func LargestFiles(files []FileInfo, n int) []FileInfo {
	sorted := make([]FileInfo, len(files))
	copy(sorted, files)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Size > sorted[i].Size {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	if n > len(sorted) {
		n = len(sorted)
	}
	return sorted[:n]
}

// ExcludeLarge removes files whose size exceeds threshold.
func ExcludeLarge(files []FileInfo, threshold int64) []FileInfo {
	var result []FileInfo
	for _, f := range files {
		if f.Size <= threshold {
			result = append(result, f)
		}
	}
	return result
}

// FormatSize returns a human-readable file size string.
func FormatSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.1f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.1f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.1f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}
