package vpn

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/akunbeben/ssht/internal/config"
)

func InterfaceName(confPath string) string {
	base := filepath.Base(confPath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func IsActive(interfaceName string) bool {
	bin := findBin("wg")
	if bin == "wg" {
		if _, err := exec.LookPath("wg"); err != nil {
			return false
		}
	}
	out, err := exec.Command(bin, "show", interfaceName).Output()
	return err == nil && len(out) > 0
}

func HasWgQuick() bool {
	bin := findBin("wg-quick")
	if bin == "wg-quick" {
		_, err := exec.LookPath("wg-quick")
		return err == nil
	}
	info, err := os.Stat(bin)
	return err == nil && !info.IsDir()
}

func Up(confPath string) error {
	expanded, err := config.ExpandHome(confPath)
	if err != nil {
		return err
	}
	bin := findBin("wg-quick")
	if bin == "wg-quick" {
		if _, err := exec.LookPath("wg-quick"); err != nil {
			return fmt.Errorf("wg-quick not found. Please run: brew install wireguard-tools")
		}
	}
	cmd := exec.Command("sudo", bin, "up", expanded)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Down(confPath string) error {
	expanded, err := config.ExpandHome(confPath)
	if err != nil {
		return err
	}
	bin := findBin("wg-quick")
	cmd := exec.Command("sudo", bin, "down", expanded)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func findBin(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	candidates := []string{
		"/opt/homebrew/bin/" + name,
		"/usr/local/bin/" + name,
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return name
}
