package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".ssht", "config.json"), nil
}

func Load() (*Config, error) {
	cfgPath, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := defaultConfig()
			if err := Save(cfg); err != nil {
				return nil, err
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		corruptPath := cfgPath + ".corrupt"
		if writeErr := os.WriteFile(corruptPath, data, 0o600); writeErr != nil {
			return nil, fmt.Errorf("invalid config json (%v) and failed to save corrupt backup: %w", err, writeErr)
		}
		newCfg := defaultConfig()
		if saveErr := Save(newCfg); saveErr != nil {
			return nil, fmt.Errorf("invalid config json (%v) and failed to recreate default config: %w", err, saveErr)
		}
		return newCfg, nil
	}

	normalize(&cfg)
	return &cfg, nil
}

func Save(cfg *Config) error {
	normalize(cfg)
	cfgPath, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	if current, readErr := os.ReadFile(cfgPath); readErr == nil {
		if err := os.WriteFile(cfgPath+".bak", current, 0o600); err != nil {
			return fmt.Errorf("write backup config: %w", err)
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	tmpPath := cfgPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := os.Rename(tmpPath, cfgPath); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

func defaultConfig() *Config {
	return &Config{
		LastProfile: "default",
		Profiles: map[string]Profile{
			"default": {
				Name:    "default",
				Servers: []Server{},
			},
		},
	}
}

func normalize(cfg *Config) {
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	if cfg.LastProfile == "" {
		for name := range cfg.Profiles {
			cfg.LastProfile = name
			break
		}
	}
	if cfg.LastProfile == "" {
		cfg.LastProfile = "default"
	}
	if _, ok := cfg.Profiles[cfg.LastProfile]; !ok {
		cfg.Profiles[cfg.LastProfile] = Profile{Name: cfg.LastProfile, Servers: []Server{}}
	}
	for name, profile := range cfg.Profiles {
		if profile.Name == "" {
			profile.Name = name
		}
		if profile.Servers == nil {
			profile.Servers = []Server{}
		}
		cfg.Profiles[name] = profile
	}
}
