## Why

LightRAG 当前的查询流程（关键词提取 → 知识图谱检索 → 向量检索 → 重排序 → 回答生成）是硬编码在 `operate.py` 的单体函数中。用户无法在查询管道中插入自定义处理步骤（如专有词库扩词、自定义 reranker、业务关键词过滤等）。

同时，现有的 topic 注册方式（所有 schema + 模板代码硬编码在 `topic_registry.go` 中）会导致文件膨胀、维护困难、新增 topic 需要重新编译。我们需要：

1. 将查询流程拆解为 EventBus topic + subscriber 架构
2. 将 topic 定义从硬编码改为 YAML 文件 + Go embed 打包，实现关注点分离

## What Changes

- 重构 topic 注册架构：从硬编码改为 YAML 文件 + `//go:embed` 打包
- 新增 6 个 query 阶段的 EventBus Topic Schema（YAML 定义 + 代码示例）
- 同时将现有 2 个 insert topic（chunking, ocr）迁移到新架构
- 每个 topic 提供 Go/Python/curl 示例代码（`examples/*.md`）
- 新增 6 个内置 Native Subscriber 适配器（Python 端），封装现有查询逻辑
- 重构 `operate.py` 中的 `kg_query()` 和 `naive_query()`，通过 dispatcher 串行调用各 topic
- 内置 subscriber 保证无外部 subscriber 时查询流程不变
- **BREAKING**: `kg_query()` 函数签名不变，但内部执行路径改为 dispatcher 模式

## Capabilities

### New Capabilities

- `topic-registry-v1`: 基于 YAML + embed 的 topic 注册架构，支持版本化 `topics/v1/`
- `query-keyword-extraction`: 关键词提取 topic
- `query-expansion`: 查询扩词 topic（对接专有词库）
- `query-kg-search`: 知识图谱检索 topic
- `query-vector-search`: 向量检索 topic
- `query-rerank`: 重排序 topic
- `query-response`: 回答生成 topic

### Modified Capabilities

- `insert-chunking`: 从硬编码迁移到 YAML 架构
- `insert-ocr`: 从硬编码迁移到 YAML 架构

## Impact

- **Go 后端**:
  - `server/topic_registry.go` 重构为 YAML 加载器 + embed 打包
  - 新增 `server/topics/v1/` 目录结构（schema.yaml + metadata.yaml + examples/*.md）
- **Python 后端**:
  - `lightrag/hooks/adapters.py` 新增 6 个 subscriber 适配器
  - `lightrag/lightrag.py` 注册默认 subscriber
  - `lightrag/operate.py` 重构 `kg_query()` / `naive_query()` 使用 dispatcher
- **前端**: 无改动（TopicsPage 通过 API 获取 schema，自动展示）
- **API**: 无改动（`/query/stream` 接口不变）
- **依赖**: 新增 `gopkg.in/yaml.v3` 用于解析 YAML
