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

func BuildArgs(s config.Server, vpn *config.VPNConf) ([]string, error) {
	args := []string{"ssh"}
	if vpn != nil {
		executable, err := os.Executable()
		if err == nil {
			// On Windows, paths in ProxyCommand need care with backslashes and quotes
			confPath := vpn.ConfPath
			if runtime.GOOS == "windows" {
				confPath = strings.ReplaceAll(confPath, "\\", "/")
			}

			proxyCmd := fmt.Sprintf("\"%s\" vpn-dial --conf \"%s\" --host %%h --port %%p", executable, confPath)

			// OpenSSH on Windows sometimes needs double escaping for ProxyCommand quotes
			if runtime.GOOS == "windows" {
				proxyCmd = strings.ReplaceAll(proxyCmd, "\"", "\\\"")
			}

			args = append(args, "-o", fmt.Sprintf("ProxyCommand=%s", proxyCmd))
		}
	}
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
		args = append(args, "-p", strconv.Itoa(port))
	}

	args = append(args, fmt.Sprintf("%s@%s", s.User, s.Host))
	return args, nil
}

func Connect(s config.Server, vpn *config.VPNConf) error {
	return connectWithRetry(s, vpn, true)
}

func connectWithRetry(s config.Server, vpn *config.VPNConf, allowRetry bool) error {
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh not found in PATH: %w", err)
	}
	args, err := BuildArgs(s, vpn)
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
			return connectWithRetry(s, vpn, false)
		}
	}

	return err
}
