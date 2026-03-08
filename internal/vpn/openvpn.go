package vpn

import (
	"fmt"
	"net"
	"os/exec"

	"github.com/akunbeben/ssht/internal/config"
)

// OpenVPN Dialer implementation wrapping the system's openvpn binary.

func DialOpenVPN(confPath, host string, port int) (net.Conn, error) {
	expanded, err := config.ExpandHome(confPath)
	if err != nil {
		return nil, err
	}

	bin := findBin("openvpn")
	if bin == "openvpn" {
		if _, err := exec.LookPath("openvpn"); err != nil {
			return nil, fmt.Errorf("openvpn binary not found in PATH")
		}
	}

	// For OpenVPN, we need to run it as a background process and then dial through it.
	// HOWEVER, OpenVPN historically creates a TUN/TAP interface which routes system-wide.
	// ssht's approach for Wireguard is to use a user-space netstack.
	// Implementing a user-space OpenVPN client is too complex.
	//
	// A simpler approach for ProxyCommand usage is to use OpenVPN's management interface
	// or to just use a SOCKS proxy if the user has one configured via OpenVPN.
	//
	// Given the constraints, we will use 'openvpn --config conf --management 127.0.0.1 0'
	// to find a way to tunnel, but OpenVPN doesn't easily provide a "dial this host through this config"
	// without system-wide routing unless using third-party socks wrappers.

	// Better approach for ssht: Use openvpn with --dev null --management to establish session
	// but that still doesn't give us a net.Conn.

	// Realistically, if we want to support OpenVPN in ssht as a Dialer (net.Conn),
	// we would need a user-space client. Since we decided to wrap the binary:
	// We will assume the user wants the binary to handle the connection.

	return nil, fmt.Errorf("openvpn support is currently limited to system-wide connection via 'sudo openvpn --config %s'", expanded)
}

func OpenVpnUp(confPath string) error {
	expanded, err := config.ExpandHome(confPath)
	if err != nil {
		return err
	}
	bin := findBin("openvpn")
	cmd := exec.Command("sudo", bin, "--config", expanded, "--daemon")
	return cmd.Run()
}

func OpenVpnDown() error {
	// This is tricky as we need to find the specific PID.
	// Simplified:
	return exec.Command("sudo", "killall", "openvpn").Run()
}
