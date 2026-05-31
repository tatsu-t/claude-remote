package cli

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var stdin = bufio.NewReader(os.Stdin)

func readLine() string {
	line, _ := stdin.ReadString('\n')
	return strings.TrimRight(line, "\r\n")
}

func readChoice() string {
	return strings.TrimSpace(readLine())
}

// getRepoInfo returns (rootDir, repoName, branch, error).
func getRepoInfo() (root, repoName, branch string, err error) {
	rootBytes, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", "", "", fmt.Errorf("not a git repository (or git not installed): %w", err)
	}
	root = strings.TrimSpace(string(rootBytes))
	repoName = filepath.Base(root)

	branchBytes, err := exec.Command("git", "-C", root, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		branch = "HEAD"
	} else {
		branch = strings.TrimSpace(string(branchBytes))
	}
	return root, repoName, branch, nil
}

func generateID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%d", b, time.Now().Unix()%10000)
}

func generateInstanceName(repoName string) string {
	n, _ := rand.Int(rand.Reader, big.NewInt(9000))
	suffix := int(n.Int64()) + 1000
	return fmt.Sprintf("%s-%d", sanitizeName(repoName), suffix)
}

func sanitizeName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	name := strings.Trim(b.String(), "-")
	if len(name) > 20 {
		name = name[:20]
	}
	return name
}

// parseSize converts strings like "200MB", "1GB", "0" to bytes.
// Returns 0 for unlimited ("0").
func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "0" {
		return 0, nil
	}
	upper := strings.ToUpper(s)
	switch {
	case strings.HasSuffix(upper, "GB"):
		n, err := strconv.ParseInt(strings.TrimSuffix(upper, "GB"), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid size %q", s)
		}
		return n * 1024 * 1024 * 1024, nil
	case strings.HasSuffix(upper, "MB"):
		n, err := strconv.ParseInt(strings.TrimSuffix(upper, "MB"), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid size %q", s)
		}
		return n * 1024 * 1024, nil
	case strings.HasSuffix(upper, "KB"):
		n, err := strconv.ParseInt(strings.TrimSuffix(upper, "KB"), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid size %q", s)
		}
		return n * 1024, nil
	default:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid size %q", s)
		}
		return n, nil
	}
}

