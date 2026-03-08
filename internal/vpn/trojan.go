package vpn

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
)

// Trojan Dialer implementation without external dependencies
// Protocol: hash(password) + CRLF + type + address + port + CRLF + payload

func DialTrojan(confPath, host string, port int) (net.Conn, error) {
	// For now, confPath is the URI directly
	u, err := url.Parse(confPath)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "trojan" {
		return nil, fmt.Errorf("invalid trojan uri")
	}

	password := u.User.Username()
	serverHost := u.Hostname()
	serverPort := u.Port()
	if serverPort == "" {
		serverPort = "443"
	}

	// Trojan usually runs over TLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: u.Query().Get("allowInsecure") == "1",
		ServerName:         u.Query().Get("sni"),
	}
	if tlsConfig.ServerName == "" {
		tlsConfig.ServerName = serverHost
	}

	c, err := tls.Dial("tcp", net.JoinHostPort(serverHost, serverPort), tlsConfig)
	if err != nil {
		return nil, err
	}

	// Create Trojan request header
	// 1. Password hash (56 bytes hex)
	h := sha256.New()
	h.Write([]byte(password))
	passwordHash := hex.EncodeToString(h.Sum(nil))

	// 2. Connect command (0x01)
	// 3. Address type + Address + Port
	var addr []byte
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			addr = append([]byte{1}, ip4...)
		} else {
			addr = append([]byte{4}, ip.To16()...)
		}
	} else {
		addr = append([]byte{3, byte(len(host))}, []byte(host)...)
	}
	addr = append(addr, byte(port>>8), byte(port))

	// Header: hash + CRLF + cmd + addr + CRLF
	header := make([]byte, 0, 56+2+1+len(addr)+2)
	header = append(header, []byte(passwordHash)...)
	header = append(header, '\r', '\n')
	header = append(header, 1) // Command: Connect
	header = append(header, addr...)
	header = append(header, '\r', '\n')

	if _, err := c.Write(header); err != nil {
		c.Close()
		return nil, err
	}

	return c, nil
}
