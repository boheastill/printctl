# printctl 🖨️⚡

[![Release](https://img.shields.io/github/v/release/boheastill/printctl?include_prereleases)](https://github.com/boheastill/printctl/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/boheastill/printctl)](https://goreportcard.com/report/github.com/boheastill/printctl)

> **"We do the dirty hardware bridge, so your apps and AI agents get instant physical I/O."**  
> A zero-dependency, 5MB static Go binary that turns any modern network printer/scanner into a standard CLI, REST API microservice, and AI Agent (MCP) Skill. **No CUPS. No SANE. No vendor bloatware.**

🌐 **[🇨🇳 阅读中文说明 (Chinese Documentation)](README_zh.md)**

---

## 💥 The Problem We Solve

If you've ever tried to make a **Linux server, Raspberry Pi, NAS, Docker container, or AI Agent** print a document or scan a paper receipt, you know the nightmare:

```
The Legacy Linux Nightmare:
1. sudo apt install cups sane sane-airscan avahi-daemon (eats 500MB of disk)
2. Struggle with the cryptic localhost:631 web UI guessing PPD driver models
3. "Filter failed" errors, transparent PNGs printed with pitch-black backgrounds, random blank pages
4. scanimage -L returns "No SANE devices found" due to daemon permission hell
5. Heavy background daemons eating RAM forever...

═══════════════════════════════════════════════════════════════════════════════════

The printctl Way:
1. Single 5MB static binary (Zero dependencies, zero background daemons, zero root required)
2. Run one command — immediate physical page output or optical scan:

   $ printctl scan ./invoice.pdf      # Auto-discovers printer, pulls 300 DPI A4 PDF
   $ printctl print ./photo.png       # Auto anti-clip margins & alpha-flattening
```

---

## 🎯 What printctl Provides (3 Consumption Surfaces)

```
                       【Your Applications & AI Brains】
       (CLI Scripts / Cron / Python Backend / DeepSeek / Claude / Cursor)
                                      │
        ┌─────────────────────────────┼─────────────────────────────┐
        ▼                             ▼                             ▼
   【1. CLI Mode】             【2. REST API Mode】          【3. AI Agent (MCP)】
   Direct terminal commands    'printctl serve --port 8080'    'printctl mcp' (stdio)
   for bash & sysadmins.       HTTP JSON microservice for     Native Tool/Skill for
                               custom SDK/Auth integrations.   LLM Physical I/O.
        └─────────────────────────────┬─────────────────────────────┘
                                      │
                 (Zero-Config eSCL REST & IPP Everywhere)
                                      ▼
                      【Physical Network Printer】
                 (Canon, HP, Epson, Brother, Mopria, AirPrint)
```

1. **CLI Engine**: Instant terminal commands with `--json` support for bash scripts and cron jobs.
2. **Headless REST API Daemon (`printctl serve`)**: Spin up a lightweight local microservice so your Node.js, Python, Go, or Java backends can print and scan via standard HTTP REST endpoints.
3. **AI-Native Skill / MCP Server (`printctl mcp`)**: Plug `printctl mcp` into **Claude Desktop, Cursor, DeepSeek Agent, or Open-Interpreter**. Your AI immediately gains the physical ability to read paper documents from the scanner and print solutions onto paper!

---

## ⚡ 10-Second Quick Start

### One-Click Install (Linux / macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/boheastill/printctl/main/install.sh | sh
```

*(Windows users: download `printctl.exe` directly from [Releases](https://github.com/boheastill/printctl/releases))*.

---

## 🛠️ Usage Examples

### Surface 1: CLI Commands

```bash
# 1. Auto-discover network printers in 0.2s:
printctl discover

# 2. Optical scan to A4 PDF or JPG:
printctl scan ./contract.pdf
printctl scan ./photo_600dpi.jpg --dpi 600

# 3. Print any image or PDF (anti-fail auto layout):
printctl print ./photo.png
printctl print ./document.pdf

# 4. Foolproof manual double-sided (duplex) printing:
printctl print page1.png page2.png --duplex

# 5. Telemetry status in JSON:
printctl status --json
```

---

### Surface 2: Local REST API Server (For App Integration)

Run `printctl serve` to expose standard HTTP endpoints for your apps:

```bash
printctl serve --port 8080
```

* `GET  /api/discover` : Returns discovered printers in JSON.
* `GET  /api/status`   : Returns scanner/printer health.
* `GET  /api/scan?dpi=300&format=pdf` : Triggers optical scan and streams back binary PDF/JPG.
* `POST /api/print`    : Post binary image/PDF to physically print.

---

### Surface 3: AI Agent MCP Integration (Claude / Cursor / DeepSeek)

Add `printctl` to your Claude Desktop or MCP client config (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "printer": {
      "command": "printctl",
      "args": ["mcp"]
    }
  }
}
```

Now your AI Agent natively possesses:
* `printer_scan(output_path, dpi)`: Look at physical paper on the desk.
* `printer_print(file_path)`: Print code, tickets, or generated documents to the physical world.
* `printer_status()`: Check if paper or ink is ready.

---

## 📍 Boundary of Service

```
 ┌──────────────────────────────────────┐     ┌───────────────────────────────────────────────────────┐
 │       【Prerequisites (Hardware)】    │ ──> │               【printctl Scope (Our Zone)】           │
 ├──────────────────────────────────────┤     ├───────────────────────────────────────────────────────┤
 │ 1. Unbox and remove orange tape      │     │ 1. Zero-config sub-second local discovery             │
 │ 2. Install printheads & fill ink     │     │ 2. Anti-fail layout (Alpha flatten, 5mm safe margin)  │
 │ 3. Complete ink charging             │     │ 3. Pure driverless IPP Everywhere printing            │
 │ 4. Join printer to home/office Wi-Fi │     │ 4. Pure driverless eSCL REST scanning (75-600 DPI)    │
 │                                      │     │ 5. Headless REST API & AI MCP Server integration      │
 └──────────────────────────────────────┘     └───────────────────────────────────────────────────────┘
```

---

## 🧩 Under the Hood (Technical Highlights)

* **Zero-Daemon Architecture**: Pure Go single binary (`CGO_ENABLED=0`), 0 resident memory when idle, zero root privileges needed.
* **Anti-Fail In-Memory Layout**: Decodes images into memory, flattens alpha channels onto `#FFFFFF` white (no black squares), clamps 5mm hardware margins to guarantee zero pagination overflow.
* **Native AI MCP Server**: Built-in JSON-RPC 2.0 stdio engine for instant integration with Claude Desktop, Cursor, and DeepSeek.
* **Sub-Second Auto-Discovery**: 254 non-blocking Goroutines sweep local subnets to resolve printer IP in `< 0.2s`.
* **Standard Wire Protocols**: Communicates over **eSCL 2.9 REST (Port 80)** and **IPP Everywhere RFC 8011 (Port 631)**.

📖 **For deep protocol state machines, packet frames, and mathematical layout proofs, read the [Architecture Whitepaper](docs/ARCHITECTURE.md).**

---

## 📄 License
MIT License. Free for open-source, personal, and commercial software.

---

**Author**: [Bohea](https://boheastill.com) — independent industrial software engineer (Shenzhen). Operator HMIs · device integration · machine data into your customer's ERP. More runnable demos & engineering notes on the site.
