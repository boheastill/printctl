# 佳能 G3833 (Canon G3030 Series) 硬件开放协议与 Linux 直连开发规范

本文档记录针对 **佳能 PIXMA G3833 连供一体机**（固件与驱动体系内部标识为 `Canon G3030 series`）进行的网络端口、行业公开标准协议实测调研结果，以及在 **纯 Linux 环境下免官方驱动直连操作（打印、扫描、状态监听、设备自发现）** 的完整规范与代码示例。

---

## 目录
1. [设备基本信息与网络概况](#1-设备基本信息与网络概况)
2. [开放端口与支持协议全景](#2-开放端口与支持协议全景)
3. [免驱网络扫描标准 (eSCL / AirScan)](#3-免驱网络扫描标准-escl--airscan)
4. [免驱网络打印标准 (IPP / RAW 9100)](#4-免驱网络打印标准-ipp--raw-9100)
5. [局域网设备自动发现机制 (Auto-Discovery)](#5-局域网设备自动发现机制-auto-discovery)
6. [IP 固化与网络稳定性建议](#6-ip-固化与网络稳定性建议)
7. [开箱即用的 Linux Python 脚本套件](#7-开箱即用的-linux-python-脚本套件)

---

## 1. 设备基本信息与网络概况

* **市场型号**：Canon PIXMA MegaTank G3833 (中国大陆版本，G3830 系列衍生型号)
* **固件产品标识**：`Canon G3030 series`
* **硬件序列号 (Serial)**：`001105619408`
* **设备 UUID**：`00000000-0000-1000-8000-001105619408`
* **认证体系**：Mopria Certified (Scan v1.4, Print), Apple AirPrint, IPP Everywhere
* **出厂默认网络协议**：DHCP 动态获取 IPv4 / IPv6，内置嵌入式 HTTP/HTTPS 服务器

---

## 2. 开放端口与支持协议全景

经过 TCP/UDP 深度端口扫描，设备在局域网内开放了以下标准网络服务端口：

| 端口 | 传输协议 | 运行服务 / 行业规范 | 说明与用途 |
| :--- | :--- | :--- | :--- |
| **80** | TCP | **HTTP / eSCL (AirScan) & Web UI** | 1. eSCL 免驱 RESTful 扫描端点<br>2. 打印机内置管理控制台 |
| **443** | TCP | **HTTPS / eSCL (AirScan) TLS** | 加密传输通道（使用内置自签名证书） |
| **631** | TCP | **IPP (Internet Printing Protocol)** | IPP Everywhere / RFC 8011，Linux CUPS 与跨平台免驱打印核心通道 |
| **9100** | TCP | **RAW / JetDirect (AppSocket)** | 裸流直接打印端口，支持将 PostScript / 打印流直接管道传输到此端口 |
| **515** | TCP | **LPD / LPR (Line Printer Daemon)** | 传统 Unix/Linux 标准打印机后台服务 (RFC 1179) |
| **5353** | UDP | **mDNS / Bonjour (Zeroconf)** | 局域网服务广播 (`_ipp._tcp`, `_uscan._tcp`, `_pdl-datastream._tcp`) |

---

## 3. 免驱网络扫描标准 (eSCL / AirScan)

打印机固件遵循 **eSCL 2.9 (Mopria Scan 1.4)** 标准协议，完全通过标准的 HTTP RESTful API 进行交互，传输格式为 XML 和二进制数据流。

### 3.1 核心 RESTful 端点

#### (1) 查询扫描仪能力：`GET /eSCL/ScannerCapabilities`
* **请求**：`GET http://<PRINTER_IP>/eSCL/ScannerCapabilities`
* **返回**：XML 描述文档。
* **硬件实测能力**：
  * **输入来源 (InputSource)**：`Platen`（平铺扫描台）
  * **支持分辨率 (DPI)**：`75`, `100`, `150`, `200`, `300` (默认), `600` (最大光学分辨率)
  * **支持色彩模式 (ColorMode)**：`RGB24` (24-bit 全彩, 默认), `Grayscale8` (8-bit 灰度)
  * **支持输出格式**：`image/jpeg` (默认), `application/pdf`
  * **最大扫描幅面**：宽 2550 像素 $\times$ 高 3508 像素 (A4 @ 300 DPI)

#### (2) 查询实时工作状态：`GET /eSCL/ScannerStatus`
* **请求**：`GET http://<PRINTER_IP>/eSCL/ScannerStatus`
* **返回 XML 状态值**：
  * `<pwg:State>`：`Idle`（空闲就绪）、`Processing`（正在扫描中）、`Testing`、`Stopped`
  * `<scan:Jobs>`：包含最近任务的 UUID、状态（`JobCompletedSuccessfully` 等）。

#### (3) 提交扫描任务：`POST /eSCL/ScanJobs`
* **请求头**：`Content-Type: text/xml`
* **Body (XML 参数配置)**：
```xml
<?xml version="1.0" encoding="UTF-8"?>
<scan:ScanSettings xmlns:scan="http://schemas.hp.com/imaging/escl/2011/05/03" xmlns:pwg="http://www.pwg.org/schemas/2010/12/sm">
   <pwg:Version>2.0</pwg:Version>
   <scan:Intent>Document</scan:Intent>
   <pwg:ScanRegions>
      <pwg:ScanRegion>
         <pwg:Height>3508</pwg:Height>
         <pwg:Width>2550</pwg:Width>
         <pwg:XOffset>0</pwg:XOffset>
         <pwg:YOffset>0</pwg:YOffset>
      </pwg:ScanRegion>
   </pwg:ScanRegions>
   <pwg:DocumentFormat>image/jpeg</pwg:DocumentFormat>
   <scan:ColorMode>RGB24</scan:ColorMode>
   <scan:XResolution>300</scan:XResolution>
   <scan:YResolution>300</scan:YResolution>
   <pwg:InputSource>Platen</pwg:InputSource>
</scan:ScanSettings>
```
* **响应**：HTTP `201 Created`，响应头中包含 `Location: http://<PRINTER_IP>/eSCL/ScanJobs/<JOB_UUID>`。

#### (4) 拉取扫描文件流：`GET /eSCL/ScanJobs/<JOB_UUID>/NextDocument`
* **请求**：直接请求由 Location 返回的文档下载地址。
* **响应**：HTTP `200 OK`，二进制图片/PDF 数据流。

---

## 4. 免驱网络打印标准 (IPP / RAW 9100)

### 4.1 IPP (Internet Printing Protocol, 端口 631)
* **协议标准**：IPP Everywhere (RFC 8010/8011)
* **端点路径**：`http://<PRINTER_IP>:631/ipp/print` 或 `ipp://<PRINTER_IP>/ipp/print`
* **在 Linux CUPS 下的配置方法**：
  无需下载任何厂商驱动，执行以下命令即可添加免驱网络打印机：
  ```bash
  lpadmin -p Canon_G3833 -E -v ipp://<PRINTER_IP>/ipp/print -m everywhere
  ```
  或者使用 CUPS Web 控制台 (`http://localhost:631`) 自动探测添加为 “IPP Everywhere” 设备。

### 4.2 RAW / JetDirect 裸端口 (端口 9100)
* 适用于无驱动管道直推：
  ```bash
  # 将打印文件直接打入网络端口
  cat document.ps | nc <PRINTER_IP> 9100
  ```

---

## 5. 局域网设备自动发现机制 (Auto-Discovery)

在 Linux 环境下无需硬编码 IP，有两种可靠的自动发现方案：

### 5.1 eSCL 快速多线程局域网探测 (推荐)
通过向当前网段（如 `192.168.31.0/24`）并发发送轻量级 `/eSCL/ScannerCapabilities` 请求，平均耗时仅 **1~2 秒**，无论打印机 IP 如何变化均能即时定位。

### 5.2 mDNS / Bonjour 广播发现
打印机会在 UDP 5353 发送广播：
* 服务名：`Canon G3030 series._ipp._tcp.local`、`Canon G3030 series._uscan._tcp.local`
* 主机名：`c0AAFE300000.local`（基于 MAC 地址后 6 位派生）

---

## 6. IP 固化与网络稳定性建议

由于路由器 DHCP 租期可能导致打印机 IP 漂移，推荐采用以下方式之一固化 IP：

1. **路由器 DHCP 静态绑定（最推荐）**：
   在路由器管理后台将打印机 MAC 地址与固定 IP（例如 `192.168.31.200`）绑定。
2. **打印机 Web 管理后台设定静态 IP**：
   通过浏览器访问 `http://<PRINTER_IP>` -> 进入网络设置 -> 将 IPv4 配置为手动固定 IP。

*(注：访问 Web 管理后台时提示的“安装根证书”仅用于自签名 HTTPS，家庭局域网内直接走 HTTP 80 端口即可，无需安装证书)*。

---

## 7. 开箱即用的 Linux Python 脚本套件

### 7.1 自动寻机与全自动扫描脚本 (`scan_tool.py`)

```python
#!/usr/bin/env python3
"""
Canon G3833 / G3030 Series Pure Linux Direct Scanner Tool
Zero-driver, standalone eSCL / AirScan client.
"""

import concurrent.futures
import time
import urllib.request
from typing import Optional

def discover_printer_ip(subnet_prefix: str = "192.168.31.") -> Optional[str]:
    """在局域网 /24 网段内并发探测支持 eSCL 的打印机 IP"""
    def probe(ip):
        url = f"http://{ip}/eSCL/ScannerCapabilities"
        try:
            req = urllib.request.Request(url, headers={"User-Agent": "CanonScannerDiscovery"})
            with urllib.request.urlopen(req, timeout=0.6) as resp:
                if resp.status == 200:
                    data = resp.read(1024).decode("utf-8", errors="ignore")
                    if "Canon" in data or "ScannerCapabilities" in data:
                        return ip
        except Exception:
            pass
        return None

    ips = [f"{subnet_prefix}{i}" for i in range(1, 255)]
    with concurrent.futures.ThreadPoolExecutor(max_workers=64) as executor:
        results = list(filter(None, executor.map(probe, ips)))
    return results[0] if results else None

def scan_to_file(printer_ip: str, output_path: str = "scanned.jpg", dpi: int = 300, color_mode: str = "RGB24"):
    """
    通过 eSCL 协议向打印机下发扫描任务并下载文件
    :param printer_ip: 打印机 IP 地址
    :param output_path: 保存的文件路径 (.jpg / .pdf)
    :param dpi: 分辨率 (75, 100, 150, 200, 300, 600)
    :param color_mode: 色彩模式 ("RGB24" 或 "Grayscale8")
    """
    doc_format = "application/pdf" if output_path.lower().endswith(".pdf") else "image/jpeg"
    
    # 300 DPI 下的标准 A4 像素尺寸 (宽 2550 x 高 3508)
    height = int(3508 * (dpi / 300))
    width = int(2550 * (dpi / 300))

    xml_payload = f"""<?xml version="1.0" encoding="UTF-8"?>
<scan:ScanSettings xmlns:scan="http://schemas.hp.com/imaging/escl/2011/05/03" xmlns:pwg="http://www.pwg.org/schemas/2010/12/sm">
   <pwg:Version>2.0</pwg:Version>
   <scan:Intent>Document</scan:Intent>
   <pwg:ScanRegions>
      <pwg:ScanRegion>
         <pwg:Height>{height}</pwg:Height>
         <pwg:Width>{width}</pwg:Width>
         <pwg:XOffset>0</pwg:XOffset>
         <pwg:YOffset>0</pwg:YOffset>
      </pwg:ScanRegion>
   </pwg:ScanRegions>
   <pwg:DocumentFormat>{doc_format}</pwg:DocumentFormat>
   <scan:ColorMode>{color_mode}</scan:ColorMode>
   <scan:XResolution>{dpi}</scan:XResolution>
   <scan:YResolution>{dpi}</scan:YResolution>
   <pwg:InputSource>Platen</pwg:InputSource>
</scan:ScanSettings>"""

    print(f"[*] 正在向 http://{printer_ip}/eSCL/ScanJobs 下发扫描任务 ({dpi} DPI, {color_mode})...")
    req = urllib.request.Request(
        f"http://{printer_ip}/eSCL/ScanJobs",
        data=xml_payload.encode("utf-8"),
        headers={"Content-Type": "text/xml"}
    )

    with urllib.request.urlopen(req) as resp:
        location = resp.headers.get("Location")
        if not location.startswith("http"):
            next_url = f"http://{printer_ip}" + location + "/NextDocument"
        else:
            next_url = location + "/NextDocument"

    print(f"[*] 任务已创建，等待打印机光学扫描完成...")
    for _ in range(30):
        time.sleep(2)
        try:
            with urllib.request.urlopen(next_url, timeout=15) as doc_resp:
                if doc_resp.status == 200:
                    data = doc_resp.read()
                    with open(output_path, "wb") as f:
                        f.write(data)
                    print(f"[+] 扫描成功！文件已保存至: {output_path} ({len(data)} 字节)")
                    return
        except Exception:
            pass
    raise TimeoutError("扫描超时或读取失败")

if __name__ == "__main__":
    ip = discover_printer_ip()
    if not ip:
        print("[-] 未在局域网内发现打印机，请检查网络连接")
    else:
        print(f"[+] 发现打印机: {ip}")
        scan_to_file(ip, "output.jpg", dpi=300)
```
