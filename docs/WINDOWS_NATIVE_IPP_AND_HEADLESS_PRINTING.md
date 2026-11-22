# Windows 11 免官方驱动直连（Microsoft IPP Class Driver）与无头打印踩坑实录

> **日期**：2026-11-22 · **机型**：Canon PIXMA G3833 (`Canon G3030 series`) · **环境**：Windows 11 Pro 26H2 (`silver`)

本文档记录在 **完全不安装佳能官方驱动与配套软件** 的前提下，仅用 Windows 11 内置协议栈与自带组件打通 **日常 `Ctrl+P` 打印、Agent 自动化批量打印、eSCL 免驱扫描** 全链路时遇到的全部真实坑点，以及与 `printctl` 纯 Go 直推管线的打印速度对比根因。

---

## 1. 为什么 Windows 原生 IPP 比 `printctl` 慢？（实测根因与提速开关）

在打印 14 页图文手册时，`Microsoft IPP Class Driver` 实测约 **15~18 秒/页**，体感明显慢于 `printctl` 直推。抓取 `Get-PrintConfiguration` 的 `PrintCapabilitiesXML / PrintTicket` 后定位到两个叠加根因：

| 维度 | `printctl` (Go 直推) | Windows `Microsoft IPP Class Driver` (默认) | 速度影响 |
| :--- | :--- | :--- | :--- |
| **传输格式** | `image/jpeg`（打印机 ASIC 硬件直解） | `image/pwg-raster`（PackBits 光栅流，主机端光栅化后流式下发） | JPEG 单页仅 **200~400 KB**；600 DPI 彩色 PWG-Raster 单页达 **数 MB ~ 十余 MB**，受限于机身 2.4GHz 单流 Wi-Fi 接收与逐行解包 |
| **默认分辨率** | `300 × 300 DPI` | **`600 × 600 DPI`**（`ns0001:DPI600x600`，该类驱动对 G3030 仅暴露 600 DPI 档位） | 像素量是 300 DPI 的 **4 倍**（单页约 3,500 万像素） |
| **色彩与喷头调度** | 按需 | 默认 **`psk:Color`（彩色模式）** | 只要文档含任何彩色元素（如蓝框/灰底），固件即启用 **`CH-8` 三色染料喷头做多遍往复打印（约 6 ipm）**，而非高速 **`BH-8` 大黑颜料宽喷头单遍直喷（约 11 ipm）** |

### 原生 IPP 下的两个无损提速手段
1. **切黑白模式（最有效，速度约快 2 倍）**：
   日常打纯文档/代码手册时，切为黑白可让数据流降为单通道 `sgray_8` 并强制启用 `BH-8` 宽幅颜料黑喷头：
   ```powershell
   # 设为默认黑白打印（需要打彩图时再改回 $true 或在 Ctrl+P 弹窗临时选彩色）
   Set-PrintConfiguration -PrinterName "Canon G3833 (IPP)" -Color $false
   ```
2. **打印质量选 `Draft`（草稿模式）**：
   驱动能力表原生暴露了 `psk:PageOutputQuality = psk:Draft`（草稿）与 `psk:Standard` / `psk:High`，在 `Ctrl+P` → 更多设置 → 质量中选“草稿”可显著减少喷头往复遍数。

---

## 2. 配网与挂载阶段的坑

### 坑 1：Windows 设置里搜不到打印机，但 IP 明明在线
* **根因 A**：Windows Wi-Fi 网络配置默认为 `Public`（公用网络），系统防火墙直接拦截 mDNS (UDP 5353) 与 WSD (UDP 3702) 设备发现广播；且 `fdPHost` / `FDResPub` 服务默认处于 `Stopped`。
* **根因 B**：在 Windows 设置里走自动 WSD/PnP 发现添加打印机时，Windows Update 有概率顺手拉取厂商配套 UWP 应用（违背“零佳能软件”原则）。
* **正确做法**：无需改网络属性、无需开 WSD 服务，直接用**管理员权限** PowerShell 显式绑定 IPP URL，100% 强制使用微软内置 `Microsoft IPP Class Driver`：
  ```powershell
  Add-Printer -IppURL "http://192.168.31.202:631/ipp/print" -Name "Canon G3833 (IPP)"
  Set-PrintConfiguration -PrinterName "Canon G3833 (IPP)" -PaperSize A4
  ```
  *(注：非管理员会话执行 `Add-Printer` 会报 `HRESULT 0x80070005 Access denied`)*。

### 坑 2：局域网快速并发扫描（<600ms 超时）漏扫在线打印机
* **现象**：对 `192.168.31.0/24` 做 600ms 超时的 TCP `631` / `9100` 全网段扫描，返回 0 台设备；单独连接又立刻通了。
* **根因**：佳能喷墨机进入低功耗待机（Wi-Fi DTIM 节能休眠）后，**首包无线唤醒延迟经常在 800ms ~ 1500ms 之间**。600ms 超时刚好在机器醒来前放弃连接。
* **解法**：自动发现探测超时至少给 **2000ms**，或采用“先发一轮唤醒包、间隔 500ms 再扫第二轮”的两段式探测。

### 坑 3：绑定 `c0AAFE300000.local` 极慢甚至超时
* **实测现象**：本机开启代理虚拟网卡（`tun0`）时，`[System.Net.Dns]::GetHostAddresses("c0AAFE300000.local")` 解析耗时超过 **20 秒**，`curl` 在 3 秒超时内直接失败；而直连 IPv4 `192.168.31.202` 毫秒级响应。
* **解法**：IPP 端口直接绑 IPv4 地址。若路由器未做 DHCP 静态绑定导致 IP 漂移，按不变硬件指纹（MAC `a4:f0:1f:0a:af:e3` 或并发请求 `/eSCL/ScannerCapabilities` 匹配序列号 `001105619408`）重找即可。

### 坑 4：设默认打印机被 Windows 悄悄改回去
* **根因**：Windows 11 默认开启“让 Windows 管理我的默认打印机”（记录最后一次使用的打印机），会导致脚本设好的默认打印机被覆盖。
* **解法**：先关闭动态接管再调用 CIM `SetDefaultPrinter`：
  ```powershell
  Set-ItemProperty -Path "HKCU:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Windows" -Name "LegacyDefaultPrinterMode" -Value 1 -Type DWord -Force
  $p = Get-CimInstance -ClassName Win32_Printer -Filter "Name='Canon G3833 (IPP)'"
  Invoke-CimMethod -InputObject $p -MethodName SetDefaultPrinter
  ```

---

## 3. Agent 无头自动化打印（PDF / HTML）连环坑

在没有安装 Adobe Acrobat / SumatraPDF / Office 的干净 Windows 11 上，让脚本静默把一份 14 页 HTML/PDF 文档打出来，连续踩中三个底层坑：

### 坑 5：在 `PrintDocument.PrintPage` 回调内调 WinRT `PdfPage.RenderToStreamAsync` 吐白纸
* **现象**：在 `.NET` `System.Drawing.Printing.PrintDocument` 的 `PrintPage` 事件脚本块里同步 `Wait()` WinRT 异步方法，直接抛 `AggregateException`，导致 `Image` 为空，如果不立即 `Remove-PrintJob` 就会连吐 14 张白纸。
* **根因**：`PrintDocument.Print()` 在 GDI 打印上下文与 PowerShell Runspace 线程模型下与 WinRT `IAsyncAction` 公寓模型冲突。

### 坑 6：WinRT `Windows.Data.Pdf` 依赖 Direct2D GPU，显卡实例挂起时彻底瘫痪
* **现象**：将 `RenderToStreamAsync` 移出打印回调单独离线渲染 PNG，第 1 页成功，随后报错：
  `The GPU device instance has been suspended. Use GetDeviceRemovedReason to determine the appropriate action.`
* **根因**：Windows 内置的 `Windows.Data.Pdf` 底层强依赖 Direct2D 硬件加速。当笔记本经历过外接显示器热插拔异常或驱动重置导致 GPU 实例挂起时，WinRT PDF 渲染器直接不可用。

### 坑 7（最终稳定解法）：Edge `--kiosk-printing --disable-gpu` 纯 CPU 矢量直出
* **为什么最优**：
  1. Windows 11 必带 `msedge.exe`，**零额外依赖**；
  2. `--disable-gpu` 彻底绕开 Direct2D / 显卡驱动挂起问题，纯 CPU Skia 渲染；
  3. `--kiosk-printing` 跳过打印弹窗，直接将 Chromium 排好的 **原生矢量字形与 CSS `@page` 分页** 送进系统默认打印机（`Microsoft IPP Class Driver`），清晰度远高于先转位图再打印，且 14 页合并为单个 Spooler 作业，不会触发 G3030 单作业槽 `0x0507` 拒收。
* **最小可复用脚本范式**：
  ```powershell
  # 1. 注入自动触发打印与打印后自动关窗脚本（使用独立临时 user-data-dir 互不干扰日常浏览器）
  $html = [IO.File]::ReadAllText($srcHtml, [Text.Encoding]::UTF8)
  $inject = "<script>window.addEventListener('load',()=>setTimeout(()=>window.print(),600));window.addEventListener('afterprint',()=>setTimeout(()=>window.close(),1500));</script></body>"
  [IO.File]::WriteAllText($tempHtml, ($html -replace '</body>', $inject), [Text.Encoding]::UTF8)

  Start-Process "C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe" -ArgumentList @(
      "--kiosk-printing",
      "--disable-gpu",
      "--no-first-run",
      "--no-default-browser-check",
      "--user-data-dir=$tempProfileDir",
      "--app=file:///$($tempHtml -replace '\\','/')"
  )
  ```
* **防浪费纸兜底命令**（发现队列异常第一时间撤单）：
  ```powershell
  Get-PrintJob -PrinterName "Canon G3833 (IPP)" | Remove-PrintJob
  ```

---

## 4. 休眠首印“咔哒响半天”说明

打印机从休眠态收到当天第一份作业时，会先花 **15~30 秒** 执行喷头解锁、抽墨准备与微自洁，机身发出连续机械咔哒声，期间 `PagesPrinted` 保持 `0`。属于正常硬件防堵自护流程，切勿误判为卡死而重复提交作业。
