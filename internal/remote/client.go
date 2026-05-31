package remote

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Client wraps an SSH connection to the claude-remote server.
type Client struct {
	conn *ssh.Client
}

// Config holds connection parameters.
type Config struct {
	Host    string
	Port    int
	User    string
	KeyPath string
}

func NewClient(cfg Config) (*Client, error) {
	signer, err := loadSigner(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("load SSH key: %w", err)
	}

	knownHostsFile, err := defaultKnownHostsFile()
	if err != nil {
		return nil, fmt.Errorf("resolve known_hosts path: %w", err)
	}

	clientConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: tofuHostKeyCallback(knownHostsFile),
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	conn, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH dial %s: %w", addr, err)
	}

	return &Client{conn: conn}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) NewSession() (*ssh.Session, error) {
	return c.conn.NewSession()
}

func loadSigner(keyPath string) (ssh.Signer, error) {
	if keyPath == "" {
		var err error
		keyPath, err = findDefaultKey()
		if err != nil {
			return nil, err
		}
	}
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(data)
}

func findDefaultKey() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	for _, name := range []string{"id_ed25519", "id_ecdsa", "id_rsa"} {
		p := filepath.Join(home, ".ssh", name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no SSH private key found in ~/.ssh/; set key_path in config")
}

func defaultKnownHostsFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", "known_hosts"), nil
}

// tofuHostKeyCallback implements Trust-On-First-Use host key verification:
//   - If the host is already in known_hosts, verify the key matches exactly.
//   - If the key has CHANGED for a known host, reject with a clear MITM warning.
//   - If the host is not yet known, append the key to known_hosts and warn the user.
func tofuHostKeyCallback(knownHostsFile string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if _, err := os.Stat(knownHostsFile); err == nil {
			checker, err := knownhosts.New(knownHostsFile)
			if err != nil {
				return fmt.Errorf("parse known_hosts: %w", err)
			}
			verifyErr := checker(hostname, remote, key)
			if verifyErr == nil {
				return nil // host known and key matches
			}
			var keyErr *knownhosts.KeyError
			if errors.As(verifyErr, &keyErr) && len(keyErr.Want) > 0 {
				// Host is known but key is DIFFERENT — possible MITM.
				return fmt.Errorf(
					"WARNING: remote host identification has changed for %s!\n"+
						"Possible MITM attack. If the server key was intentionally rotated,\n"+
						"remove the old entry from %s and retry.",
					hostname, knownHostsFile,
				)
			}
			// Host not yet in file (Want is empty) — fall through to TOFU append.
		}

		// First time seeing this host: append to known_hosts.
		fmt.Fprintf(os.Stderr, "Warning: permanently added %q (%s) to known hosts.\n",
			hostname, key.Type())
		return appendKnownHost(knownHostsFile, hostname, key)
	}
}

func appendKnownHost(path, hostname string, key ssh.PublicKey) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create .ssh directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open known_hosts for writing: %w", err)
	}
	defer f.Close()
	line := knownhosts.Line([]string{hostname}, key)
	_, err = fmt.Fprintln(f, line)
	return err
}
