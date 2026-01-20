# GhostShip Release Notes

## v1.0.0 (Alpha) 🚢👻🛠️
*Released: 2026-01-20*

**Status:** Alpha / Public Research Release

GhostShip v1.0.0 is the first public release of a peer-to-peer (P2P) Command & Control (C2) system designed for stealth and absolute network silence on the target side.

### ✨ Key Features

- **Universal Project Structure**: Single codebase supporting both **Linux** and **Windows**.
- **Phantom Socket Stealth**: Inter-process communication via kernel-level **Anonymous Pipes** (Linux) and **Named Pipes** (Windows). Zero network indicators on the target machine; `netstat` and `ss` report no listening ports.
- **Embedded P2P Transport**: Integrated [HyperDHT](https://github.com/holepunchto/hyperdht) for NAT-traversing, encrypted communications without central infrastructure.
- **Hardened Stealth (Windows)**:
  - **PPID Spoofing**: Automatically impersonates `svchost.exe` as the parent process.
  - **Memory Patching**: In-memory patching of **AMSI** and **ETW** to blind local telemetry.
- **Fileless Execution**:
  - **Linux**: Resident in memory via `memfd_create`.
  - **Windows**: Hidden folder residency with aggressive self-deletion logic.

### 🚀 Usage

Refer to [README.md](README.md) for installation and usage instructions.

```bash
# Build GhostShip
make build-all
```

---

*For academic research and authorized security testing only*
