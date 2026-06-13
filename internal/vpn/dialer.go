package vpn

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/akunbeben/ssht/internal/config"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

func keyToHex(key string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func Dial(vpnType, confPath, host string, port int) (net.Conn, error) {
	switch strings.ToLower(vpnType) {
	case "wireguard", "wg", "":
		return DialSharedWireguard(confPath, host, port)
	case "shadowsocks", "ss":
		return DialShadowsocks(confPath, host, port)
	case "trojan":
		return DialTrojan(confPath, host, port)
	case "openvpn", "ovpn":
		return DialOpenVPN(confPath, host, port)
	default:
		return nil, fmt.Errorf("unsupported vpn type: %s", vpnType)
	}
}

func DialWireguard(confPath, host string, port int) (net.Conn, error) {
	dev, tnet, err := newWireguardNet(confPath)
	if err != nil {
		return nil, err
	}

	target := fmt.Sprintf("%s:%d", host, port)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := tnet.DialContext(ctx, "tcp", target)
	if err != nil {
		dev.Close()
		return nil, fmt.Errorf("dial %s: %w", target, err)
	}

	return &lazyClosingConn{Conn: conn, dev: dev}, nil
}

func newWireguardNet(confPath string) (*device.Device, *netstack.Net, error) {
	confPath, err := config.ExpandHome(confPath)
	if err != nil {
		return nil, nil, fmt.Errorf("expand home: %w", err)
	}

	cfg, err := ParseConfig(confPath)
	if err != nil {
		return nil, nil, fmt.Errorf("parse config: %w", err)
	}

	localIPs := parseAddresses(cfg.Address, true)
	if len(localIPs) == 0 {
		return nil, nil, fmt.Errorf("no valid Address in [Interface]")
	}
	dnsIPs := parseAddresses(cfg.DNS, false)

	tun, tnet, err := netstack.CreateNetTUN(localIPs, dnsIPs, cfg.MTU)
	if err != nil {
		return nil, nil, fmt.Errorf("create netstack: %w", err)
	}

	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(device.LogLevelSilent, ""))

	uapi, err := buildUAPI(cfg)
	if err != nil {
		dev.Close()
		return nil, nil, err
	}

	if err := dev.IpcSet(uapi); err != nil {
		dev.Close()
		return nil, nil, fmt.Errorf("configure device: %w", err)
	}

	if err := dev.Up(); err != nil {
		dev.Close()
		return nil, nil, fmt.Errorf("bring up device: %w", err)
	}

	return dev, tnet, nil
}

func parseAddresses(addrs []string, stripMask bool) []netip.Addr {
	res := make([]netip.Addr, 0, len(addrs))
	for _, addr := range addrs {
		if stripMask {
			if idx := strings.Index(addr, "/"); idx != -1 {
				addr = addr[:idx]
			}
		}
		if ip, err := netip.ParseAddr(addr); err == nil {
			res = append(res, ip)
		}
	}
	return res
}

func buildUAPI(cfg *Config) (string, error) {
	privHex, err := keyToHex(cfg.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("invalid private key: %w", err)
	}
	uapi := fmt.Sprintf("private_key=%s\n", privHex)
	for _, p := range cfg.Peers {
		pubHex, err := keyToHex(p.PublicKey)
		if err != nil {
			continue
		}
		uapi += fmt.Sprintf("public_key=%s\n", pubHex)
		if p.Endpoint != "" {
			uapi += fmt.Sprintf("endpoint=%s\n", p.Endpoint)
		}
		if p.PersistentKeepalive > 0 {
			uapi += fmt.Sprintf("persistent_keepalive_interval=%d\n", p.PersistentKeepalive)
		}
		for _, a := range p.AllowedIPs {
			uapi += fmt.Sprintf("allowed_ip=%s\n", a)
		}
	}
	return uapi, nil
}

type lazyClosingConn struct {
	net.Conn
	dev *device.Device
}

func (c *lazyClosingConn) Close() error {
	err := c.Conn.Close()
	c.dev.Close()
	return err
}
