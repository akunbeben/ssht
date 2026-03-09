package vpn

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	PrivateKey string
	Address    []string
	DNS        []string
	MTU        int
	Peers      []PeerConfig
}

type PeerConfig struct {
	PublicKey           string
	AllowedIPs          []string
	Endpoint            string
	PersistentKeepalive int
}

func ParseConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := &Config{}
	var currentPeer *PeerConfig
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if isSection, nextPeer := cfg.processSection(line); isSection {
			currentPeer = nextPeer
			continue
		}

		cfg.processLine(line, currentPeer)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("missing PrivateKey in [Interface]")
	}
	return cfg, nil
}
func (cfg *Config) processSection(line string) (bool, *PeerConfig) {
	if strings.EqualFold(line, "[Interface]") {
		return true, nil
	}
	if strings.EqualFold(line, "[Peer]") {
		cfg.Peers = append(cfg.Peers, PeerConfig{})
		return true, &cfg.Peers[len(cfg.Peers)-1]
	}
	return false, nil
}

func (cfg *Config) processLine(line string, currentPeer *PeerConfig) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return
	}
	key := strings.TrimSpace(parts[0])
	val := strings.TrimSpace(parts[1])

	if currentPeer == nil {
		cfg.processInterfaceLine(key, val)
	} else {
		currentPeer.processPeerLine(key, val)
	}
}

func (cfg *Config) processInterfaceLine(key, val string) {
	switch {
	case strings.EqualFold(key, "PrivateKey"):
		cfg.PrivateKey = val
	case strings.EqualFold(key, "Address"):
		cfg.Address = splitValue(val)
	case strings.EqualFold(key, "DNS"):
		cfg.DNS = splitValue(val)
	case strings.EqualFold(key, "MTU"):
		cfg.MTU, _ = strconv.Atoi(val)
	}
}

func (p *PeerConfig) processPeerLine(key, val string) {
	switch {
	case strings.EqualFold(key, "PublicKey"):
		p.PublicKey = val
	case strings.EqualFold(key, "AllowedIPs"):
		p.AllowedIPs = splitValue(val)
	case strings.EqualFold(key, "Endpoint"):
		p.Endpoint = val
	case strings.EqualFold(key, "PersistentKeepalive"):
		p.PersistentKeepalive, _ = strconv.Atoi(val)
	}
}

func splitValue(val string) []string {
	parts := strings.Split(val, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			res = append(res, s)
		}
	}
	return res
}
