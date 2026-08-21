# printctl 🖨️✨

[![Release](https://img.shields.io/github/v/release/boheastill/printctl?include_prereleases)](https://github.com/boheastill/printctl/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/boheastill/printctl)](https://goreportcard.com/report/github.com/boheastill/printctl)

> **The Modern Driverless Print & Scan Engine in Pure Go.**  
> 基于第一性原理构建的极简免驱打印与扫描一体化 CLI 引擎。单静态二进制（~5MB），零系统守护进程、零外部依赖。

[English](#english) | [中文说明](#中文说明)

---

## 中文说明

### 📍 服务区间与能力边界定义

在设计哲学上，`printctl` 遵循明确的**职责分界线**：

```
 ┌──────────────────────────────────────┐     ┌───────────────────────────────────────────────────────┐
 │       【前置物理准备区间 (Prerequisite)】     │ ──> │               【printctl 核心服务区间 (Our Scope)】    │
 ├──────────────────────────────────────┤     ├───────────────────────────────────────────────────────┤
 │ 1. 拆箱与撕除胶带                    │     │ 1. 局域网毫秒级设备自发现 (Zero-Config Discovery)     │
 │ 2. 安装打印头与灌注墨水              │     │ 2. 图像智能规整与防翻车排版 (Alpha拍平/边距/防截断)   │
 │ 3. 完成硬件初始化充墨                │     │ 3. 纯 Linux 免驱直连打印 (IPP Everywhere 直传)        │
 │ 4. 首次配网（打印机连入家庭 Wi-Fi）  │     │ 4. 纯 Linux 免驱高清扫描 (eSCL 75~600 DPI 直拉流)     │
 │ 5. 任何设备完成第一次网络握手        │     │ 5. 硬件状态与耗材遥测 (JSON 输出，对 AI / 脚本友好)   │
 └──────────────────────────────────────┘     └───────────────────────────────────────────────────────┘
```

> **服务分界点**：打印机完成物理开箱并连入 Wi-Fi 后，**后续在 Linux / Unix 环境下的所有免驱打印、无损拉流扫描、状态监控与 AI 自动化集成，均由 `printctl` 一站式全权接管。**

---

### ⚡ 核心特性

* **单静态文件 (Single Binary)**：纯 Go 编写 (`CGO_ENABLED=0`)，编译后仅 ~5MB，扔到任何 Linux / 树莓派 / NAS / Docker 就能跑，**彻底告别 CUPS / SANE 庞大系统级守护进程**。
* **打扫一体化 (All-in-One)**：同时打通 **IPP Everywhere (打印)** 与 **eSCL / AirScan (扫描)** 两大行业公开标准。
* **防翻车排版引擎 (Anti-Fail Layout)**：
  * **Alpha 拍平**：透明 PNG 自动合成在纯白物理底色上，永不出现“大黑底”。
  * **物理边距保护**：A4 画布强制预留 5mm 硬件安全区，`math.Floor` 像素严控，数学上杜绝“多吐一张白纸”。
  * **EXIF 方向校正**：手机照片自动转正。
* **毫秒级局域网自发现 (Auto-Discovery)**：254 协程并发探测，0.2 秒内自动定位打印机 IP，无需手动配置。
* **AI Agent / 脚本原生友好**：支持 `--json` 输出与 Unix Pipeline 标准输入输出。

---

### 🚀 一键极速安装

#### Linux / macOS：
```bash
curl -fsSL https://raw.githubusercontent.com/boheastill/printctl/main/install.sh | sh
```

或者直接前往 [Releases 页面](https://github.com/boheastill/printctl/releases) 下载对应架构的单二进制文件。

---

### 📖 使用指南

#### 1. 自动搜索局域网内的打印机：
```bash
printctl discover
```

#### 2. 扫描纸张（支持 75~600 DPI，JPG / PDF）：
```bash
# 默认 300 DPI 彩色扫描为 JPG：
printctl scan ./scanned_doc.jpg

# 高精度 600 DPI 扫描：
printctl scan ./photo_600dpi.jpg --dpi 600

# 直接保存为标准 A4 PDF：
printctl scan ./contract.pdf
```

#### 3. 打印图片或 PDF 文档：
```bash
# 打印任意图片（自动居中、等比缩放、白底拍平）：
printctl print ./wallpaper.png

# 打印 PDF 文档：
printctl print ./document.pdf
```

#### 4. 查看设备在线与硬件状态：
```bash
# 格式化输出：
printctl status

# JSON 输出（供脚本与 AI Agent 解析）：
printctl status --json
```

---

<a name="english"></a>
## English Description

### Overview
`printctl` is a minimalist, driverless print and scan CLI engine written in pure Go. It abstracts any modern network multi-function printer (supporting IPP Everywhere and eSCL / AirScan) into a standard Unix I/O pipeline.

### Key Highlights
* **Zero Dependencies**: Pure Go with `CGO_ENABLED=0`. No CUPS, no SANE, no bloated daemons.
* **All-in-One**: Combines IPP Everywhere (printing), eSCL REST (scanning), and auto-discovery into a single 5MB binary.
* **Anti-Fail Layout Engine**: Automatic alpha-flattening (no black square PNGs), hardware safe-margin bounding box, and EXIF rotation.
* **Universal Compatibility**: Works with Canon, HP, Epson, Brother, and 95%+ modern Wi-Fi printers.

---

## 📄 License
MIT License. Free for open-source and commercial use.
