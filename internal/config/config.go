// Package config loads `.sevro.yaml` for the CLI.
//
// The config file lets users persist common flag combinations (e.g.
// "always exclude this detector", "always set --severity=med") so a
// `sevro analyze` invocation in a known repo behaves consistently
// without flag-soup. Flags still override config when supplied.
//
// Lookup order (first match wins):
//
//  1. --config <path>
//  2. SEVRO_CONFIG env var
//  3. ./.sevro.yaml in the current working directory
//  4. zero value (no config)
package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the on-disk schema. Add fields in additive-only fashion
// — old configs must keep loading after schema growth.
type Config struct {
	// MinSeverity is the default --severity flag.
	MinSeverity string `yaml:"min_severity,omitempty"`
	// Detectors is the default --detector allow-list.
	Detectors []string `yaml:"detectors,omitempty"`
	// FailOn is the default --fail-on threshold.
	FailOn string `yaml:"fail_on,omitempty"`
	// NoColor disables ANSI output everywhere.
	NoColor bool `yaml:"no_color,omitempty"`
}

// ConfigName is the conventional filename. Hidden so it doesn't
// clutter `ls`.
const ConfigName = ".sevro.yaml"

// Load resolves and reads the config. Returns the zero Config when no
// file is present (which is the safe default — users opt in by
// creating one). Returns an error only when a file is named
// explicitly via --config or SEVRO_CONFIG and that file fails to
// load.
func Load(explicit string) (Config, error) {
	if explicit != "" {
		return readFile(explicit)
	}
	if env := os.Getenv("SEVRO_CONFIG"); env != "" {
		return readFile(env)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return Config{}, nil
	}
	candidate := filepath.Join(cwd, ConfigName)
	if _, err := os.Stat(candidate); err == nil {
		return readFile(candidate)
	}
	return Config{}, nil
}

func readFile(path string) (Config, error) {
	f, err := os.Open(path) //nolint:gosec // user-specified config path
	if err != nil {
		return Config{}, fmt.Errorf("config: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return Decode(f)
}

// Decode reads a YAML config from r.
func Decode(r io.Reader) (Config, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return Config{}, fmt.Errorf("config: read: %w", err)
	}
	if len(raw) == 0 {
		return Config{}, nil
	}
	var c Config
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("config: yaml: %w", err)
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Validate checks the config for known-good values.
func (c Config) Validate() error {
	for _, key := range []struct {
		name, value string
	}{
		{"min_severity", c.MinSeverity},
		{"fail_on", c.FailOn},
	} {
		if key.value == "" {
			continue
		}
		switch toLower(key.value) {
		case "low", "med", "medium", "high":
			// ok
		default:
			return fmt.Errorf("config: %s must be low|med|high (got %q)", key.name, key.value)
		}
	}
	return nil
}

// ErrNotFound is returned by Load when an explicit config path was
// supplied but did not resolve to a file. Default Load (no explicit
// path) does not return this — it returns a zero Config.
var ErrNotFound = errors.New("config: file not found")

func toLower(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}
