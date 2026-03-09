package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Version string           `json:"version"`
	Aliases map[string]Alias `json:"aliases"`
}

type Alias struct {
	URL       string `json:"url"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

func configPath() string {
	return filepath.Join(configDir, "config.json")
}

func loadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if os.IsNotExist(err) {
		return &Config{Version: "10", Aliases: make(map[string]Alias)}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Aliases == nil {
		cfg.Aliases = make(map[string]Alias)
	}
	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0600)
}

func getAlias(name string) (Alias, error) {
	cfg, err := loadConfig()
	if err != nil {
		return Alias{}, fmt.Errorf("loading config: %w", err)
	}
	a, ok := cfg.Aliases[name]
	if !ok {
		return Alias{}, fmt.Errorf("alias %q not found; run 'mc alias set %s <url> <user> <password>'", name, name)
	}
	return a, nil
}

// parseAlias extracts "alias" from "alias[/bucket[/prefix]]".
// Returns (aliasName, rest) where rest is everything after the first slash.
func parseAlias(target string) (aliasName, rest string) {
	i := strings.IndexByte(target, '/')
	if i < 0 {
		return target, ""
	}
	return target[:i], target[i+1:]
}

// aliasEndpoint returns (host:port, useSSL) from an alias URL.
func aliasEndpoint(a Alias) (endpoint string, useSSL bool, err error) {
	u, err := url.Parse(a.URL)
	if err != nil {
		return "", false, fmt.Errorf("invalid alias URL %q: %w", a.URL, err)
	}
	useSSL = u.Scheme == "https"
	endpoint = u.Host
	return
}
