# 事故报告：G3030 纯光栅机型静默丢弃 PDF/PNG，printctl 误报成功 🚨

> **日期**：2026-08-23 · **机型**：Canon G3030 series (G3833) @ `192.168.31.202` · **版本**：printctl v0.0.2-patch
> **结论先行**：该机型 IPP 能力表中 **没有 `application/pdf` 也没有 `image/png`**（仅 `image/jpeg` / `image/urf` / `image/pwg-raster` / `application/octet-stream`）。printctl 直送不支持的格式时，打印机**接受作业后静默丢弃，不留任何失败记录**，而 printctl 报 `spooled successfully`。用户侧表现 = 命令成功 + 无纸 + 无错。
> **本文用途**：修复工单 + 已验证的 workaround。附全部复现命令与验证过的 IPP 探测代码，可直接取用。

---

## 1. 现象与排查时间线

| 步骤 | 命令/动作 | 结果 |
|---|---|---|
| 1 | `printctl print --ip 192.168.31.202 doc.pdf`（Chrome 生成的 711KB A4 PDF） | `[+] Print job spooled successfully!` → **无出纸** |
| 2 | `printctl status --ip 192.168.31.202` | `Online: true, Scanner: Idle, Message: Ready`（排除脱机/卡纸） |
| 3 | 换 PNG 探针（2480×3508） | 同样"成功" → **无出纸**（排除 PDF 编码问题） |
| 4 | 原始 IPP `Get-Jobs`（not-completed / completed 各查一次） | **两个列表皆空**：作业被接受后蒸发，无失败痕迹 |
| 5 | 原始 IPP `Get-Printer-Attributes` | `document-format-supported = [application/octet-stream, image/jpeg, image/urf, image/pwg-raster]` —— **根因实锤** |

## 2. 根因

G3030 是**纯光栅机型，无 PDF 解释器**（AirPrint 最低要求集的一类实现：只吃 JPEG / URF / PWG-Raster）。它对 `Create-Job` + 不支持的 `document-format` 的处理是：HTTP 层接受，作业层丢弃，`Get-Jobs` 两头（排队/已完成）都不留尸体。**静默失败是最恶劣的失败模式**——用户只能靠纸面上有没有纸来判断。

## 3. 附带发现的第二、三个坑

1. **多文件批量打印默认带 duplex 属性**：`printctl print f1.jpg f2.jpg ...` 在无双面器的 G3030 上直接 `IPP error 0x0507 (server-error-operation-not-supported)`，一个字节都没送进去。
2. **单作业槽**：该机忙时（打印中）`Create-Job` 返回 `0x0507`，看起来像"不支持此操作"，实际是"忙"。实测每页物理打印 ≈ 18s，需退避重试。

## 4. 已验证的 Workaround（本次 8/8 页出纸）

链路：**任意文档 → Chrome 无头转 PDF → PyMuPDF 逐页 JPEG@220dpi → 逐页重试进槽**。

```bash
# PDF → JPEG（PEP 668 环境用 venv；注意新 API 是 import pymupdf，fitz 已弃用）
python3 -m venv ~/.local/bhprint-venv && ~/.local/bhprint-venv/bin/pip install pymupdf
~/.local/bhprint-venv/bin/python - <<'PY'
import pymupdf
doc = pymupdf.open('in.pdf')
for i, page in enumerate(doc, 1):
    page.get_pixmap(dpi=220).save(f'p{i:02d}.jpg', jpg_quality=88)  # A4 → 1818×2573
PY

# 逐页重试进槽（关键：单槽设备忙时拒收，重试 + 间隔）
for f in p*.jpg; do
  for try in $(seq 8); do
    printctl print --ip 192.168.31.202 "$f" 2>&1 | grep -q "spooled successfully" && break
    sleep 12
  done
  sleep 18   # 等本页物理走完
done
```

现成脚本：`~/bin/bh-print <file.html|file.md>` 封装了全链路（含 md 包装、venv 懒初始化）。

## 5. 修复工单（给处理本事的 agent）

### BUG-1（P0）：不探测格式能力，静默丢任务还报成功
- **位置**：`print` 命令的作业提交路径。
- **修法**：提交前 `Get-Printer-Attributes` 读 `document-format-supported`（按 printer-uuid 或 IP 缓存）。目标格式不在列时**自动转码**：
  - PDF → 首选生成 `image/pwg-raster`（机型 `pwg-raster-document-type-supported: srgb_8, sgray_8`），JPEG 逐页为兜底；
  - PNG/WebP → 转 JPEG（机型 JPEG 原生支持）。
- **验收**：G3030 上 `printctl print doc.pdf` 出纸；`--json` 输出含实际发送的 document-format 与转码标记。

### BUG-2（P1）：多文件默认 duplex，无双面器设备整单 0x0507
- **修法**：探测 `sides-supported`（本机为空）；无双面能力时剥掉 sides 属性走单面，或至少失败自动降级重发，而不是整单失败。

### BUG-3（P1）："spooled successfully" 未回查 job-state
- **修法**：Create-Job 成功后轮询 `Get-Jobs`：`job-state` 到 `processing(5)` 或 `completed(9)` 才报成功；`aborted(8)` / `pending-held(4)` 如实报错并带 `job-state-reasons`。当前"成功"只等于"HTTP 200 收下了"，与物理出纸无关——这是本次用户被误导的直接原因。

### 增强（P2）：单槽设备忙时 0x0507 自动退避重试
- 低价喷墨普遍单作业槽。识别"忙"型 0x0507（可先 Create-Job 前查 `printer-state`），内置指数退避（12s 起），而不是把设备内部忙碌当成协议错误抛给用户。

## 6. 验证过的 IPP 探测代码（可直接复用为能力探测模块）

```python
import struct, urllib.request

def ipp_attr(tag, name, value):
    nb, vb = name.encode(), value.encode()
    return bytes([tag]) + struct.pack('>H', len(nb)) + nb + struct.pack('>H', len(vb)) + vb

def printer_attributes(ip):
    body = b'\x02\x00\x0b\x00' + b'\x00\x01\x00\x02'      # IPP 2.0 / Get-Printer-Attributes
    body += b'\x01'
    body += ipp_attr(0x47, 'attributes-charset', 'utf-8')
    body += ipp_attr(0x48, 'attributes-natural-language', 'en')
    body += ipp_attr(0x45, 'printer-uri', f'ipp://{ip}:631/ipp/print')
    body += b'\x03'
    req = urllib.request.Request(f'http://{ip}:631/ipp/print', data=body,
                                 headers={'Content-Type': 'application/ipp'})
    return urllib.request.urlopen(req, timeout=10).read()   # 解析时注意 value-tag 0x49=mimeMediaType

def get_jobs(ip, which='not-completed'):                    # which: not-completed / completed
    body = b'\x02\x00\x0a\x00' + b'\x00\x01\x00\x01'        # Get-Jobs
    body += b'\x01'
    body += ipp_attr(0x47, 'attributes-charset', 'utf-8')
    body += ipp_attr(0x48, 'attributes-natural-language', 'en')
    body += ipp_attr(0x45, 'printer-uri', f'ipp://{ip}:631/ipp/print')
    body += ipp_attr(0x44, 'which-jobs', which)
    body += b'\x03'
    req = urllib.request.Request(f'http://{ip}:631/ipp/print', data=body,
                                 headers={'Content-Type': 'application/ipp'})
    return urllib.request.urlopen(req, timeout=10).read()
```

## 7. 现场证据（Get-Printer-Attributes 关键片段）

```
document-format-supported : application/octet-stream, image/jpeg, image/urf, image/pwg-raster
document-format-default   : application/octet-stream
pwg-raster-document-type-supported : srgb_8, sgray_8
media-default             : iso_a4_210x297mm
printer-state-reasons     : none
marker-names              : Black(PGBK), Cyan, Magenta, Yellow, MC   ← 墨水正常，非耗材问题
printer-uuid              : urn:uuid:00000000-0000-1000-8000-001105619408
```
