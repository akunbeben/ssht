package ssh

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/akunbeben/ssht/internal/config"
)

func TestInstallKeyScriptHasNoSingleQuotes(t *testing.T) {
	if strings.Contains(installKeyScript, "'") {
		t.Fatal("installKeyScript is wrapped in single quotes and must not contain one")
	}
}

func TestInstallKeyScriptAppendsOnce(t *testing.T) {
	home := t.TempDir()
	auth := filepath.Join(home, ".ssh", "authorized_keys")
	os.MkdirAll(filepath.Dir(auth), 0o700)
	os.WriteFile(auth, []byte("ssh-ed25519 OLD old"), 0o600) // no trailing newline

	for range 2 {
		cmd := exec.Command("sh", "-c", installKeyScript)
		cmd.Env = append(os.Environ(), "HOME="+home)
		cmd.Stdin = strings.NewReader("ssh-ed25519 NEW new\n")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("script failed: %v: %s", err, out)
		}
	}

	got, _ := os.ReadFile(auth)
	if want := "ssh-ed25519 OLD old\nssh-ed25519 NEW new\n"; string(got) != want {
		t.Fatalf("authorized_keys = %q, want %q", got, want)
	}
}

func TestInstallKeyCommand(t *testing.T) {
	pub := filepath.Join(t.TempDir(), "id.pub")
	os.WriteFile(pub, []byte("ssh-ed25519 AAAA test\n"), 0o600)
	s := config.Server{Host: "example.com", Port: 2222, User: "deploy"}

	cmd, err := InstallKeyCommand(s, nil, pub)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-p", "2222", "--", "deploy@example.com", "exec sh -c '" + installKeyScript + "'"}
	if !slices.Equal(cmd.Args[1:], want) {
		t.Fatalf("args = %q, want %q", cmd.Args[1:], want)
	}

	os.WriteFile(pub, []byte("ssh-ed25519 A a\nssh-ed25519 B b\n"), 0o600)
	if _, err := InstallKeyCommand(s, nil, pub); err == nil {
		t.Fatal("expected error for a file with two keys")
	}
}

func TestVerifyKeyCommandUsesOnlyNewKey(t *testing.T) {
	s := config.Server{Host: "example.com", User: "deploy", KeyPath: "/old"}
	cmd, err := VerifyKeyCommand(s, &config.VPNConf{Type: "wireguard", ConfPath: "/wg.conf"}, "/new")
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Join(cmd.Args, " ")
	for _, want := range []string{"BatchMode=yes", "IdentitiesOnly=yes", "ProxyCommand=", "-i /new", "deploy@example.com true"} {
		if !strings.Contains(args, want) {
			t.Fatalf("args %q missing %q", args, want)
		}
	}
}
