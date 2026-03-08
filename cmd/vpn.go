package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/akunbeben/ssht/internal/vpn"
)

var vpnDialCmd = &cobra.Command{
	Use:    "vpn-dial",
	Short:  "Establish a tunneled connection for SSH ProxyCommand",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		conf, _ := cmd.Flags().GetString("conf")
		host, _ := cmd.Flags().GetString("host")
		portStr, _ := cmd.Flags().GetString("port")
		port, _ := strconv.Atoi(portStr)
		vpnType, _ := cmd.Flags().GetString("type")

		if conf == "" || host == "" {
			return fmt.Errorf("missing required flags")
		}

		conn, err := vpn.Dial(vpnType, conf, host, port)
		if err != nil {
			return err
		}
		defer conn.Close()

		copyErr := make(chan error, 2)
		go func() {
			_, err := io.Copy(conn, os.Stdin)
			copyErr <- err
		}()
		go func() {
			_, err := io.Copy(os.Stdout, conn)
			copyErr <- err
		}()

		return <-copyErr
	},
}

func init() {
	rootCmd.AddCommand(vpnDialCmd)
	vpnDialCmd.Flags().String("conf", "", "VPN config path or URI")
	vpnDialCmd.Flags().String("host", "", "Target host")
	vpnDialCmd.Flags().String("port", "22", "Target port")
	vpnDialCmd.Flags().String("type", "wireguard", "VPN type (wireguard, shadowsocks, trojan, openvpn)")
}
