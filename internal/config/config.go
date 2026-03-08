package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".ssht"), nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func ProfilesDir() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "profiles"), nil
}

func Load() (*Config, error) {
	cfgPath, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}

	pDir, err := ProfilesDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(pDir, 0o700); err != nil {
		return nil, fmt.Errorf("create profiles dir: %w", err)
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
		return nil, fmt.Errorf("parse config.json: %w", err)
	}

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}

	files, err := os.ReadDir(pDir)
	if err == nil {
		for _, f := range files {
			if !f.IsDir() && filepath.Ext(f.Name()) == ".json" {
				pPath := filepath.Join(pDir, f.Name())
				pData, err := os.ReadFile(pPath)
				if err != nil {
					continue
				}
				var p Profile
				if err := json.Unmarshal(pData, &p); err == nil {
					name := strings.TrimSuffix(f.Name(), ".json")
					if p.Name == "" {
						p.Name = name
					}
					cfg.Profiles[name] = p
				}
			}
		}
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
	dir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	pDir, err := ProfilesDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(pDir, 0o700); err != nil {
		return fmt.Errorf("create profiles dir: %w", err)
	}

	globalCfg := *cfg
	globalCfg.Profiles = nil

	data, err := json.MarshalIndent(globalCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal global config: %w", err)
	}

	if err := os.WriteFile(cfgPath, data, 0o600); err != nil {
		return fmt.Errorf("write global config: %w", err)
	}

	// Save each profile and track active ones
	activeProfiles := make(map[string]bool)
	for name, p := range cfg.Profiles {
		pPath := filepath.Join(pDir, name+".json")
		activeProfiles[name+".json"] = true
		pData, err := json.MarshalIndent(p, "", "  ")
		if err != nil {
			continue
		}
		if err := os.WriteFile(pPath, pData, 0o600); err != nil {
			continue
		}
	}

	// Cleanup deleted profiles from disk
	files, err := os.ReadDir(pDir)
	if err == nil {
		for _, f := range files {
			if !f.IsDir() && filepath.Ext(f.Name()) == ".json" {
				if !activeProfiles[f.Name()] {
					_ = os.Remove(filepath.Join(pDir, f.Name()))
				}
			}
		}
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
