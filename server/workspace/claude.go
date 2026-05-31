package workspace

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"time"
)

var remoteControlURLRe = regexp.MustCompile(`https://claude\.ai/code/[^\s"']+`)

// localPortRe matches "localhost:PORT" or "127.0.0.1:PORT" in stdout,
// which is how Claude Code server mode advertises its local IPC port.
var localPortRe = regexp.MustCompile(`(?:localhost|127\.0\.0\.1):(\d{2,5})\b`)

// Session represents a running claude --server-mode process.
type Session struct {
	cmd       *exec.Cmd
	URL       string
	LocalPort int
	LogLines  <-chan string
	Done      <-chan error
}

// StartClaude launches claude in server mode inside workspaceDir and returns
// a Session once startup signals appear on stdout (or after 30 s timeout).
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
	portCh := make(chan int, 1)

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
			if session.LocalPort == 0 {
				if m := localPortRe.FindStringSubmatch(line); len(m) == 2 {
					if p, err := strconv.Atoi(m[1]); err == nil && p > 0 && p < 65536 {
						select {
						case portCh <- p:
						default:
						}
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

	// Wait up to 30 s for the Remote Control URL (required for usability).
	select {
	case url := <-urlCh:
		session.URL = url
	case exitErr := <-doneCh:
		return nil, fmt.Errorf("claude exited before producing URL: %w", exitErr)
	case <-time.After(30 * time.Second):
		// Proceed without URL; the handler will still stream logs.
	}

	// Give a short grace period for the local port to appear after the URL.
	if session.LocalPort == 0 {
		select {
		case port := <-portCh:
			session.LocalPort = port
		case <-time.After(2 * time.Second):
			// Port not found; SSH forwarding will be unavailable.
		}
	}

	return session, nil
}

func (s *Session) Stop() error {
	if s.cmd != nil && s.cmd.Process != nil {
		return s.cmd.Process.Kill()
	}
	return nil
}
