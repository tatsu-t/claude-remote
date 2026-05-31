package remote

import (
	"fmt"
	"net"
	"os"

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

	hostKeyCallback, err := buildHostKeyCallback()
	if err != nil {
		return nil, fmt.Errorf("host key callback: %w", err)
	}

	clientConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: hostKeyCallback,
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
		p := fmt.Sprintf("%s/.ssh/%s", home, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no SSH private key found in ~/.ssh/; set key_path in config")
}

func buildHostKeyCallback() (ssh.HostKeyCallback, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	knownHostsFile := fmt.Sprintf("%s/.ssh/known_hosts", home)
	if _, err := os.Stat(knownHostsFile); os.IsNotExist(err) {
		// TOFU warning: first connection accepts any key.
		// Users should run ssh-keyscan manually for production use.
		fmt.Fprintf(os.Stderr, "WARNING: ~/.ssh/known_hosts not found; accepting server key without verification.\n")
		return ssh.InsecureIgnoreHostKey(), nil //nolint:gosec
	}
	return knownhosts.New(knownHostsFile)
}
