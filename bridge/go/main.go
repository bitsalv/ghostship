package main

import (
	"fmt"
	"os"

	"github.com/security-research/holesail-sliver-bridge/pkg/bridge"
	"github.com/spf13/cobra"
)

var (
	sliverPort       int
	secure           bool
	logDir           string
	connStringFile   string
)

var rootCmd = &cobra.Command{
	Use:   "holesail-sliver-bridge",
	Short: "Expose Sliver C2 server through Holesail P2P network",
	Long: `Holesail-Sliver Bridge (Operator Side)

This bridge component exposes your local Sliver C2 server through the
Holesail P2P network, enabling covert command and control communications
without requiring public IP addresses, domain names, or port forwarding.

The bridge generates a connection string that must be embedded in your
implants to establish the P2P tunnel.

For academic research and authorized security testing only.`,
	RunE: run,
}

func init() {
	rootCmd.Flags().IntVarP(&sliverPort, "port", "p", 8888, "Sliver server port")
	rootCmd.Flags().BoolVarP(&secure, "secure", "s", true, "Use secure mode (recommended)")
	rootCmd.Flags().StringVarP(&logDir, "log-dir", "l", "./logs", "Directory for logs")
	rootCmd.Flags().StringVarP(&connStringFile, "output", "o", "./connection_string.txt", "Output file for connection string")
}

func run(cmd *cobra.Command, args []string) error {
	config := bridge.Config{
		SliverPort:           sliverPort,
		Secure:               secure,
		LogDir:               logDir,
		ConnectionStringFile: connStringFile,
	}

	b, err := bridge.NewBridge(config)
	if err != nil {
		return fmt.Errorf("failed to create bridge: %w", err)
	}

	if err := b.Start(); err != nil {
		return fmt.Errorf("failed to start bridge: %w", err)
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
