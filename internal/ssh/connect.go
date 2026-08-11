package ssh

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"runtime"

	"github.com/akunbeben/ssht/internal/config"
)

func BuildArgs(s config.Server, vpn *config.VPNConf, privacyMode bool) ([]string, error) {
	args := []string{"ssh"}
	if privacyMode {
		args = append(args, "-o", "LogLevel=ERROR", "-t")
	}

	proxyArgs, err := buildVPNProxyArgs(s, vpn)
	if err != nil {
		return nil, err
	}
	args = append(args, proxyArgs...)

	args, err = appendIdentityAndPort(args, s, "-p")
	if err != nil {
		return nil, err
	}

	args = append(args, "--", fmt.Sprintf("%s@%s", s.User, s.Host))
	if privacyMode {
		args = append(args, "clear; exec $SHELL -l")
	}
	return args, nil
}

// BuildSCPArgs builds a system scp command using the same per-session VPN
// proxy and SSH identity settings as BuildArgs.
func BuildSCPArgs(s config.Server, vpn *config.VPNConf, remotePath, localPath string, upload bool) ([]string, error) {
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		return nil, fmt.Errorf("remote path is required")
	}
	localPath = strings.TrimSpace(localPath)
	if localPath == "" {
		return nil, fmt.Errorf("local path is required")
	}

	localPath, err := config.ExpandHome(localPath)
	if err != nil {
		return nil, err
	}

	args := []string{"scp", "-r"}
	proxyArgs, err := buildVPNProxyArgs(s, vpn)
	if err != nil {
		return nil, err
	}
	args = append(args, proxyArgs...)

	args, err = appendIdentityAndPort(args, s, "-P")
	if err != nil {
		return nil, err
	}
	args = append(args, "--")

	remote := fmt.Sprintf("%s@%s:%s", s.User, s.Host, remotePath)
	if upload {
		return append(args, localPath, remote), nil
	}
	return append(args, remote, localPath), nil
}

func Connect(s config.Server, vpn *config.VPNConf, privacyMode bool) error {
	return connectWithRetry(s, vpn, privacyMode, true)
}

// Copy transfers files with scp while keeping the TUI's terminal attached.
func Copy(s config.Server, vpn *config.VPNConf, remotePath, localPath string, upload bool) error {
	return copyWithRetry(s, vpn, remotePath, localPath, upload, true)
}

func connectWithRetry(s config.Server, vpn *config.VPNConf, privacyMode bool, allowRetry bool) error {
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh not found in PATH: %w", err)
	}
	args, err := BuildArgs(s, vpn, privacyMode)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		args = args[1:]
	}

	cmd := exec.Command(sshPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	var stderrBuf bytes.Buffer
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	err = cmd.Run()
	if err != nil && allowRetry {
		stderr := stderrBuf.String()
		if strings.Contains(stderr, "REMOTE HOST IDENTIFICATION HAS CHANGED") {
			fmt.Fprintf(os.Stderr, "\n[ssht] Notice: Host key for %s changed. Cleaning known_hosts...\n", s.Host)
			_ = exec.Command("ssh-keygen", "-R", s.Host).Run()
			fmt.Fprintf(os.Stderr, "[ssht] Retrying connection...\n\n")
			return connectWithRetry(s, vpn, privacyMode, false)
		}
	}

	return err
}

func copyWithRetry(s config.Server, vpn *config.VPNConf, remotePath, localPath string, upload, allowRetry bool) error {
	scpPath, err := exec.LookPath("scp")
	if err != nil {
		return fmt.Errorf("scp not found in PATH: %w", err)
	}
	args, err := BuildSCPArgs(s, vpn, remotePath, localPath, upload)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		args = args[1:]
	}

	cmd := exec.Command(scpPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	var stderrBuf bytes.Buffer
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	err = cmd.Run()
	if err != nil && allowRetry {
		stderr := stderrBuf.String()
		if strings.Contains(stderr, "REMOTE HOST IDENTIFICATION HAS CHANGED") {
			fmt.Fprintf(os.Stderr, "\n[ssht] Notice: Host key for %s changed. Cleaning known_hosts...\n", s.Host)
			_ = exec.Command("ssh-keygen", "-R", s.Host).Run()
			fmt.Fprintf(os.Stderr, "[ssht] Retrying file transfer...\n\n")
			return copyWithRetry(s, vpn, remotePath, localPath, upload, false)
		}
	}

	return err
}

func buildVPNProxyArgs(s config.Server, vpn *config.VPNConf) ([]string, error) {
	// Prioritize server-level VPN override.
	activeVPN := vpn
	if s.VPN != nil {
		activeVPN = s.VPN
	}
	if activeVPN == nil {
		return nil, nil
	}

	if !isSafeProxyHost(s.Host) {
		return nil, fmt.Errorf("host %q contains characters that are unsafe for ProxyCommand", s.Host)
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve ssht executable for VPN proxy: %w", err)
	}

	// On Windows, paths in ProxyCommand need care with backslashes and quotes.
	confPath := activeVPN.ConfPath
	if runtime.GOOS == "windows" {
		confPath = strings.ReplaceAll(confPath, "\\", "/")
	}

	vpnType := activeVPN.Type
	if vpnType == "" {
		vpnType = "wireguard"
	}

	proxyCmd := fmt.Sprintf("%s vpn-dial --type %s --conf %s --host %s --port %s",
		proxyCommandQuote(executable),
		proxyCommandQuote(vpnType),
		proxyCommandQuote(confPath),
		proxyCommandQuote("%h"),
		proxyCommandQuote("%p"),
	)

	return []string{"-o", fmt.Sprintf("ProxyCommand=%s", proxyCmd)}, nil
}

func appendIdentityAndPort(args []string, s config.Server, portFlag string) ([]string, error) {
	if s.KeyPath != "" {
		keyPath, err := config.ExpandHome(s.KeyPath)
		if err != nil {
			return nil, err
		}
		args = append(args, "-i", keyPath)
	}

	port := s.Port
	if port == 0 {
		port = 22
	}
	if port != 22 {
		args = append(args, portFlag, strconv.Itoa(port))
	}
	return args, nil
}

func proxyCommandQuote(arg string) string {
	if runtime.GOOS == "windows" {
		arg = strings.ReplaceAll(arg, `\`, `\\`)
		arg = strings.ReplaceAll(arg, `"`, `\"`)
		return `"` + arg + `"`
	}
	return `'` + strings.ReplaceAll(arg, `'`, `'\''`) + `'`
}

func isSafeProxyHost(host string) bool {
	if host == "" {
		return false
	}
	for _, r := range host {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.' || r == '-' || r == '_' || r == ':' || r == '[' || r == ']':
		default:
			return false
		}
	}
	return true
}
