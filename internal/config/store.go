package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	DefaultAppViewURL       = "https://tangled.org"
	DefaultConstellationURL = "https://constellation.microcosm.blue"
	DefaultKnotHost         = "knot1.tangled.sh"
	LegacyKnotHost          = "tangled.org"
)

var ErrUnsupportedKey = errors.New("unsupported config key")
var ErrUnsupportedValue = errors.New("unsupported config value")

type Config struct {
	Knot          KnotConfig          `toml:"knot" json:"knot"`
	Constellation ConstellationConfig `toml:"constellation" json:"constellation"`
	AppView       AppViewConfig       `toml:"appview" json:"appview"`
	Clone         CloneConfig         `toml:"clone" json:"clone"`
	Remote        string              `toml:"remote,omitempty" json:"remote,omitempty"`

	path string
}

type KnotConfig struct {
	Hosts []string `toml:"hosts" json:"hosts"`
}

type ConstellationConfig struct {
	URL string `toml:"url" json:"url"`
}

type AppViewConfig struct {
	URL string `toml:"url" json:"url"`
}

type CloneConfig struct {
	Protocol string `toml:"protocol" json:"protocol"`
}

func Defaults() *Config {
	return &Config{
		Knot:          KnotConfig{Hosts: []string{DefaultKnotHost, LegacyKnotHost}},
		Constellation: ConstellationConfig{URL: DefaultConstellationURL},
		AppView:       AppViewConfig{URL: DefaultAppViewURL},
		Clone:         CloneConfig{Protocol: "https"},
	}
}

func Load() (*Config, error) {
	path, err := defaultPath()
	if err != nil {
		return nil, err
	}
	return LoadAt(path)
}

func LoadAt(path string) (*Config, error) {
	cfg := Defaults()
	cfg.path = path

	data, err := os.ReadFile(path) // #nosec G304 -- config path is user-controlled by design and only parsed as TOML.
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.applyDefaults()
	cfg.path = path
	return cfg, nil
}

func LoadProjectAt(repoRoot string) (*Config, error) {
	return LoadAt(filepath.Join(repoRoot, ".tangled", "config.toml"))
}

func (c *Config) Get(key string) (any, error) {
	switch key {
	case "knot.hosts":
		return append([]string(nil), c.Knot.Hosts...), nil
	case "constellation.url":
		if v := os.Getenv("TANG_CONSTELLATION_URL"); v != "" {
			return v, nil
		}
		return c.Constellation.URL, nil
	case "appview.url":
		return c.AppView.URL, nil
	case "clone.protocol":
		return c.Clone.Protocol, nil
	case "remote":
		return c.Remote, nil
	case "pds.url":
		return nil, fmt.Errorf("%w: pds.url is intentionally unsupported; PDS is resolved from DID documents", ErrUnsupportedKey)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedKey, key)
	}
}

func (c *Config) Set(key string, value string) error {
	switch key {
	case "knot.hosts":
		hosts := splitCSV(value)
		if len(hosts) == 0 {
			return fmt.Errorf("knot.hosts requires at least one host")
		}
		c.Knot.Hosts = hosts
	case "constellation.url":
		c.Constellation.URL = strings.TrimSpace(value)
	case "appview.url":
		c.AppView.URL = strings.TrimSpace(value)
	case "clone.protocol":
		protocol, err := normalizeCloneProtocol(value)
		if err != nil {
			return err
		}
		c.Clone.Protocol = protocol
	case "remote":
		c.Remote = strings.TrimSpace(value)
	case "pds.url":
		return fmt.Errorf("%w: pds.url is intentionally unsupported; PDS is resolved from DID documents", ErrUnsupportedKey)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedKey, key)
	}
	return c.Save()
}

func (c *Config) List() map[string]any {
	constellationURL := c.Constellation.URL
	if v := os.Getenv("TANG_CONSTELLATION_URL"); v != "" {
		constellationURL = v
	}
	return map[string]any{
		"knot.hosts":        append([]string(nil), c.Knot.Hosts...),
		"constellation.url": constellationURL,
		"appview.url":       c.AppView.URL,
		"clone.protocol":    c.Clone.Protocol,
		"remote":            c.Remote,
	}
}

func (c *Config) Save() error {
	if c.path == "" {
		path, err := defaultPath()
		if err != nil {
			return err
		}
		c.path = path
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o700); err != nil {
		return err
	}
	data, err := toml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0o600)
}

func (c *Config) Path() string {
	return c.path
}

func (c *Config) applyDefaults() {
	if len(c.Knot.Hosts) == 0 {
		c.Knot.Hosts = []string{DefaultKnotHost, LegacyKnotHost}
	}
	if c.Constellation.URL == "" {
		c.Constellation.URL = DefaultConstellationURL
	}
	if c.AppView.URL == "" {
		c.AppView.URL = DefaultAppViewURL
	}
	if c.Clone.Protocol == "" {
		c.Clone.Protocol = "https"
	}
}

func defaultPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "tang", "config.toml"), nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		item := strings.TrimSpace(part)
		item = strings.TrimPrefix(item, "https://")
		item = strings.TrimPrefix(item, "http://")
		item = strings.TrimSuffix(item, "/")
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func normalizeCloneProtocol(value string) (string, error) {
	protocol := strings.ToLower(strings.TrimSpace(value))
	switch protocol {
	case "ssh", "https":
		return protocol, nil
	default:
		return "", fmt.Errorf("%w: clone.protocol must be ssh or https", ErrUnsupportedValue)
	}
}
