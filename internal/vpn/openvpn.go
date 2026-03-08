package vpn

import (
	"fmt"
	"net"
	"os/exec"

	"github.com/akunbeben/ssht/internal/config"
)

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

	// OpenVPN historically creates a TUN/TAP interface which routes system-wide.
	// Implementing a user-space OpenVPN client is complex, so we currently
	// support it by wrapping the system binary for system-wide connection.

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
