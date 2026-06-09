# DNS-Load-Benchmark

`DNS-Load-Benchmark` 是一个用于 DNS 基础设施容量验证的小型命令行工具，适合在自有或已授权的 DNS 环境中做吞吐、延迟和错误率观察。

它支持对单个 resolver 进行测试，也支持使用用户自备 resolver 列表进行轮询分发。项目本身不内置、不维护、不推荐任何公共 resolver 列表。

## 功能特点

- 支持 UDP/TCP 查询。
- 支持 A、AAAA、CNAME、MX、NS、TXT 等常见查询类型。
- 支持固定 QPS、并发 worker、测试时长和单次请求超时配置。
- 支持随机子域名前缀，用于降低缓存命中对基准结果的影响。
- 支持多个 resolver 轮询分发，并输出整体与分 resolver 统计。
- 支持文本摘要和 JSON 输出。

## 使用边界

请只在你拥有或明确获得授权的 DNS 基础设施上运行测试。  
如果使用外部 resolver 或第三方网络资源，请确保你有对应授权，并遵守其服务条款、速率限制和可接受使用政策。

多 resolver 模式主要面向封闭实验环境、企业自有递归节点、合作方授权节点或故障排查场景。公开环境下建议优先使用自有 resolver 或专门的压测链路，不建议把第三方公共 resolver 作为常规测试路径。

## 构建

```bash
go build -o dns-load-benchmark ./cmd/dns-load-benchmark
```

Windows:

```powershell
go build -o dns-load-benchmark.exe ./cmd/dns-load-benchmark
```

## 基本用法

测试本机 resolver：

```bash
./dns-load-benchmark -domain example.com -resolver 127.0.0.1:53 -rate 100 -duration 30s
```

测试指定 resolver：

```bash
./dns-load-benchmark -domain example.com -resolver 192.0.2.53:53 -type A -rate 500 -concurrency 64
```

使用多个 resolver 轮询分发：

```bash
./dns-load-benchmark \
  -domain example.com \
  -resolver 192.0.2.53:53 \
  -resolver 198.51.100.53:53 \
  -rate 1000 \
  -duration 1m
```

从文件读取 resolver：

```bash
./dns-load-benchmark -domain example.com -resolver-file examples/resolvers.example.txt -rate 1000
```

输出 JSON：

```bash
./dns-load-benchmark -domain example.com -resolver 127.0.0.1:53 -json
```

## 参数

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `-domain` | 必填 | 查询的基础域名 |
| `-resolver` | 可重复 | DNS resolver，格式为 `host:port`；未填写时使用 `127.0.0.1:53` |
| `-resolver-file` | 空 | 从文本文件读取 resolver，一行一个，支持 `#` 注释 |
| `-type` | `A` | DNS 查询类型 |
| `-protocol` | `udp` | `udp` 或 `tcp` |
| `-rate` | `100` | 目标 QPS |
| `-concurrency` | `16` | 并发 worker 数 |
| `-duration` | `30s` | 测试时长 |
| `-timeout` | `2s` | 单次查询超时 |
| `-random-prefix` | `true` | 是否在域名前添加随机标签 |
| `-label-depth` | `2` | 随机标签层级 |
| `-json` | `false` | 输出 JSON 汇总 |

## 输出说明

工具会输出：

- 实际 QPS。
- 总请求数、响应数、错误数。
- RCODE 分布。
- 平均延迟、P50、P95、P99。
- 多 resolver 场景下的分 resolver 统计。

## 许可证

Apache License 2.0
