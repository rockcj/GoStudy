# Freecat 自由猫 · Linux 命令行客户端

一个开箱即用的简易 Linux 客户端：**登录账号 → 拉取订阅 → 启动代理/TUN → 手动选节点**。
不含图形界面，核心是官方使用的 mihomo（Clash Meta）内核，分流规则与 Windows 客户端一致。

已在 **Ubuntu 24.04 (x86_64)** 实测：登录、订阅、HTTP/SOCKS 代理、TUN 全局透明代理、
节点手动切换均通过，可正常访问 Google / YouTube / GitHub。

当前版本：**0.2.0**

---

## 一、原理（为什么能用）

对 `ziyoumaoCore.exe` 与 `app.so` 的静态分析 + 运行态观察得到：

1. Windows 客户端 = Flutter 界面 + Go 内核，内核内嵌 **mihomo**（构建标签 `with_gvisor`），
   TUN 由 gvisor 网络栈提供。
2. 登录/订阅是标准 v2board/Xboard：
   - 登录 `POST /api/v1/passport/auth/login`（返回 `token` 与 `auth_data`）
   - 订阅 `GET /api/v1/client/subscribe?token=...`（UA 用 `ClashforWindows/0.20.39`，返回完整 Clash YAML）
3. **关键坑一**：订阅里的节点域名 `freevip*.freecatnodles.com` 已被 DNS 污染
   （公共 DoH 都返回 `CNAME google.com`），直接用会全部超时。
4. **关键坑二**：官方客户端实际使用的入口不是这些域名，而是
   `aqaenca1.tobigyang.cn` / `bqbencb2.tobigyang.cn` / `cqcencc3.tobigyang.cn` 这类
   **按账号生成的真实域名**。它们通过 GTM 做 DNS 轮换（例：`aqaenca1` → `34.92.4.52`、`35.241.85.185`），
   端口与订阅完全一致。
5. 因此本工具默认把订阅里的 `server:` 统一改写为 **`aqaenca1.tobigyang.cn`**（域名，不是写死的 IP），
   由 DNS 动态解析；官方换 IP 时无需改配置。可用 `--server-host` 换成其它域名或 IP。
6. mihomo 首次启动会尝试从 GitHub 下载 `geoip.metadb`（国内通常失败并退出）。
   安装包已附带 `geo/` 数据，`freecat run` 会自动复制到配置目录。

> 说明：这是对**你自己的账号**做协议级复刻，不修改、不破解官方服务端；不包含账号密码等硬编码凭据。

---

## 二、安装

```bash
tar -xzf freecat-linux-amd64.tar.gz
cd freecat-dist
sudo ./install.sh
```

安装内容：

| 路径 | 说明 |
| --- | --- |
| `/opt/freecat/freecat` | CLI 主程序 |
| `/opt/freecat/mihomo` | mihomo v1.19.33 内核 |
| `/opt/freecat/geo/` | geoip.metadb / geosite.dat |
| `/usr/local/bin/freecat` | 软链接 |
| `/etc/systemd/system/freecat.service` | 可选的 TUN 常驻服务 |

只依赖 glibc，无需安装 Go/Node/Python。

---

## 三、三步开始

```bash
# 1. 登录（令牌保存在 ~/.config/freecat/state.json，权限 600）
freecat login -email 你的邮箱 -password 你的密码

# 2. 拉取订阅与节点（自动把 server 改写为动态域名）
freecat update

# 3a. 仅本地代理（无需 root）
freecat run
#     另开终端：
eval "$(freecat env)"     # 导出 http_proxy/https_proxy
freecat test              # 连通性测试，显示出口 IP

# 3b. 全局 TUN 透明代理（推荐，需 root）
sudo freecat dns on       # 把系统 DNS 指向 127.0.0.1（自动备份/可恢复）
sudo freecat run --tun
#     另开终端直接 curl https://www.google.com 即可，无需设置代理变量
```

停止：`Ctrl+C`；TUN 模式停止后记得 `sudo freecat dns off` 恢复 DNS。

### 用 systemd 常驻（开机自启）

```bash
sudo env FREECAT_HOME=/etc/freecat freecat login -email 你的邮箱 -password 你的密码
sudo env FREECAT_HOME=/etc/freecat freecat update
sudo systemctl enable --now freecat
journalctl -u freecat -f
```

---

## 四、选择节点

默认行为：主策略组 `Freecat 🐾自由猫` 的默认项是 **`自动选择`**（`url-test`，测
`http://www.gstatic.com/generate_204`，每 24 小时重测一次），即“延迟最低的节点”。

```bash
freecat nodes                 # 列出策略组；当前项会展开成实际节点
freecat nodes --all           # 展开所有组的全部节点
freecat use 7                 # 按序号切换（序号见 nodes）
freecat use "🇭🇰 GG香港1-1｜媒体AI"   # 按名称切换
freecat use auto              # 恢复“自动选择”
freecat status                # 显示 组 → 实际节点，以及延迟/流量
```

- 选择会写入 mihomo 的 `cache.db`，下次 `freecat run` 自动恢复。
- 只想启动时指定一次：`freecat run --node 7`（或 `--node auto`）。
- 固定出口 IP 的场景（远程办公、账号风控）请手动选一个节点，不要用自动选择——
  自动选择每 24 小时可能换出口 IP。

---

## 五、命令一览

| 命令 | 作用 |
| --- | --- |
| `freecat login -email E -password P` | 登录并保存 token（也读 `FREECAT_EMAIL/FREECAT_PASSWORD`） |
| `freecat update [--server-host H]` | 拉取订阅、生成运行配置 |
| `freecat run [--tun] [--node N] [--port 7890] [--ctl 9090]` | 启动 mihomo |
| `freecat nodes [--group 组名] [--all]` | 列出策略组与节点 |
| `freecat use 序号\|节点名\|auto` | 手动切换节点 |
| `freecat status` | 运行状态、实际节点、流量、订阅时间 |
| `freecat test [-url URL]` | 通过本地代理测试并显示出口 IP |
| `freecat env` | 输出可 `eval` 的代理环境变量 |
| `freecat dns on\|off\|status` | TUN 配套的系统 DNS 管理 |
| `freecat version` | 版本 |

常用参数：

- `--server-host aqaenca1.tobigyang.cn`：节点入口（默认）。可换
  `bqbencb2.tobigyang.cn`、`cqcencc3.tobigyang.cn`，或回退 IP `34.92.4.52`；
  填 `keep` 保留订阅原域名。
- `--tun`：TUN 全局代理，需要 root 与 `/dev/net/tun`。
- `--node 7` / `--node auto`：启动后自动切节点。
- `--no-update`：`run` 时不自动拉订阅。

配置文件目录：`$FREECAT_HOME`，缺省 `~/.config/freecat`
（`profile.yaml` 原始订阅、`config.yaml` 运行配置、`state.json` 令牌与流量）。

---

## 六、验证记录（2026-11-08，Ubuntu 24.04 / WSL2）

代理模式（域名入口）：

```
google=200 0.57s   github=200 0.79s
test: 177 ms，出口 IP 139.162.17.147
use 7 → 🇭🇰 GG香港1-1｜媒体AI
```

TUN 模式（`sudo freecat dns on && sudo freecat run --tun`）：

```
google=200 1.50s   youtube=200 1.09s   github=200
出口 IP 140.235.141.46
```

---

## 七、故障排查

| 现象 | 处理 |
| --- | --- |
| 登录报 403 / 567 | 换服务器：`freecat login ... --host https://tuc.skyoutdoors.com.cn` |
| 所有节点超时 | 换入口：`freecat update --server-host 34.92.4.52`（或 `bqbencb2.tobigyang.cn`） |
| `can't download MMDB` | 确认 `/opt/freecat/geo/geoip.metadb` 存在，或重新执行 `install.sh` |
| TUN 能连但 DNS 解析异常 | 确认执行过 `sudo freecat dns on`，且 `freecat dns status` 指向 127.0.0.1 |
| 端口被占用 | `--port 7891 --ctl 9091` |
| 想看内核日志 | `freecat run` 前台运行即为日志；systemd 用 `journalctl -u freecat -f` |

安全提示：`state.json` 含登录令牌（非密码），已设为 600；不要把该文件或订阅内容发给他人。

---

## 八、已知限制

- 入口域名按账号生成、可能随官方调整变化；失效时用 `--server-host` 换域名或回退到 IP。
- 仅提供命令行与系统级代理/TUN，不含 GUI、测速图、规则编辑等 Windows 客户端功能。
- 若官方后续给接口加签名/风控，登录与订阅可能需要同步调整（代码集中在 `api.go`，改动面很小）。
