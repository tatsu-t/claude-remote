package workspace

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"time"
)

var remoteControlURLRe = regexp.MustCompile(`https://claude\.ai/code/[^\s"']+`)

// Session represents a running claude --server-mode process.
type Session struct {
	cmd      *exec.Cmd
	URL      string
	LogLines <-chan string
	Done     <-chan error
}

// StartClaude launches claude in server mode inside workspaceDir and returns
// a Session once the Remote Control URL appears on stdout (or after timeout).
func StartClaude(workspaceDir string) (*Session, error) {
	cmd := exec.Command("claude", "--server-mode", "--project-dir", workspaceDir)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start claude: %w", err)
	}

	logCh := make(chan string, 512)
	doneCh := make(chan error, 1)
	urlCh := make(chan string, 1)

	session := &Session{
		cmd:      cmd,
		LogLines: logCh,
		Done:     doneCh,
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			logCh <- line
			if session.URL == "" {
				if match := remoteControlURLRe.FindString(line); match != "" {
					select {
					case urlCh <- match:
					default:
					}
				}
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			logCh <- "[stderr] " + scanner.Text()
		}
	}()

	go func() {
		doneCh <- cmd.Wait()
		close(logCh)
	}()

	// Wait up to 30 s for the Remote Control URL to appear.
	select {
	case url := <-urlCh:
		session.URL = url
	case err := <-doneCh:
		return nil, fmt.Errorf("claude exited before producing URL: %w", err)
	case <-time.After(30 * time.Second):
		// Proceed without URL; the handler will still stream logs.
	}

	return session, nil
}

func (s *Session) Stop() error {
	if s.cmd != nil && s.cmd.Process != nil {
		return s.cmd.Process.Kill()
	}
	return nil
}
