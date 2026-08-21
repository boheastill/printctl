# printctl 🖨️⚡ (中文说明)

> **“我们把电脑与打印机直连的‘脏活’全包了，让你的程序和 AI Agent 瞬间具备物理输入输出能力。”**  
> 一个零依赖、仅 5MB 的纯 Go 单静态二进制工具。将任何现代网络打印机/扫描仪抽象为 **CLI 命令行**、**REST API 微服务** 以及 **AI Agent (MCP) 物理技能**。**彻底告别 CUPS、SANE 与厂商流氓全家桶。**

🌐 **[English Documentation](README.md)**

---

## 💥 为什么需要它？（痛点暴击对比）

如果你曾经尝试在 **Linux 服务器、树莓派、NAS、Docker 容器或 AI 智能体** 里连接打印机，你一定经历过这种绝望：

```
传统 Linux 打印与扫描方式（折磨之路）：
1. sudo apt install cups sane sane-airscan avahi-daemon ... (吃掉 500MB 硬盘)
2. 打开庞大的 CUPS 网页后台 (localhost:631)，在几百个玄学 PPD 驱动列表里盲猜型号
3. 打印时疯狂报错 "Filter Failed"、透明图片打成大黑底、多吐一张白纸
4. 扫描时 scanimage -L 提示 "No SANE devices found"，配置群组权限折腾两小时
5. 常驻后台服务永远占着几十兆内存...

═══════════════════════════════════════════════════════════════════════════════════

使用 printctl 的方式（极客之路）：
1. 下载 5MB 单文件二进制（零依赖，无需 root，无任何后台守护进程）
2. 只要打印机连上了局域网 Wi-Fi，敲一行命令直接出纸 / 扫图：

   $ printctl scan ./contract.pdf      # 自动找机，直接从扫描台拉取高清 A4 PDF
   $ printctl print ./photo.png        # 自动防切边、自动白底拍平，直接物理出纸
```

---

## 🎯 printctl 提供的三大消费接口（三大形态）

```
                     【你的应用程序 / 业务系统 / AI 大脑】
       (CLI 脚本 / Cron 定时器 / Python/Node 后端 / DeepSeek / Claude / Cursor)
                                      │
        ┌─────────────────────────────┼─────────────────────────────┐
        ▼                             ▼                             ▼
   【1. CLI 命令行形态】       【2. REST API 接口形态】      【3. AI Agent (MCP) 形态】
   适用于运维、终端极客        'printctl serve --port 8080'    'printctl mcp' (stdio)
   与自动化 Bash 脚本。         轻量 HTTP 微服务，方便集成     让大模型直接拥有操控
                               到你自己的权限与业务系统。      物理打印机/扫描仪的技能。
        └─────────────────────────────┬─────────────────────────────┘
                                      │
                 (Zero-Config eSCL REST & IPP Everywhere)
                                      ▼
                      【物理网络打印机 (局域网直连)】
                 (佳能 Canon、惠普 HP、爱普生 Epson、兄弟 Brother 等)
```

1. **CLI 命令行引擎**：原生支持 `--json` 输出，与 Unix Pipeline 管道完美配合。
2. **本地无头 REST API 服务 (`printctl serve`)**：启动一个极轻量的本地微服务，你的 Node.js、Python、Go、Java 后端系统可以通过标准 HTTP REST 接口完成打印和扫描，方便你集成进自己的权限管理系统或业务流。
3. **AI 原生技能 / MCP 服务 (`printctl mcp`)**：原生支持 **Model Context Protocol (MCP)**。直接配置进 **Claude Desktop、Cursor、DeepSeek Agent、Open-Interpreter**，你的 AI 即可自动感知物理纸张内容并打印输出！

---

## ⚡ 10 秒极速上手

### 一键安装 (Linux / macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/boheastill/printctl/main/install.sh | sh
```

*(Windows 用户可在 [Releases 页面](https://github.com/boheastill/printctl/releases) 直接下载 `printctl.exe` 使用)*

---

## 🛠️ 使用方式全解

### 形态 1：CLI 终端命令

```bash
# 1. 自动搜索局域网打印机（0.2秒并发探测）：
printctl discover

# 2. 扫描纸张为标准 A4 PDF 或 JPG：
printctl scan ./contract.pdf
printctl scan ./photo_600dpi.jpg --dpi 600

# 3. 打印图片或 PDF 文档（内置防翻车排版：自动居中、等比缩放、白底拍平）：
printctl print ./photo.png
printctl print ./document.pdf

# 4. JSON 格式遥测状态：
printctl status --json
```

---

### 形态 2：本地 REST API 服务（供后端/业务系统集成）

运行后台守护：
```bash
printctl serve --port 8080
```

* `GET  /api/discover` : 返回局域网发现的打印机列表 (JSON)。
* `GET  /api/status`   : 查询打印机与扫描仪在线就绪状态。
* `GET  /api/scan?dpi=300&format=pdf` : 触发物理扫描，直接以二进制流返回 PDF/JPG。
* `POST /api/print`    : 接收二进制图片或 PDF 文件流并直接物理出纸。

---

### 形态 3：AI Agent MCP 技能接入 (Claude / Cursor / DeepSeek)

在 Claude Desktop 或 MCP 客户端配置文件（如 `claude_desktop_config.json`）中添加：

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

你的 AI 助手即可原生调用三大物理工具：
* `printer_scan(output_path, dpi)`：读取桌面上的物理纸张/手写笔记。
* `printer_print(file_path)`：将代码、解答或生成报告直接打印到物理世界。
* `printer_status()`：检查打印机状态与就绪情况。

---

## 📍 服务区间与能力边界定义

```
 ┌──────────────────────────────────────┐     ┌───────────────────────────────────────────────────────┐
 │       【前置物理准备区间 (Prerequisite)】     │ ──> │               【printctl 核心服务区间 (Our Scope)】    │
 ├──────────────────────────────────────┤     ├───────────────────────────────────────────────────────┤
 │ 1. 拆箱与撕除胶带                    │     │ 1. 局域网毫秒级设备自发现 (Zero-Config Discovery)     │
 │ 2. 安装打印头与灌注墨水              │     │ 2. 图像智能规整与防翻车排版 (Alpha拍平/边距/防截断)   │
 │ 3. 完成硬件初始化充墨                │     │ 3. 纯 Linux 免驱直连打印 (IPP Everywhere 直传)        │
 │ 4. 首次配网（打印机连入家庭 Wi-Fi）  │     │ 4. 纯 Linux 免驱高清扫描 (eSCL 75~600 DPI 直拉流)     │
 │                                      │     │ 5. 无头 REST API 与 AI MCP Server 原生集成            │
 └──────────────────────────────────────┘     └───────────────────────────────────────────────────────┘
```

---

## 🧩 技术揭秘与硬核亮点 (Under the Hood)

* **零守护进程架构 (Zero-Daemon)**：单文件纯静态 Go 编译，空闲时 0 内存占用，无需 root 权限与系统常驻后台。
* **纯内存防翻车排版引擎**：在内存中直接将透明度 Alpha 拍平为纯白底色（杜绝透明 PNG 打成大黑底），内置 5mm 物理安全边距，数学上杜绝多吐白纸。
* **原生 AI MCP 协议引擎**：内置标准 JSON-RPC 2.0 stdio 管道，大模型（Claude / Cursor / DeepSeek）无缝即插即用。
* **亚秒级 254 协程并发自发现**：自动扫描本地 `/24` 子网，200 毫秒内锁定可用打印机 IP。
* **严格遵循公开行业标准**：**eSCL 2.9 (HTTP 80)** 扫描 + **IPP Everywhere RFC 8011 (HTTP 631)** 打印，通杀佳能、惠普、爱普生等绝大多数网络机型。

📖 **查看完整的协议状态机、数据帧格式与数学推导，请阅读 [技术架构白皮书 (docs/ARCHITECTURE.md)](docs/ARCHITECTURE.md)。**

---

## 📄 开源协议
MIT License. 自由用于开源、个人与商业项目。
