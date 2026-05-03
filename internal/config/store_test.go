package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadAtReturnsDefaultsWhenMissing(t *testing.T) {
	cfg, err := LoadAt(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatalf("LoadAt returned error: %v", err)
	}
	if !reflect.DeepEqual(cfg.Knot.Hosts, []string{"knot1.tangled.sh", "tangled.org"}) {
		t.Fatalf("default knot hosts = %#v", cfg.Knot.Hosts)
	}
	if cfg.Clone.Protocol != "https" {
		t.Fatalf("default clone protocol = %q", cfg.Clone.Protocol)
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
	if err := cfg.Set("clone.protocol", "ssh"); err != nil {
		t.Fatalf("Set clone.protocol returned error: %v", err)
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
	if reloaded.Clone.Protocol != "ssh" {
		t.Fatalf("reloaded clone protocol = %q", reloaded.Clone.Protocol)
	}
}

func TestCloneProtocolValidation(t *testing.T) {
	cfg, err := LoadAt(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatalf("LoadAt returned error: %v", err)
	}
	if err := cfg.Set("clone.protocol", "git"); !errors.Is(err, ErrUnsupportedValue) {
		t.Fatalf("Set clone.protocol invalid error = %v", err)
	}
}

func TestLoadAtAppliesDefaultsToPartialConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[appview]\nurl = \"https://app.example.com\"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	cfg, err := LoadAt(path)
	if err != nil {
		t.Fatalf("LoadAt returned error: %v", err)
	}
	if !reflect.DeepEqual(cfg.Knot.Hosts, []string{DefaultKnotHost, LegacyKnotHost}) {
		t.Fatalf("default knot hosts = %#v", cfg.Knot.Hosts)
	}
	if cfg.Constellation.URL != DefaultConstellationURL {
		t.Fatalf("constellation URL = %q", cfg.Constellation.URL)
	}
	if cfg.Clone.Protocol != "https" {
		t.Fatalf("clone protocol = %q", cfg.Clone.Protocol)
	}
	if cfg.AppView.URL != "https://app.example.com" {
		t.Fatalf("appview URL = %q", cfg.AppView.URL)
	}
}

func TestLoadAtRejectsInvalidTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[invalid"), 0o600); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	if _, err := LoadAt(path); err == nil {
		t.Fatal("expected invalid TOML error")
	}
}

func TestGetListAndPath(t *testing.T) {
	t.Setenv("TANG_CONSTELLATION_URL", "https://constellation.env.example.com")
	path := filepath.Join(t.TempDir(), "config.toml")
	cfg, err := LoadAt(path)
	if err != nil {
		t.Fatalf("LoadAt returned error: %v", err)
	}
	cfg.Remote = "upstream"
	if cfg.Path() != path {
		t.Fatalf("Path = %q", cfg.Path())
	}
	got, err := cfg.Get("constellation.url")
	if err != nil {
		t.Fatalf("Get constellation.url returned error: %v", err)
	}
	if got != "https://constellation.env.example.com" {
		t.Fatalf("constellation.url = %#v", got)
	}
	list := cfg.List()
	if list["constellation.url"] != "https://constellation.env.example.com" || list["remote"] != "upstream" {
		t.Fatalf("List = %#v", list)
	}
	for key, want := range map[string]any{
		"knot.hosts":     []string{DefaultKnotHost, LegacyKnotHost},
		"appview.url":    DefaultAppViewURL,
		"clone.protocol": "https",
		"remote":         "upstream",
	} {
		got, err := cfg.Get(key)
		if err != nil {
			t.Fatalf("Get %s returned error: %v", key, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Get %s = %#v", key, got)
		}
	}
	if _, err := cfg.Get("missing.key"); !errors.Is(err, ErrUnsupportedKey) {
		t.Fatalf("Get unsupported error = %v", err)
	}
	if _, err := cfg.Get("pds.url"); !errors.Is(err, ErrUnsupportedKey) {
		t.Fatalf("Get pds.url error = %v", err)
	}
}

func TestLoadAndLoadProjectAtUseExpectedPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error = %v", err)
	}
	if cfg.Path() == "" {
		t.Fatal("Load path is empty")
	}

	project := t.TempDir()
	projectCfg, err := LoadProjectAt(project)
	if err != nil {
		t.Fatalf("LoadProjectAt error = %v", err)
	}
	if projectCfg.Path() != filepath.Join(project, ".tangled", "config.toml") {
		t.Fatalf("project path = %q", projectCfg.Path())
	}
}

func TestSaveUsesDefaultPathWhenUnset(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	cfg := Defaults()
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save error = %v", err)
	}
	if cfg.Path() == "" {
		t.Fatal("Save did not set path")
	}
	if _, err := os.Stat(cfg.Path()); err != nil {
		t.Fatalf("saved config stat error = %v", err)
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
	if err := cfg.Set("missing.key", "value"); !errors.Is(err, ErrUnsupportedKey) {
		t.Fatalf("Set missing.key error = %v", err)
	}
	if err := cfg.Set("knot.hosts", " , "); err == nil {
		t.Fatal("expected empty knot.hosts error")
	}
	if err := cfg.Set("appview.url", "https://app.example.com"); err != nil {
		t.Fatalf("Set appview.url error = %v", err)
	}
	if err := cfg.Set("remote", "upstream"); err != nil {
		t.Fatalf("Set remote error = %v", err)
	}
}
