package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server ServerConfig `toml:"server"`
}

type ServerConfig struct {
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
	User    string `toml:"user"`
	KeyPath string `toml:"key_path"`
	DataDir string `toml:"data_dir"`
}

func Load() (*Config, error) {
	path := Path()
	cfg := Default()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func Save(cfg *Config) error {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port:    22,
			User:    "claude-remote",
			DataDir: "/var/lib/claude-remote",
		},
	}
}

func Path() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude-remote", "config.toml")
}
