# printctl Deep Architectural Whitepaper 🏗️

This document details the first-principles design, protocol state machines, in-memory layout engine, and concurrency architecture behind **printctl**.

---

## Table of Contents
1. [Core Design Philosophy](#1-core-design-philosophy)
2. [Protocol State Machines](#2-protocol-state-machines)
3. [The Anti-Fail In-Memory Layout Engine](#3-the-anti-fail-in-memory-layout-engine)
4. [Subnet Discovery & Concurrency Model](#4-subnet-discovery--concurrency-model)
5. [AI Agent Model Context Protocol (MCP) Architecture](#5-ai-agent-model-context-protocol-mcp-architecture)
6. [Security, Performance & Zero-Daemon Guarantees](#6-security-performance--zero-daemon-guarantees)

---

## 1. Core Design Philosophy

`printctl` operates on the first-principles definition of physical paper interaction:
* **Optical Scanning** is an image sampling operation over HTTP REST (`eSCL / AirScan`).
* **Physical Printing** is a rasterized vector stream dispatch over HTTP/2 / TCP 631 (`IPP Everywhere / RFC 8011`).

By strictly adhering to open industry standards, `printctl` achieves complete independence from vendor-specific drivers, proprietary daemons, and OS-level printer subsystem bloat.

```
 +-----------------------------------------------------------------------------+
 |                         Application / AI Agent Layer                        |
 +-----------------------------------------------------------------------------+
               | (CLI Flags)                 | (HTTP REST)              | (JSON-RPC)
               v                             v                          v
       +---------------+             +---------------+          +---------------+
       | printctl CLI  |             | REST Server   |          | MCP Server    |
       |  (Terminal)   |             | (Port 8080)   |          |    (stdio)    |
       +---------------+             +---------------+          +---------------+
               \                             |                          /
                +----------------------------+-------------------------+
                                             |
                                             v
 +-----------------------------------------------------------------------------+
 |                      printctl Core Go Engine (In-Memory)                    |
 |                                                                             |
 |  [ Discovery Engine ]       [ Anti-Fail Layout Engine ]      [ IPP/eSCL ]  |
 |  254 Goroutines Sweep       Alpha Flattening + Margins       Wire Protocol  |
 +-----------------------------------------------------------------------------+
                                             |
                               (Raw LAN Protocol Stream)
                                             v
 +-----------------------------------------------------------------------------+
 |                      Physical Network Multi-Function Printer                |
 |                     (Canon, HP, Epson, Brother, Xerox, etc.)                |
 +-----------------------------------------------------------------------------+
```

---

## 2. Protocol State Machines

### 2.1 Driverless Scanning (eSCL 2.9 / Apple AirScan)
The scanner pipeline communicates directly with the printer's onboard HTTP server over port 80/443:

```
 Client (printctl)                                        Printer (Hardware eSCL)
        |                                                            |
        | 1. GET /eSCL/ScannerCapabilities                          |
        |----------------------------------------------------------->|
        |    <- 200 OK (XML Capabilities: 75-600 DPI, RGB24, Platen) |
        |<-----------------------------------------------------------|
        |                                                            |
        | 2. POST /eSCL/ScanJobs (XML Settings Payload)              |
        |----------------------------------------------------------->|
        |    <- 201 Created (Location: /eSCL/ScanJobs/<UUID>)        |
        |<-----------------------------------------------------------|
        |                                                            |
        | 3. Polling: GET /eSCL/ScanJobs/<UUID>/NextDocument         |
        |----------------------------------------------------------->|
        |    <- 200 OK (Chunked Binary JPEG/PDF Stream)              |
        |<-----------------------------------------------------------|
```

### 2.2 Driverless Printing (IPP Everywhere / RFC 8011)
The print pipeline constructs standard IPP binary frames sent to `http://<IP>:631/ipp/print`:
* **Header**: Version `0x0200` (IPP 2.0), Operation `0x0002` (Print-Job), Request-ID `1`.
* **Attributes**: `attributes-charset=utf-8`, `printer-uri`, `document-format=image/jpeg` (or `image/pwg-raster`).
* **Hardware Ingest Capabilities (Known Device Traits)**:
  Consumer network ink-tank hardware natively supports:
  * `image/jpeg` (Direct hardware ASIC JPEG rendering)
  * `image/pwg-raster` (PWG Standard Raster)
  * `image/urf` (Apple AirPrint UNIRAST)
  * `application/octet-stream`
* **Payload**: Clean normalized single-page stream with 300 DPI geometry and hardware safe margins.

---

## 3. The Anti-Fail In-Memory Layout Engine

To prevent common printer failures (black backgrounds on transparent PNGs, accidental multi-page pagination overflows, and cut off edges), `printctl` implements deterministic in-memory preprocessing:

```
 [ Raw Image File ]
         │
         ▼
 1. Decode into image.Image (RGBA)
         │
         ▼
 2. Alpha Compositing (Flattening onto Pure White #FFFFFF):
    C_out = C_src * alpha + 255 * (1 - alpha)
         │
         ▼
 3. Aspect-Ratio Bounding Box Fitting:
    - Target: A4 Standard (595.28 pt x 841.89 pt @ 72 DPI)
    - Hardware Safe Margin Clamp: 20.0 pt (~7mm) on all 4 borders
    - Strict Single-Page Math: Math.Floor on raster bounding boxes
         │
         ▼
 4. Minimal Single-Page PDF Generation (In-Memory Byte Stream)
```

---

## 4. Subnet Discovery & Concurrency Model

* **Mechanism**: Concurrent Goroutine Subnet Sweep.
* **Concurrency**: Spawns 254 non-blocking Goroutines across all local `/24` subnets (`192.168.x.x`, `10.x.x.x`, `172.16-31.x.x`).
* **Timeout**: 500ms per probe with `io.LimitReader` protection.
* **Latency**: Resolves active local printer IP in `< 200ms` on typical gigabit/Wi-Fi 6 networks.

---

## 5. AI Agent Model Context Protocol (MCP) Architecture

`printctl mcp` runs as a standard **Model Context Protocol (MCP)** stdio server based on JSON-RPC 2.0.

### Exposed Tool Primitives:
1. `printer_scan(output_path, dpi)`: Triggers optical scanning directly into a local file.
2. `printer_print(file_path)`: Streams an image or PDF to the physical printer.
3. `printer_status()`: Queries online telemetry, scanner idle state, and health.

---

## 6. Security, Performance & Zero-Daemon Guarantees

* **Zero-Daemon**: No background processes, no root privileges required, zero resident memory when idle.
* **Zero Network Leakage**: All traffic is strictly bounded within the local broadcast/multicast LAN domain.
* **Static Binary**: Compiled with `CGO_ENABLED=0 -ldflags="-s -w"`, delivering a single ~5MB binary.
