# XIU2/SNIProxy

[![Go Version](https://img.shields.io/github/go-mod/go-version/XIU2/SNIProxy.svg?style=flat-square&label=Go&color=00ADD8&logo=go)](https://github.com/XIU2/SNIProxy/)
[![Release Version](https://img.shields.io/github/v/release/huhengbo/SNIProxy.svg?style=flat-square&label=Release&color=00ADD8&logo=github)](https://github.com/XIU2/SNIProxy/releases/latest)
[![GitHub license](https://img.shields.io/github/license/XIU2/SNIProxy.svg?style=flat-square&label=License&color=00ADD8&logo=github)](https://github.com/XIU2/SNIProxy/)
[![GitHub Star](https://img.shields.io/github/stars/huhengbo/SNIProxy.svg?style=flat-square&label=Star&color=00ADD8&logo=github)](https://github.com/XIU2/SNIProxy/)
[![GitHub Fork](https://img.shields.io/github/forks/huhengbo/SNIProxy.svg?style=flat-square&label=Fork&color=00ADD8&logo=github)](https://github.com/XIU2/SNIProxy/)




> [!NOTE]
> 本项目为开源项目，在[SNIProxy](https://github.com/XIU2/SNIProxy/)的基础上进行二次开发，补充了一些个人自用的功能逻辑

## 增加功能

- 多端口监听、非 TLS/HTTP Host 识别
- 通配符规则（`*.example.com`）、`backend`/`port` 自定义后端
- 首包/空闲超时、最大连接数、优雅退出
- Prometheus 指标（`/metrics`）与健康检查（`/healthz`）
- `SIGHUP` / `systemctl reload` 热加载配置（`listen` 变更需重启）
- systemd 单元、Docker 镜像、GitHub Actions 自动发版
- **不兼容旧版配置**：`listen_addr` / 字符串 `rules` / 扁平 `enable_socks5` 已移除


## \# 使用方法

<details>
<summary><code><strong>「 点击查看 Linux 系统下的使用示例 」</strong></code></summary>

****

以下命令仅为示例，版本号和文件名请前往 [**Releases**](https://github.com/huhengbo/SNIProxy/releases) 查看。

```yaml
# 如果是第一次使用，则建议创建新文件夹（后续更新时，跳过该步骤）
mkdir sniproxy

# 进入文件夹（后续更新，只需要从这里重复下面的下载、解压命令即可）
cd sniproxy

# 下载 sniproxy 压缩包（自行根据需求替换 URL 中 [版本号] 和 [文件名]）
wget -N https://github.com/huhengbo/SNIProxy/releases/download/v1.0.5/sniproxy_linux_amd64.tar.gz
# 如果你是在国内服务器上下载，那么请使用下面这几个镜像加速：
# wget -N https://ghp.ci/https://github.com/huhengbo/SNIProxy/releases/download/v1.0.5/sniproxy_linux_amd64.tar.gz
# wget -N https://ghproxy.cc/https://github.com/huhengbo/SNIProxy/releases/download/v1.0.5/sniproxy_linux_amd64.tar.gz
# wget -N https://ghproxy.net/https://github.com/huhengbo/SNIProxy/releases/download/v1.0.5/sniproxy_linux_amd64.tar.gz
# wget -N https://gh-proxy.com/https://github.com/huhengbo/SNIProxy/releases/download/v1.0.5/sniproxy_linux_amd64.tar.gz

# 如果下载失败的话，尝试删除 -N 参数（如果是为了更新，则记得提前删除旧压缩包 rm sniproxy_linux_amd64.tar.gz ）

# 解压（不需要删除旧文件，会直接覆盖，自行根据需求替换 文件名）
tar -zxf sniproxy_linux_amd64.tar.gz

# 赋予执行权限
chmod +x sniproxy

# 编辑配置文件（根据下面的 配置文件说明 来自定义配置内容并保存(按下 Ctrl+X 然后再按 2 下回车)
nano config.yaml

# 运行（不带参数）
./sniproxy

# 运行（带参数示例）
./sniproxy -c "config.yaml"

# 后台运行（带参数示例）
nohup ./sniproxy -c "config.yaml" > "sni.log" 2>&1 &
```

> 另外，强烈建议顺便提高一下 [系统文件句柄数上限](https://github.com/XIU2/SNIProxy#-提高系统文件句柄数上限-避免报错-too-many-open-files)，避免遇到报错 **too many open files**

> 另外，如果你希望 **开机启动、守护进程(异常退出自动恢复)、后台运行、方便管理** 等，那么可以将其 [注册为系统服务](https://github.com/huhengbo/SNIProxy#-linux-配置为系统服务-systemd---以支持开机启动守护进程等)。

</details>

****

<details>
<summary><code><strong>「 点击查看 Windows 系统下的使用示例 」</strong></code></summary>

****

### 下载

下载已编译好的可执行文件并解压：

1. [Github Releases](https://github.com/huhengbo/SNIProxy/releases)

### 配置

找到配置文件 `config.yaml` 右键菜单 - 打开方式 - 记事本。

根据下面的 [配置文件说明](https://github.com/huhengbo/SNIProxy#-配置文件说明-configyaml) 来自定义配置内容并保存。

### 运行

双击运行 `sniproxy.exe` 文件。

或者在 CMD 命令行中进入软件所在目录并运行 `sniproxy.exe`：

```yaml
# CMD 命令行中进入解压后的 sniproxy 程序所在目录（记得修改下面示例路径）
cd /d C:\xxx\sniproxy

# 运行（不带参数）
sniproxy.exe

# 运行（带参数示例）
sniproxy.exe -c "config.yaml"
```

</details>

---

<details>
<summary><code><strong>「 点击查看 Mac 系统下的使用示例 」</strong></code></summary>

---

下载已编译好的可执行文件并解压：

1. [Github Releases](https://github.com/huhengbo/SNIProxy/releases)

```yaml
# 通过命令行进入 sniproxy 压缩包所在目录（记得修改下面示例路径）
cd /xxx/xxx

# 解压（不需要删除旧文件，会直接覆盖，自行根据需求替换 文件名）
tar -zxf sniproxy_linux_amd64.tar.gz

# 赋予执行权限
chmod a+x sniproxy

# 编辑配置文件（根据下面的 配置文件说明 来自定义配置内容并保存(按下 Contrl+X 然后再按 2 下回车)
nano config.yaml

# 运行（不带参数）
./sniproxy

# 运行（带参数示例）
./sniproxy -c "config.yaml"
```

</details>

---

```css
home@xiu:~# ./sniproxy -h

SNIProxy vX.X.X
https://github.com/huhengbo/SNIProxy

参数：
    -c config.yaml
        配置文件 (默认 config.yaml)
    -l sni.log
        日志文件 (默认 无)
    -d
        调试模式 (默认 关)
    -v
        程序版本
    -h
        帮助说明

信号：
    SIGHUP
        热加载配置（listen 变更需重启）
    SIGINT / SIGTERM
        优雅退出
```

---

## \# 其他说明

#### \# 配置文件说明 (config.yaml)

<details>
<summary><code><strong>「 点击展开 查看内容 」</strong></code></summary>

---

> **注意：** 配置为 YAML。`#` 为注释。旧版字段（`listen_addr`、字符串 `rules`、`enable_socks5`/`socks_addr`）**已不支持**，请按下方格式迁移。

```yaml
# 监听地址（需要引号）
# ":443" / "0.0.0.0:443" / "127.0.0.1:443" / "[::]:443"
listen:
  - ":443"

# 指标与健康检查（Prometheus 文本格式）
# GET /metrics  /healthz  /readyz
metrics_addr: "127.0.0.1:9100"

# 超时与连接上限（"10s"/"5m" 或整数秒）
dial_timeout: 10s
idle_timeout: 5m
header_timeout: 5s
max_conns: 10000

# SOCKS5 前置代理（失败不会静默降级直连）
socks5:
  enable: false
  addr: 127.0.0.1:40000

# 允许所有域名（true 时忽略 rules）
allow_all_hosts: false

# 转发规则：仅对象格式，host 必填
# 匹配：精确或后缀边界（example.com 允许 a.example.com，拒绝 evil-example.com）
rules:
  - host: example.com              # 目标 = SNI/Host + 本地监听端口
  - host: b.example2.com
  - host: "*.cdn.example.com"      # 通配符
    backend: 10.0.0.5
    port: 443
  - host: api.example.com
    backend: 127.0.0.1:8443
```

热加载：`kill -HUP <pid>` 重载 rules / socks5 / 超时等；**`listen` 变更需重启进程**。

---

一些示例：

1. 允许所有域名访问

```yaml
listen:
  - ":443"
allow_all_hosts: true
```

> 开启 `allow_all_hosts` 可能被滥用；建议限制来源 IP 或仅监听 IPv6。

2. 仅允许指定域名

```yaml
listen:
  - ":443"
rules:
  - host: example.com
  - host: b.example2.com
```

3. 通配符 + 自定义后端

```yaml
listen:
  - ":443"
rules:
  - host: "*.cdn.example.com"
    backend: 10.0.0.5
    port: 443
  - host: api.example.com
    backend: 127.0.0.1:8443
```

4. 前置代理 + 指标

```yaml
listen:
  - ":443"
metrics_addr: "127.0.0.1:9100"
socks5:
  enable: true
  addr: 127.0.0.1:40000
rules:
  - host: example.com
```

</details>

---

#### \# Linux 配置为系统服务 (systemd)

<details>
<summary><code><strong>「 点击展开 查看内容 」</strong></code></summary>

---

仓库已提供正式单元文件与安装脚本：

- `deploy/sniproxy.service` — 含 `ExecReload=HUP`、文件句柄上限、`CAP_NET_BIND_SERVICE`
- `deploy/install.sh` — 创建用户、安装二进制与服务

```bash
# 1. 编译（或从 Releases 下载 linux 包）
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=v1.2.0" -o sniproxy .

# 2. 安装（需 root）
sudo bash deploy/install.sh

# 3. 编辑配置后启动
sudo nano /opt/sniproxy/config.yaml
sudo systemctl start sniproxy
sudo systemctl status sniproxy
```

常用命令：

```bash
systemctl stop sniproxy
systemctl restart sniproxy          # 完整重启
systemctl reload sniproxy           # 热加载配置（SIGHUP）
journalctl -u sniproxy -f           # 标准输出日志
tail -f /var/log/sniproxy/sni.log   # 文件日志（若配置了 -l）
systemctl daemon-reload             # 修改 .service 后执行
```

默认路径：

| 项 | 路径 |
|----|------|
| 二进制 | `/opt/sniproxy/sniproxy` |
| 配置 | `/opt/sniproxy/config.yaml` |
| 日志 | `/var/log/sniproxy/sni.log` |
| 单元 | `/etc/systemd/system/sniproxy.service` |

</details>

---

#### \# Docker

<details>
<summary><code><strong>「 点击展开 查看内容 」</strong></code></summary>

---

```bash
# 构建
docker build --build-arg VERSION=v1.2.0 -t sniproxy:latest .

# 运行（映射 80/443，挂载自定义配置）
docker run -d --name sniproxy --restart unless-stopped \
  -p 80:80 -p 443:443 -p 9100:9100 \
  -v "$PWD/config.yaml:/etc/sniproxy/config.yaml:ro" \
  sniproxy:latest
```

指标（需在配置中设置 `metrics_addr: "0.0.0.0:9100"`）：

```bash
curl -s http://127.0.0.1:9100/metrics
curl -s http://127.0.0.1:9100/healthz
```

</details>

---

#### \# 本地多平台打包 / 发版

<details>
<summary><code><strong>「 点击展开 查看内容 」</strong></code></summary>

---

```bash
# 本地打包到 dist/（版本取自 git tag 或 VERSION 环境变量）
./build.sh
VERSION=v1.2.0 ./build.sh
```

GitHub Actions：

- `CI`：push / PR 时 `go vet` + `go test -race` + build
- `Release`：推送 `v*` tag 时自动构建各平台压缩包并上传 Release（含 `checksums.txt`）

```bash
git tag v1.2.0
git push origin v1.2.0
```

</details>

---

#### \# SNIProxy 优先通过 IPv4 还是 IPv6 转发流量给目标域名源服务器？

<details>
<summary><code><strong>「 点击展开 查看内容 」</strong></code></summary>

---

首先需要清楚，SNIProxy 是通过 IPv4 还是 IPv6 地址转发流量给目标域名源服务器，和你是通过 IPv4 还是 IPv6 访问 SNIProxy 服务***无关***，`"你 与 SNIProxy"` 和 `"SNIProxy 与 源服务器"` **这两个环节是独立的，互不影响的**。

因为 SNIProxy 的 DNS 解析环节是交由系统 DNS 服务处理的，因此对于 SNIProxy 是通过 IPv4 还是 IPv6 地址转发流量给目标域名源服务器，则取决于：

1. 运行 SNIProxy 的服务器**是否有 IPv4 或 IPv6 地址**（或都有，也就是双栈服务器）
2. 运行 SNIProxy 的服务器当前系统配置的 **DNS 优先级是 IPv4 优先还是 IPv6 优先**（一般默认都是 IPv6 优先）
3. 该目标域名解析记录中**是否有 IPv4 或 IPv6 地址**（也就是 A 和 AAAA 记录）

即，如果你的服务器有 IPv6 地址，且系统为默认的 IPv6 优先，那么无论你是通过 IPv4 还是 IPv6 访问的 SNIProxy 服务器，只要该域名有 IPv6 解析地址，那么 SNIProxy 就会通过 IPv6 转发流量给目标域名源服务器。

```javascript
访问 example.com <=IPv4=> SNIProxy <=优先 IPv6=> 系统 DNS 解析获得该域名的 IP 地址  <=IPv6=> 源站(example.com)

访问 example.com <=IPv6=> SNIProxy <=优先 IPv6=> 系统 DNS 解析获得该域名的 IP 地址  <=IPv6=> 源站(example.com)
```

假如目标域名解析只有 IPv6 地址，你本地只有 IPv4 地址，但你的服务器有 IPv4+IPv6 地址，那么你就可以通过 IPv4 来访问 SNIProxy，然后 SNIProxy 通过 IPv6 访问目标域名源服务器。

```javascript
访问 example.com(仅 IPv4) <=IPv4=> SNIProxy(支持 IPv4+IPv6)  <=IPv6=> 源站(example.com 仅 IPv6)
```

---

关于这个系统 DNS 服务的 IPv4 IPv6 优先级是可以调的（以下为**将默认的 IPv6 优先改为 IPv4 优先**）：

打开并编辑文件（你也可以使用 vim 来编辑）：

```yaml
nano /etc/gai.conf
```

找到以下行：

```yaml
#precedence ::ffff:0:0/96  100
```

去掉改行行首的 `#` 注释符号，使其变为：

> 如果没有找到的话，可以直接在文件末尾另起一行写上下面这行代码。

```yaml
precedence ::ffff:0:0/96  100
```

按下 `Ctrl+O` 并回车保存文件，然后再按下 `Ctrl+X` 退出当前的 nano 编辑器。

此时随便 `ping` 一个同时拥有 IPv4 及 IPv6 地址的域名，看一下结果是不是 IPv4 地址。

> 另外，修改系统 DNS 优先级后，可能需要清理服务器的 DNS 缓存并重启 SNIProxy 服务。

</details>

---

#### \# 提高系统文件句柄数上限 (避免报错 too many open files)

<details>
<summary><code><strong>「 点击展开 查看内容 」</strong></code></summary>

---

Linux 系统下，一些人可能会遇到报错（日志如下）：

```
接受连接请求时出错: accept tcp [::]:443: accept4: too many open files
```

这是因为系统的文件句柄数耗尽了（默认 1024），提高系统文件句柄数上限可有效缓解该问题（不能完全解决，因为理论上，当打开文件、连接等等足够多时，迟早会耗尽，一般来说不管是做代理还是做网站，这个操作都是必须的）。

- **临时提高**（重启后恢复为 1024）

```shell
ulimit -n 65535
```

- **永久提高**（重启后依然为 65535，当然打开文件后手动删除就恢复了）

```shell
echo "* soft nofile 65535
* hard nofile 65535
root soft nofile 65535
root hard nofile 65535" >> /etc/security/limits.conf
```

执行以上命令后，需要重启 SNIProxy 来使其生效，如果还不行请尝试重启系统。

```yaml
systemctl restart sniproxy
```

</details>

---

## 手动编译

<details>
<summary><code><strong>「 点击展开 查看内容 」</strong></code></summary>

---

为了方便，我是在编译的时候将版本号写入代码中的 version 变量，因此你手动编译时，需要像下面这样在 `go build` 命令后面加上 `-ldflags` 参数来指定版本号：

```bash
go build -ldflags "-s -w -X main.version=v1.0.5"
# 在 SNIProxy 目录中通过命令行（例如 CMD、Bat 脚本）运行该命令，即可编译一个可在和当前设备同样系统、位数、架构的环境下运行的二进制程序（Go 会自动检测你的系统位数、架构）且版本号为 v1.0.5
```

如果想要在 Windows 64位系统下编译**其他系统、架构、位数**，那么需要指定 **GOOS** 和 **GOARCH** 变量。

例如在 Windows 系统下编译一个适用于 **Linux 系统 amd 架构 64 位**的二进制程序：

```bat
SET GOOS=linux
SET GOARCH=amd64
go build -ldflags "-s -w -X main.version=v1.0.5"
```

例如在 Linux 系统下编译一个适用于 **Windows 系统 amd 架构 32 位**的二进制程序：

```bash
GOOS=windows
GOARCH=386
go build -ldflags "-s -w -X main.version=v1.0.5"
```

> 可以运行 `go tool dist list` 来查看当前 Go 版本支持编译哪些组合。

---

当然，为了方便批量编译，我会专门指定一个变量为版本号，后续编译直接调用该版本号变量即可。
同时，批量编译的话，还需要分开放到不同文件夹才行（或者文件名不同），需要加上 `-o` 参数指定。

```bat
:: Windows 系统下是这样：
SET version=v1.0.5
SET GOOS=linux
SET GOARCH=amd64
go build -o Releases\sniproxy_linux_amd64\sniproxy -ldflags "-s -w -X main.version=%version%"
```

```bash
# Linux 系统下是这样：
version=v1.0.5
GOOS=windows
GOARCH=386
go build -o Releases/sniproxy_windows_386/sniproxy.exe -ldflags "-s -w -X main.version=${version}"
```

</details>

---
