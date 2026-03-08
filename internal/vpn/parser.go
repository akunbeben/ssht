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

		if strings.EqualFold(line, "[Interface]") {
			currentPeer = nil
			continue
		}
		if strings.EqualFold(line, "[Peer]") {
			cfg.Peers = append(cfg.Peers, PeerConfig{})
			currentPeer = &cfg.Peers[len(cfg.Peers)-1]
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		if currentPeer == nil {
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
		} else {
			switch {
			case strings.EqualFold(key, "PublicKey"):
				currentPeer.PublicKey = val
			case strings.EqualFold(key, "AllowedIPs"):
				currentPeer.AllowedIPs = splitValue(val)
			case strings.EqualFold(key, "Endpoint"):
				currentPeer.Endpoint = val
			case strings.EqualFold(key, "PersistentKeepalive"):
				currentPeer.PersistentKeepalive, _ = strconv.Atoi(val)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("missing PrivateKey in [Interface]")
	}
	if len(cfg.Peers) == 0 {
		return nil, fmt.Errorf("missing [Peer] section")
	}
	return cfg, nil
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
