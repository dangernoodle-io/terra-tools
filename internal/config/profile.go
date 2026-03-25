package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Profile holds per-profile configuration.
type Profile struct {
	Bind []string   `yaml:"bind,omitempty"`
	Lint LintConfig `yaml:"lint,omitempty"`
}

// GlobalConfig is the schema for ~/.config/terranoodle/config.yml.
type GlobalConfig struct {
	Lint     LintConfig         `yaml:"lint,omitempty"`
	Profiles map[string]Profile `yaml:"profiles,omitempty"`
}

// LoadGlobal reads a global config file.
func LoadGlobal(path string) (*GlobalConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %q: %w", path, err)
	}
	return &cfg, nil
}

// SaveGlobal writes a global config file.
func SaveGlobal(path string, cfg *GlobalConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("config: create dir %q: %w", dir, err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// DefaultGlobal returns a GlobalConfig with a default profile containing built-in rules.
func DefaultGlobal() *GlobalConfig {
	return &GlobalConfig{
		Profiles: map[string]Profile{
			"default": {
				Lint: Default().Lint,
			},
		},
	}
}

// MatchProfile returns the name of the first profile whose bind paths match cwd.
// Returns "" if no profile matches (default will be used).
// The "default" profile is skipped — it's the fallback.
func MatchProfile(cfg *GlobalConfig, cwd string) string {
	if cfg == nil || len(cfg.Profiles) == 0 {
		return ""
	}

	// Sort profile names for deterministic matching
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		if name == "default" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		profile := cfg.Profiles[name]
		for _, bind := range profile.Bind {
			if matchBind(cwd, bind) {
				return name
			}
		}
	}
	return ""
}

// matchBind checks if cwd matches a bind path.
// If bind contains glob chars (*, ?, [), uses filepath.Match.
// Otherwise uses prefix matching (cwd starts with bind path).
func matchBind(cwd, bind string) bool {
	if containsGlob(bind) {
		matched, _ := filepath.Match(bind, cwd)
		return matched
	}
	// Prefix match: cwd equals bind or is a subdirectory
	return cwd == bind || strings.HasPrefix(cwd, bind+string(filepath.Separator))
}

// containsGlob reports whether s contains glob metacharacters.
func containsGlob(s string) bool {
	return strings.ContainsAny(s, "*?[")
}
