# Changelog

All notable changes to GhostShip will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0-alpha] - 2026-01-20

### Added
- **Initial Public Alpha Release** 🚢👻
- **Cross-Platform Loader**: Single-binary project tree supporting both **Linux** and **Windows**.
- **Stealth Memory Bridge**: Inter-process communication via kernel-level anonymous/named pipes. No TCP/UDP listening ports on the target host (`netstat` clean).
- **Windows Hardening**:
  - **PPID Spoofing**: Automatically impersonates `svchost.exe` as the parent process.
  - **In-Memory Patching**: Patching of **AMSI** (`amsi.dll`) and **ETW** (`ntdll.dll`) to blind host defenses.
- **Residency Management**:
  - **Linux**: Memory residency via `memfd_create`.
  - **Windows**: Hidden directory residency with self-deletion logic.
- **Advanced Evasion**: Integrated anti-sandbox checks and XOR string obfuscation.
- **P2P Command & Control**: Integrated [HyperDHT](https://github.com/holepunchto/hyperdht) for serverless, encrypted C2 transport.
