package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Brook-sys/auxitalk/pkg/types"
)

var envPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

type Mode string

const (
	ModeDev    Mode = "dev"
	ModeLocal  Mode = "local"
	ModeStrict Mode = "strict"
)

type Duration time.Duration

func (d Duration) Std() time.Duration {
	return time.Duration(d)
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("duration must be a string: %w", err)
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", value, err)
	}

	*d = Duration(parsed)
	return nil
}

type Runtime struct {
	RequestTimeout     Duration `json:"requestTimeout"`
	HealthTimeout      Duration `json:"healthTimeout"`
	MaxPayloadSize     int64    `json:"maxPayloadSize"`
	MaxEventsPerSecond int      `json:"maxEventsPerSecond"`
}

type Storage struct {
	SQLitePath string `json:"sqlitePath,omitempty"`
}

type Plugin struct {
	Manifest string                `json:"manifest,omitempty"`
	Enabled  bool                  `json:"enabled"`
	Env      map[string]string     `json:"env,omitempty"`
	Config   map[string]any        `json:"config,omitempty"`
	Inline   *types.PluginManifest `json:"inline,omitempty"`
}

type Config struct {
	Mode      Mode             `json:"mode"`
	Runtime   Runtime          `json:"runtime"`
	Storage   Storage          `json:"storage,omitempty"`
	Plugins   []Plugin         `json:"plugins"`
	Workflows []types.Workflow `json:"workflows,omitempty"`
}

func Default() Config {
	return Config{
		Mode: ModeDev,
		Runtime: Runtime{
			RequestTimeout:     Duration(10 * time.Second),
			HealthTimeout:      Duration(2 * time.Second),
			MaxPayloadSize:     1024 * 1024,
			MaxEventsPerSecond: 50,
		},
		Plugins:   []Plugin{},
		Workflows: []types.Workflow{},
	}
}

func Load(path string) (Config, error) {
	if strings.TrimSpace(path) == "" {
		cfg := Default()
		return cfg, cfg.Validate()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Plugins == nil {
		cfg.Plugins = []Plugin{}
	}
	if cfg.Workflows == nil {
		cfg.Workflows = []types.Workflow{}
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	switch c.Mode {
	case ModeDev, ModeLocal, ModeStrict:
	default:
		return errors.New("config mode is invalid")
	}

	if c.Runtime.RequestTimeout.Std() <= 0 {
		return errors.New("runtime requestTimeout must be greater than zero")
	}
	if c.Runtime.HealthTimeout.Std() <= 0 {
		return errors.New("runtime healthTimeout must be greater than zero")
	}
	if c.Runtime.MaxPayloadSize <= 0 {
		return errors.New("runtime maxPayloadSize must be greater than zero")
	}
	if c.Runtime.MaxEventsPerSecond <= 0 {
		return errors.New("runtime maxEventsPerSecond must be greater than zero")
	}

	for _, plugin := range c.Plugins {
		if err := plugin.Validate(); err != nil {
			return err
		}
	}
	for _, workflow := range c.Workflows {
		if err := workflow.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (p Plugin) Validate() error {
	if strings.TrimSpace(p.Manifest) == "" && p.Inline == nil {
		return errors.New("plugin manifest or inline manifest is required")
	}
	for key := range p.Env {
		if strings.TrimSpace(key) == "" {
			return errors.New("plugin env key cannot be empty")
		}
	}
	if p.Inline != nil {
		if err := p.Inline.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (p Plugin) ResolvedEnv(lookup func(string) (string, bool)) []string {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	env := make([]string, 0, len(p.Env))
	for key, value := range p.Env {
		resolved := envPattern.ReplaceAllStringFunc(value, func(match string) string {
			parts := envPattern.FindStringSubmatch(match)
			if len(parts) != 2 {
				return match
			}
			if resolved, ok := lookup(parts[1]); ok {
				return resolved
			}
			return ""
		})
		env = append(env, key+"="+resolved)
	}
	return env
}
