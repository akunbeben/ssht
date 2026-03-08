package vpn

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"strings"

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

func Dial(confPath, host string, port int) (net.Conn, error) {
	confPath, err := config.ExpandHome(confPath)
	if err != nil {
		return nil, fmt.Errorf("expand home: %w", err)
	}

	cfg, err := ParseConfig(confPath)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	var localIPs []netip.Addr
	for _, addr := range cfg.Address {
		if idx := strings.Index(addr, "/"); idx != -1 {
			addr = addr[:idx]
		}
		if ip, err := netip.ParseAddr(addr); err == nil {
			localIPs = append(localIPs, ip)
		}
	}
	if len(localIPs) == 0 {
		return nil, fmt.Errorf("no valid Address in [Interface]")
	}

	var dnsIPs []netip.Addr
	for _, addr := range cfg.DNS {
		if ip, err := netip.ParseAddr(addr); err == nil {
			dnsIPs = append(dnsIPs, ip)
		}
	}

	tun, tnet, err := netstack.CreateNetTUN(localIPs, dnsIPs, cfg.MTU)
	if err != nil {
		return nil, fmt.Errorf("create netstack: %w", err)
	}

	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(device.LogLevelSilent, ""))

	privHex, err := keyToHex(cfg.PrivateKey)
	if err != nil {
		dev.Close()
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	uapi := fmt.Sprintf("private_key=%s\n", privHex)
	for _, p := range cfg.Peers {
		pubHex, err := keyToHex(p.PublicKey)
		if err != nil {
			continue // skip invalid peers
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

	if err := dev.IpcSet(uapi); err != nil {
		dev.Close()
		return nil, fmt.Errorf("configure device: %w", err)
	}

	if err := dev.Up(); err != nil {
		dev.Close()
		return nil, fmt.Errorf("bring up device: %w", err)
	}

	target := fmt.Sprintf("%s:%d", host, port)

	// If host is not an IP, we might need DNS.
	// Netstack handles DNS if we provided it in dnsIPs.

	ctx := context.Background()
	conn, err := tnet.DialContext(ctx, "tcp", target)
	if err != nil {
		dev.Close()
		return nil, fmt.Errorf("dial %s: %w", target, err)
	}

	return &lazyClosingConn{Conn: conn, dev: dev}, nil
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
