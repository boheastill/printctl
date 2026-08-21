# printctl 🖨️⚡

[![Release](https://img.shields.io/github/v/release/boheastill/printctl?include_prereleases)](https://github.com/boheastill/printctl/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/boheastill/printctl)](https://goreportcard.com/report/github.com/boheastill/printctl)

> **告别 500MB 的 CUPS、SANE 与流氓厂商全家桶。**  
> 一个仅 5MB 的纯 Go 单二进制文件，让你在 Linux / 树莓派 / NAS / Docker 上，**零配置、零驱动、零后台守护进程**，直接像用 `curl` 一样扫描与打印。

[快速开始](#-10-秒极速上手) | [为什么做这个 (暴击对比)](#-为什么你需要它-痛点暴击对比) | [典型使用场景](#-典型实战场景) | [English README](#english)

---

## 💥 为什么你需要它？（痛点暴击对比）

如果你曾经尝试在 **Linux 服务器、树莓派、软路由、Docker 容器或无桌面环境** 里连打印机，你一定经历过这种绝望：

```
传统 Linux 打印与扫描方式（折磨之路）：
1. sudo apt install cups sane sane-airscan avahi-daemon ... (吃掉 500MB 硬盘)
2. 打开庞大的 CUPS 网页后台 (localhost:631)，在几百个玄学 PPD 驱动列表里盲猜型号
3. 打印时疯狂报错 "Filter Failed"、"media-empty"、透明图片打成大黑底、多吐一张白纸
4. 扫描时 scanimage -L 提示 "No SANE devices found"，配置群组权限折腾两小时
5. 两个常驻后台服务永远占着几十兆内存...

═══════════════════════════════════════════════════════════════════════════════════

使用 printctl 的方式（极客之路）：
1. 下载 5MB 单文件二进制（零依赖，无需 root，无任何后台守护进程）
2. 只要打印机连上了局域网 Wi-Fi，敲一行命令直接出纸 / 扫图：

   $ printctl scan ./contract.pdf      # 自动找机，直接从扫描台拉取高清 A4 PDF
   $ printctl print ./photo.png        # 自动防切边、自动白底拍平，直接物理出纸
```

---

## 🎯 我们的目标用户是谁？

* **Linux 服务器 / 树莓派 / NAS / Homelab 玩家**：受够了 CUPS 和 SANE 的庞大与脆弱，想要最干净、开箱即用的打印/扫描 CLI。
* **Docker / CI/CD 自动化开发者**：需要在轻量容器里实现自动打印送货单、发票、PDF 报告，无需在容器里塞整个 CUPS 协议栈。
* **AI Agent / 自动化工程师**：让 DeepSeek、Gemini、Claude 等大模型拥有**“纸质物理世界 I/O”**——读纸上写的内容，把推理结果打印出来。
* **极简主义者**：厌恶官方动辄 500MB 还要注册登录、强制绑定云账户的流氓驱动。

---

## ⚡ 10 秒极速上手

### 一键安装 (Linux / macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/boheastill/printctl/main/install.sh | sh
```

*(Windows 用户可在 [Releases 页面](https://github.com/boheastill/printctl/releases) 直接下载 `printctl.exe` 使用)*

---

## 🛠️ 核心常用命令

### 1. 自动搜索局域网打印机
无需手动查 IP，254 协程并发探测，0.2 秒内锁定设备：
```bash
printctl discover
```

### 2. 纯命令行无损扫描（支持 75 ~ 600 DPI）
```bash
# 扫描为标准 A4 PDF（默认 300 DPI 高清）：
printctl scan ./invoice.pdf

# 扫描为 JPG 图片：
printctl scan ./receipt.jpg

# 600 DPI 极高精度照片扫描：
printctl scan ./photo_hd.jpg --dpi 600
```

### 3. 一键防翻车物理打印
```bash
# 打印图片（内置防翻车引擎：自动居中、等比缩放、透明底自动拍平成白底）：
printctl print ./wallpaper.png

# 打印 PDF 文档：
printctl print ./document.pdf
```

### 4. 机器状态与墨水就绪遥测
```bash
# 人类易读格式：
printctl status

# JSON 格式（专为 Python / Shell / AI Agent 管道设计）：
printctl status --json
```

---

## 💡 典型实战场景

### 场景 1：打造 10 行代码的“AI 纸质交互工作站”
将草稿/手写笔记放进扫描台，让 AI 看懂并自动打印解答：
```bash
# 1. 扫描手写纸
printctl scan /tmp/input.jpg

# 2. 调大模型视觉推理 (Python / Shell) 生成 answer.pdf
python3 -c "import ai_agent; ai_agent.solve('/tmp/input.jpg', '/tmp/answer.pdf')"

# 3. 自动物理出纸
printctl print /tmp/answer.pdf
```

### 场景 2：每早 7:00 自动打印“极客个人晨报”
在 Linux `crontab` 加一行：
```bash
0 7 * * * /usr/local/bin/generate_morning_news.sh && printctl print /tmp/today.pdf
```

### 场景 3：Paperless-ngx / 本地文档一键无损归档
```bash
printctl scan ~/documents/archive_$(date +%Y%m%d_%H%M%S).pdf
```

---

## 🧩 为什么我们不依赖驱动就能跑？（技术揭秘）

现代网络打印机遵循两套**开源/行业公开标准**：
1. **eSCL / Apple AirScan (HTTP REST)**：打印机自身内置微型 Web 服务器，开放标准 XML RESTful 接口。`printctl` 直接调用该接口拉取扫描数据。
2. **IPP Everywhere (RFC 8011)**：基于 HTTP 631 端口的通用打印标准。`printctl` 将任意位图进行 Alpha 拍平与 A4 几何拟合后直推硬件。

**兼容性**：支持佳能 (Canon)、惠普 (HP)、爱普生 (Epson)、兄弟 (Brother) 等 **95% 以上支持 Wi-Fi / AirPrint / Mopria 的现代网络打印机**。

---

<a name="english"></a>
## English Overview

**printctl** is a zero-dependency, driverless print and scan CLI engine in pure Go. It eliminates the need for heavyweight CUPS, SANE, and vendor bloatware by talking directly to your network printer over standard **eSCL (REST)** and **IPP Everywhere (RFC 8011)**.

### Quick Install
```bash
curl -fsSL https://raw.githubusercontent.com/boheastill/printctl/main/install.sh | sh
```

### Features
* **Zero Daemons, Zero Dependencies**: 5MB single static binary (`CGO_ENABLED=0`).
* **All-in-One**: Direct scanning (75-600 DPI, JPG/PDF) and direct printing in one tool.
* **Anti-Fail Layout Engine**: Auto alpha-flattening (no black transparent PNGs), 5mm hardware safe margin clamp.
* **Universal**: Compatible with Canon, HP, Epson, Brother, and any Mopria/AirPrint printer.

---

## 📄 License
MIT License. Free for personal and commercial use.
