package config

// Config is the root config structure.
type Config struct {
	LastProfile string             `json:"last_profile"`
	Profiles    map[string]Profile `json:"profiles"`
	PrivacyMode bool               `json:"privacy_mode"`
	SyncEnabled bool               `json:"sync_enabled"`
	SyncRepo    string             `json:"sync_repo,omitempty"`
}

// Profile stores one working context.
type Profile struct {
	Name             string   `json:"name"`
	VPN              *VPNConf `json:"vpn,omitempty"`
	Servers          []Server `json:"servers"`
	ImportExceptions []string `json:"import_exceptions,omitempty"` // endpoint keys to skip on auto-import
}

// VPNConf stores WireGuard settings for one profile.
type VPNConf struct {
	Type     string `json:"type"`
	ConfPath string `json:"conf_path"`
	AutoUp   bool   `json:"auto_up"`
	AutoDown bool   `json:"auto_down"`
}

// Server stores one ssh target.
type Server struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Host    string   `json:"host"`
	Port    int      `json:"port"`
	User    string   `json:"user"`
	KeyPath string   `json:"key_path"`
	VPN     *VPNConf `json:"vpn,omitempty"`
	Tags    []string `json:"tags"`
	Note    string   `json:"note"`
}
