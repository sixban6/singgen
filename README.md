# SingGen
SingGen 是一个用于生成 sing-box 配置文件的工具，支持从各种订阅链接和协议解析节点信息。

[![Release](https://img.shields.io/github/v/release/sixban6/singgen)](https://github.com/sixban6/singgen/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/sixban6/singgen)](https://golang.org/)
[![License](https://img.shields.io/github/license/sixban6/singgen)](https://github.com/sixban6/singgen/blob/main/LICENSE)
## 特性
- 🔧 支持多种协议：VMess, VLESS, Trojan, Hysteria2, AnyTLS, TUIC, Shadowsocks
- 📄 支持多种输出格式：JSON, YAML
- 🌐 支持订阅链接和本地文件
- 🎯 多版本 sing-box 配置模板 (v1.12, v1.14+)
- 🔄 模板系统支持热更新和扩展
- 🚀 高性能模块化设计
- 🧪 完整的测试覆盖


## 安装

从 [Releases](https://github.com/sixban6/singgen/releases) 下载对应平台的压缩包（单文件，模板已内嵌）：

| 平台 | 文件 |
|---|---|
| Linux x86_64 / ARM64 | `singgen-linux-amd64.tar.gz` / `singgen-linux-arm64.tar.gz` |
| OpenWrt 等路由器 | `singgen-linux-armv7.tar.gz` / `singgen-linux-mipsle-softfloat.tar.gz` |
| macOS Intel / Apple Silicon | `singgen-darwin-amd64.tar.gz` / `singgen-darwin-arm64.tar.gz` |
| Windows x86_64 | `singgen-windows-amd64.zip` |

可用 `sha256sum -c checksums.txt` 校验下载文件。

### 基本用法

```bash
# 从订阅链接生成配置
./singgen -url https://example.com/subscription -out config.json

# 从订阅链接生成配置，使用特定模板版本。配置本地dns，配置subnet、配置镜像地址，配置文件针对的平台linux
./singgen -url https://example.com/subscription -mirror https://ghfast.top -platform linux -out config.json -template v1.14 -dns 119.119.119.119 -subnet  119.119.119.119/24

# 从本地文件生成配置
./singgen -url subscription.txt -out config.json

# 使用镜像站点下载规则集
./singgen -url subscription.txt -out config.json -mirror https://mirror.example.com

# 启用调试日志
./singgen -url subscription.txt -out config.json -log debug

# 列出所有可用模板版本
./singgen --list-templates

# 多订阅模式：由配置文件驱动
./singgen -config test-config.yaml -out output.json

# 多订阅模式 + 命令行覆盖：只有显式写出的参数会覆盖配置文件
./singgen -config test-config.yaml -platform ios -out ios.json


```
### 进阶用法
可以自己编辑 internal/template/configs/template.yaml 定制自己的模版（单一基准模板，v1.12 由程序自动降级适配）

### 命令行参数
- `-url`: 订阅URL或文件路径（必需）
- `-out`: 输出文件路径（默认: config.json；`-` 表示输出到 stdout，日志始终输出到 stderr）
- `-format`: 输出格式 json/yaml（默认: json）
- `-template`: 模板版本 v1.14/v1.12（默认: v1.14）
- `-mirror`: 规则集下载镜像URL
- `-log`: 日志级别 debug/info/warn/error（默认: warn）
- `--list-templates`: 列出可用的模板版本
- `-dns`: 配置默认的本地dns地址。
- `-subnet`: 配置本地子网地址，用于CDN加速
- `-emoji`: 是否移除节点名中的 emoji（默认: true，使用 `-emoji=false` 保留）
- `-check`: 本机安装了 sing-box 时，先用 `sing-box check` 校验生成的配置，**校验通过才替换输出文件**（默认: true，`-check=false` 关闭）。linux 专用配置在非 linux 机器上会跳过校验
- `-cache-dir`: 订阅缓存目录（默认: 系统用户缓存目录下的 `singgen`）。订阅拉取失败时使用上一次成功的副本并给出警告；设为空字符串 `-cache-dir=""` 关闭

### 节点过滤 / 重命名 / 去重（多订阅配置文件）
规则使用 Go 正则表达式，匹配订阅中原始的节点名（在移除 emoji、添加订阅名前缀之前）。
`global` 中的规则对所有订阅生效；订阅中设置同名字段会覆盖全局规则（设为 `""` 表示对该订阅关闭）。

```yaml
global:
  exclude: "剩余|到期|官网|流量"          # 命中即丢弃
  include: ""                            # 非空时只保留命中的节点
  rename:                                # 按顺序依次替换
    - {match: " Premium$", replace: ""}
    - {match: "^HK-", replace: "香港-"}
  dedupe_by_server: true                 # 跨订阅去重：同一协议+服务器+端口只保留第一个
subscriptions:
  - name: backup
    url: https://example.com/sub
    include: "US|JP"                     # 只覆盖这个订阅的规则
```

### 可靠性
- 订阅拉取遇到网络错误、5xx、429 时自动重试（最多 3 次，指数退避）
- 多订阅并发拉取（最多 4 个同时进行），输出顺序与配置文件一致
- 输出文件原子写入：不会出现写了一半的配置文件
- 机场返回 `subscription-userinfo` 时，剩余流量低于 10% 或 7 天内到期会打印警告

## 支持的协议格式

### VMess
```
vmess://eyJ2IjoiMiIsInBzIjoidGVzdCIsImFkZCI6IjEyNy4wLjAuMSIsInBvcnQiOiI4MCIsImlkIjoiMTIzNDU2NzgiLCJhaWQiOiIwIiwibmV0IjoidGNwIiwiaG9zdCI6IiIsInBhdGgiOiIiLCJ0bHMiOiIifQ==
```

### VLESS
```
vless://uuid@server:port?type=ws&host=example.com&path=/path&security=tls#name
```

### Trojan
```
trojan://password@server:port?type=ws&host=example.com&path=/path#name
```

### Hysteria2
```
hysteria2://password@server:port#name
```

### AnyTLS
```
anytls://password@server:port/?sni=example.com#name
```

### TUIC
```
tuic://uuid:password@server:port/?congestion_control=bbr&alpn=h3&sni=example.com#name
```

### Shadowsocks
```
ss://method:password@server:port#name
```

## 项目结构

```
singgen/
├── cmd/singgen/           # CLI 入口
├── internal/
│   ├── constant/          # 全局常量
│   ├── util/             # 通用工具
│   ├── fetcher/          # 数据获取器
│   ├── parser/           # 协议解析器
│   ├── transformer/      # 节点转换器
│   ├── template/         # 配置模板
│   ├── renderer/         # 输出渲染器
│   └── registry/         # 组件注册中心
├── pkg/model/            # 数据模型
└── test/                 # 测试模块
```

## 开发

### 运行测试

```bash
go test ./test/... -v
```

如果 `PATH` 中有 `sing-box`，测试会对各平台 / 模板版本生成的配置执行 `sing-box check` 端到端校验（CI 中强制执行）。

### 添加新协议支持

1. 在 `internal/parser/` 下创建新的协议解析器
2. 在 `internal/constant/protocol.go` 中定义协议常量
3. 在解析器的 `init()` 函数中注册协议
4. 在 `internal/transformer/` 中添加转换逻辑
5. 编写相应的测试用例

### 添加新模板版本

1. 在 `configs/` 目录下创建 `template-v1.xx.json` 文件
2. 按照 sing-box 配置格式编写模板
3. 使用 `{mirror_url}` 占位符支持镜像URL替换
4. 模板中的 `{all}` 占位符会被替换为实际的代理节点
5. 系统会自动检测并支持新版本模板


## 许可证
[MIT License](LICENSE)
