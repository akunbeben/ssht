package ssh

import (
	"reflect"
	"strings"
	"testing"

	"github.com/akunbeben/ssht/internal/config"
)

func TestBuildSCPArgsDownload(t *testing.T) {
	server := config.Server{
		Host:    "example.com",
		Port:    2222,
		User:    "deploy",
		KeyPath: "/tmp/deploy_key",
	}

	args, err := BuildSCPArgs(server, nil, "/var/log/app.log", "./downloads", false)
	if err != nil {
		t.Fatalf("BuildSCPArgs returned error: %v", err)
	}

	want := []string{
		"scp", "-r", "-i", "/tmp/deploy_key", "-P", "2222", "--",
		"deploy@example.com:/var/log/app.log", "./downloads",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("BuildSCPArgs returned %v, want %v", args, want)
	}
}

func TestBuildSCPArgsUpload(t *testing.T) {
	server := config.Server{Host: "example.com", User: "deploy"}

	args, err := BuildSCPArgs(server, nil, "/srv/releases/app.tar.gz", "./app.tar.gz", true)
	if err != nil {
		t.Fatalf("BuildSCPArgs returned error: %v", err)
	}

	want := []string{
		"scp", "-r", "--", "./app.tar.gz", "deploy@example.com:/srv/releases/app.tar.gz",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("BuildSCPArgs returned %v, want %v", args, want)
	}
}

func TestBuildSCPArgsUsesVPNProxy(t *testing.T) {
	server := config.Server{Host: "10.0.0.12", User: "root"}
	vpn := &config.VPNConf{Type: "wireguard", ConfPath: "/tmp/wg.conf"}

	args, err := BuildSCPArgs(server, vpn, "/etc/hosts", ".", false)
	if err != nil {
		t.Fatalf("BuildSCPArgs returned error: %v", err)
	}

	joined := strings.Join(args, " ")
	for _, part := range []string{"ProxyCommand=", "vpn-dial", "--conf", "/tmp/wg.conf", "--host", "%h", "--port", "%p"} {
		if !strings.Contains(joined, part) {
			t.Fatalf("SCP args %q do not contain %q", joined, part)
		}
	}
}

func TestBuildSCPArgsRequiresPaths(t *testing.T) {
	server := config.Server{Host: "example.com", User: "deploy"}

	if _, err := BuildSCPArgs(server, nil, "", ".", false); err == nil {
		t.Fatal("expected missing remote path error")
	}
	if _, err := BuildSCPArgs(server, nil, "/tmp/file", "", false); err == nil {
		t.Fatal("expected missing local path error")
	}
}
