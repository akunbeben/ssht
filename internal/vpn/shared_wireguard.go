package vpn

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/akunbeben/ssht/internal/config"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

const wireguardHubIdleTimeout = 30 * time.Second

func DialSharedWireguard(confPath, host string, port int) (net.Conn, error) {
	confPath, err := config.ExpandHome(confPath)
	if err != nil {
		return nil, err
	}

	if conn, err := dialWireguardHub(confPath, host, port); err == nil {
		return conn, nil
	}

	if err := startWireguardHub(confPath); err != nil {
		return nil, err
	}

	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := dialWireguardHub(confPath, host, port)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		time.Sleep(100 * time.Millisecond)
	}

	return nil, fmt.Errorf("connect to WireGuard hub: %w", lastErr)
}

func RunWireguardHub(confPath string) error {
	confPath, err := config.ExpandHome(confPath)
	if err != nil {
		return err
	}

	socketPath, lockDir, err := wireguardHubPaths(confPath)
	if err != nil {
		return err
	}

	if err := os.Mkdir(lockDir, 0o700); err != nil {
		if conn, dialErr := net.Dial("unix", socketPath); dialErr == nil {
			_ = conn.Close()
			return nil
		}
		_ = os.RemoveAll(lockDir)
		_ = os.Remove(socketPath)
		if err := os.Mkdir(lockDir, 0o700); err != nil {
			return fmt.Errorf("WireGuard hub already starting")
		}
	}
	defer os.RemoveAll(lockDir)
	defer os.Remove(socketPath)

	dev, tnet, err := newWireguardNet(confPath)
	if err != nil {
		return err
	}
	defer dev.Close()

	_ = os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("listen on WireGuard hub socket: %w", err)
	}
	defer listener.Close()

	unixListener, ok := listener.(*net.UnixListener)
	if !ok {
		return fmt.Errorf("unexpected listener type %T", listener)
	}

	active := 0
	done := make(chan struct{}, 1024)
	lastActive := time.Now()

	for {
		for {
			select {
			case <-done:
				active--
				lastActive = time.Now()
			default:
				goto drained
			}
		}
	drained:
		if active == 0 && time.Since(lastActive) >= wireguardHubIdleTimeout {
			return nil
		}

		_ = unixListener.SetDeadline(time.Now().Add(1 * time.Second))
		conn, err := unixListener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return err
		}

		active++
		lastActive = time.Now()
		go func() {
			defer func() { done <- struct{}{} }()
			handleWireguardHubConn(conn, tnet)
		}()
	}
}

func dialWireguardHub(confPath, host string, port int) (net.Conn, error) {
	socketPath, _, err := wireguardHubPaths(confPath)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTimeout("unix", socketPath, time.Second)
	if err != nil {
		return nil, err
	}

	if _, err := fmt.Fprintf(conn, "%s:%d\n", host, port); err != nil {
		conn.Close()
		return nil, err
	}

	status := []byte{1}
	if _, err := io.ReadFull(conn, status); err != nil {
		conn.Close()
		return nil, err
	}
	if status[0] != 0 {
		msg, _ := io.ReadAll(conn)
		conn.Close()
		return nil, errors.New(strings.TrimSpace(string(msg)))
	}

	return conn, nil
}

func startWireguardHub(confPath string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(executable, "vpn-hub", "--type", "wireguard", "--conf", confPath)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func handleWireguardHubConn(client net.Conn, tnet *netstack.Net) {
	defer client.Close()

	reader := bufio.NewReader(client)
	target, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	target = strings.TrimSpace(target)
	if target == "" {
		_, _ = client.Write(append([]byte{1}, []byte("missing target")...))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	remote, err := tnet.DialContext(ctx, "tcp", target)
	if err != nil {
		_, _ = client.Write(append([]byte{1}, []byte(err.Error())...))
		return
	}
	defer remote.Close()

	if _, err := client.Write([]byte{0}); err != nil {
		return
	}

	copyDone := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(remote, reader)
		copyDone <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(client, remote)
		copyDone <- struct{}{}
	}()
	<-copyDone
}

func wireguardHubPaths(confPath string) (string, string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", "", err
	}
	stateDir := filepath.Join(dir, "sessions")
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return "", "", err
	}

	sum := sha256.Sum256([]byte(confPath))
	name := hex.EncodeToString(sum[:16])
	return filepath.Join(stateDir, "wg-"+name+".sock"), filepath.Join(stateDir, "wg-"+name+".lock"), nil
}
