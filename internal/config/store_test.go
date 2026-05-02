package config

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadAtReturnsDefaultsWhenMissing(t *testing.T) {
	cfg, err := LoadAt(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatalf("LoadAt returned error: %v", err)
	}
	if !reflect.DeepEqual(cfg.Knot.Hosts, []string{"tangled.org"}) {
		t.Fatalf("default knot hosts = %#v", cfg.Knot.Hosts)
	}
	if cfg.Constellation.URL != DefaultConstellationURL {
		t.Fatalf("default constellation URL = %q", cfg.Constellation.URL)
	}
	if cfg.AppView.URL != DefaultAppViewURL {
		t.Fatalf("default appview URL = %q", cfg.AppView.URL)
	}
}

func TestSetAndReloadSupportedKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	cfg, err := LoadAt(path)
	if err != nil {
		t.Fatalf("LoadAt returned error: %v", err)
	}
	if err := cfg.Set("knot.hosts", "tangled.org, knot.example.com,https://knot.example.com/"); err != nil {
		t.Fatalf("Set knot.hosts returned error: %v", err)
	}
	if err := cfg.Set("constellation.url", "https://constellation.example.com"); err != nil {
		t.Fatalf("Set constellation.url returned error: %v", err)
	}

	reloaded, err := LoadAt(path)
	if err != nil {
		t.Fatalf("reload returned error: %v", err)
	}
	if !reflect.DeepEqual(reloaded.Knot.Hosts, []string{"tangled.org", "knot.example.com"}) {
		t.Fatalf("reloaded knot hosts = %#v", reloaded.Knot.Hosts)
	}
	if reloaded.Constellation.URL != "https://constellation.example.com" {
		t.Fatalf("reloaded constellation URL = %q", reloaded.Constellation.URL)
	}
}

func TestPDSURLIsUnsupported(t *testing.T) {
	cfg, err := LoadAt(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatalf("LoadAt returned error: %v", err)
	}
	if err := cfg.Set("pds.url", "https://example.com"); !errors.Is(err, ErrUnsupportedKey) {
		t.Fatalf("Set pds.url error = %v", err)
	}
}
