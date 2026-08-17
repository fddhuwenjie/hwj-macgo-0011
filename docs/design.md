# 实验运行溯源引擎设计文档

## 概述

本系统是一个本地优先的实验运行溯源引擎，用于管理实验定义、参数版本、输入快照、运行计划、执行尝试、输出制品及其谱系。系统采用 Go 语言编写，仅依赖标准库，支持文件系统持久化、WAL 日志、恢复机制和后台调度。

## 架构

系统分为三个主要职责层：

1. **领域层（internal/domain）**：定义核心实体、状态机和业务规则。
2. **应用层（internal/application）**：协调领域对象，提供用例服务，如创建实验、冻结、排队、领取、完成和发布。
3. **基础设施层（internal/repository、internal/persistence、internal/journal、internal/recovery）**：负责持久化、日志、恢复等。

此外，internal/query 提供查询服务，internal/web 提供 HTTP API 和前端页面，internal/scheduler 负责后台任务调度。

## 核心实体

- **Experiment**：实验聚合根，包含当前参数版本和输入快照引用。
- **ParamVersion**：参数定义版本，不可变。
- **InputSnapshot**：输入数据快照，不可变。
- **RunPlan**：运行计划，描述一次实验执行。
- **ExecutionAttempt**：执行尝试，记录单次运行。
- **WorkerLease**：工作者租约，用于并发控制。
- **OutputArtifact**：输出制品，带哈希校验。
- **LineageEdge**：谱系边，记录实体间关系。
- **ReleaseTag**：发布标签。
- **ResourceBudget**：资源预算限制。

## 状态机

- 实验状态：`draft -> frozen -> queued -> running -> sealed -> published`，失败时进入 `failed`。
- 运行计划状态：`draft -> frozen -> queued -> claimed -> executing -> sealed/failed/retry_wait`，`retry_wait` 可重新排队。

## 持久化

文件存储采用每个实体类型一个子目录，每个实体一个 JSON 文件，使用自定义记录格式（魔数、版本、长度、校验和）保证完整性。写操作通过临时文件重命名实现原子性。

## 恢复

系统支持 WAL（write-ahead log）记录操作，恢复时重放日志。快照功能可创建数据目录的完整副本。

## 并发控制

使用乐观锁（版本号）实现并发更新检测。租约机制用于工作节点领取任务。

## 扩展性

通过接口抽象仓储，便于替换为其他存储后端。查询服务支持过滤、排序和分页。
