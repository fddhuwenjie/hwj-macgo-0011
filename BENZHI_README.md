# 本质评测环境说明

## 项目

- 项目编号：`hwj-macgo-0011`
- 项目名称：实验运行溯源引擎
- 项目说明：管理实验定义、参数版本、输入快照、执行尝试和输出制品的本地溯源服务。

## 固定环境

- Go toolchain：`go1.26.5`
- go.mod language version：`go 1.21`
- GOTOOLCHAIN：`local`
- 支持平台：`linux/amd64`、`linux/arm64`
- Docker 基础镜像：`golang:1.26.5-bookworm`
- Docker manifest：`golang@sha256:53eeac89074db483fdf0ab3be1df32bf6e47562263d2d0d6baa7f26acb4957dd`

## 构建

```bash
./build_benzhi_docker.sh hwj-macgo-0011:benzhi-amd64 linux/amd64
./build_benzhi_docker.sh hwj-macgo-0011:benzhi-arm64 linux/arm64
```

## 运行

```bash
docker run --rm -it --network none hwj-macgo-0011:benzhi-amd64 bash
```

## 容器内验证

```bash
go version
go env GOTOOLCHAIN GOPROXY GOMODCACHE GOCACHE
go test ./...
go vet ./...
go build ./...
```

---

# 项目 README 同步内容

# 实验运行溯源引擎

本系统是一个本地优先的实验运行溯源引擎，用于管理实验定义、参数版本、输入快照、运行计划、执行尝试、输出制品及其谱系。

## 核心特性

- 完整的多步状态机：草稿、冻结、排队、领取、执行、重试等待、成功封存、发布。
- 跨实体原子事务与乐观版本控制。
- 本地文件系统持久化，支持写前日志、原子快照、校验与恢复。
- 后台执行器支持取消传播、超时、指数退避与重启续跑。
- 组合过滤、分页与稳定排序查询。
- 标准库 HTTP 服务托管可操作前端页面。
- 无外部依赖，仅使用 Go 标准库。

## 快速开始

```bash
go run ./cmd/server
```

服务默认监听 `PORT` 环境变量指定的端口，若未设置则使用 `8080`。

健康检查：`/healthz`

自检模式：

```bash
go run ./cmd/server --self-check
```

## 构建与测试

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./...
```

## 目录结构

- `cmd/server`：HTTP 服务入口
- `internal/domain`：领域对象与状态机定义
- `internal/application`：应用服务层
- `internal/repository`：仓储接口与文件实现
- `internal/journal`：写前日志
- `internal/recovery`：恢复逻辑
- `internal/scheduler`：后台执行器
- `internal/query`：查询构建与排序
- `internal/audit`：审计日志
- `internal/persistence`：持久化工具
- `internal/web`：HTTP 处理器与前端
- `internal/util`：通用工具

## 文档

详细设计请参见 `docs` 目录（如有）。
