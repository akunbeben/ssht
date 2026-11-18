package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/akunbeben/ssht/internal/config"
)

// installKeyScript reads one public key from stdin and appends it to
// authorized_keys unless the exact line is already there. It stays on one line
// and avoids single quotes so it can be wrapped for any remote login shell.
const installKeyScript = `umask 077 && mkdir -p ~/.ssh && touch ~/.ssh/authorized_keys && k=$(cat) && ` +
	`{ [ ! -s ~/.ssh/authorized_keys ] || [ -z "$(tail -c1 ~/.ssh/authorized_keys)" ] || echo >> ~/.ssh/authorized_keys; } && ` +
	`{ grep -qxF -- "$k" ~/.ssh/authorized_keys || printf "%s\n" "$k" >> ~/.ssh/authorized_keys; } && ` +
	`{ ! command -v restorecon >/dev/null 2>&1 || restorecon -F ~/.ssh ~/.ssh/authorized_keys; }`

// InstallKeyCommand builds an ssh command that adds pubkeyPath to the server's
// authorized_keys. ssh asks for the password on the terminal as usual.
func InstallKeyCommand(s config.Server, vpn *config.VPNConf, pubkeyPath string) (*exec.Cmd, error) {
	data, err := os.ReadFile(pubkeyPath)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}
	pub := strings.TrimSpace(string(data))
	if pub == "" || strings.ContainsAny(pub, "\r\n") {
		return nil, fmt.Errorf("%s must contain exactly one public key", pubkeyPath)
	}

	args, err := baseArgs(s, vpn)
	if err != nil {
		return nil, err
	}
	args = append(args, "--", target(s), "exec sh -c '"+installKeyScript+"'")

	cmd, err := sshCommand(args)
	if err != nil {
		return nil, err
	}
	cmd.Stdin = strings.NewReader(pub + "\n")
	return cmd, nil
}

// VerifyKeyCommand builds a non-interactive ssh command that succeeds only
// when the server accepts keyPath without a password.
func VerifyKeyCommand(s config.Server, vpn *config.VPNConf, keyPath string) (*exec.Cmd, error) {
	s.KeyPath = keyPath
	args, err := baseArgs(s, vpn)
	if err != nil {
		return nil, err
	}
	args = append([]string{"-o", "BatchMode=yes", "-o", "IdentitiesOnly=yes", "-o", "ConnectTimeout=15"}, args...)
	return sshCommand(append(args, "--", target(s), "true"))
}

func baseArgs(s config.Server, vpn *config.VPNConf) ([]string, error) {
	args, err := buildVPNProxyArgs(s, vpn)
	if err != nil {
		return nil, err
	}
	return appendIdentityAndPort(args, s, "-p")
}

func target(s config.Server) string { return fmt.Sprintf("%s@%s", s.User, s.Host) }

func sshCommand(args []string) (*exec.Cmd, error) {
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return nil, fmt.Errorf("ssh not found in PATH: %w", err)
	}
	return exec.Command(sshPath, args...), nil
}
