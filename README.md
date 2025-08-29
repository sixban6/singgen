# SingGen
SingGen 是一个用于生成 sing-box 配置文件的工具，支持从各种订阅链接和协议解析节点信息。

## 特性
- 🔧 支持多种协议：VMess, VLESS, Trojan, Hysteria2, Shadowsocks
- 📄 支持多种输出格式：JSON, YAML
- 🌐 支持订阅链接和本地文件
- 🎯 多版本 sing-box 配置模板 (v1.11, v1.12+)
- 🔄 模板系统支持热更新和扩展
- 🚀 高性能模块化设计
- 🧪 完整的测试覆盖

## 安装

### 基本用法

```bash
# 从订阅链接生成配置
./singgen -url https://example.com/subscription -out config.json

# 从订阅链接生成配置，使用特定模板版本。配置本地dns，配置subnet、配置镜像地址，配置文件针对的平台linux
./singgen -url https://example.com/subscription -mirror https://ghfast.top -platform linux -out config.json -template v1.13 -dns 119.119.119.119 -subnet  119.119.119.119/24

# 从本地文件生成配置
./singgen -url subscription.txt -out config.json

# 使用镜像站点下载规则集
./singgen -url subscription.txt -out config.json -mirror https://mirror.example.com

# 启用调试日志
./singgen -url subscription.txt -out config.json -log debug

# 列出所有可用模板版本
./singgen --list-templates

# 多订阅模式：完全由配置文件驱动
./singgen -config my-config.yaml -out output.json


```

### 命令行参数
- `-url`: 订阅URL或文件路径（必需）
- `-out`: 输出文件路径（默认: config.json）
- `-format`: 输出格式 json/yaml（默认: json）
- `-template`: 模板版本 v1.12/v1.13等（默认: v1.12）
- `-mirror`: 规则集下载镜像URL
- `-log`: 日志级别 debug/info/warn/error（默认: warn）
- `--list-templates`: 列出可用的模板版本
- `-dns`: 配置默认的本地dns地址。
- `-subnet`: 配置本地子网地址，用于CDN加速

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