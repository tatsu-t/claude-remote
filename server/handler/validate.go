package handler

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	reInstanceID   = regexp.MustCompile(`^[a-f0-9]{6,16}-[0-9]{1,6}$`)
	reInstanceName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)
	reRepoName     = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)
	reBranchPart   = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

func validateInstanceID(id string) error {
	if !reInstanceID.MatchString(id) {
		return fmt.Errorf("invalid instance ID %q: must match [a-f0-9]{6,16}-[0-9]{1,6}", id)
	}
	return nil
}

func validateInstanceName(name string) error {
	if !reInstanceName.MatchString(name) {
		return fmt.Errorf("invalid instance name %q: must be lowercase alphanumeric/dash, 1-32 chars", name)
	}
	return nil
}

func validateRepoName(name string) error {
	if name == "" {
		return nil
	}
	if !reRepoName.MatchString(name) {
		return fmt.Errorf("invalid repo name %q", name)
	}
	return nil
}

// validateBranch allows "/" for namespaced branches (e.g. "feature/foo") but
// rejects ".." segments and other path traversal sequences.
func validateBranch(branch string) error {
	if branch == "" {
		return nil
	}
	if len(branch) > 200 {
		return fmt.Errorf("branch name too long")
	}
	for _, segment := range strings.Split(branch, "/") {
		if segment == ".." || segment == "." || !reBranchPart.MatchString(segment) {
			return fmt.Errorf("invalid branch %q", branch)
		}
	}
	return nil
}

// serverGenerateID creates a unique instance ID entirely on the server side.
// Format matches the client pattern so local stores can hold both.
func serverGenerateID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%d", b, time.Now().Unix()%10000)
}
