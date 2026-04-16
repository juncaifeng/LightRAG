## Context

LightRAG 查询流程当前是硬编码在 `operate.py` 的 `kg_query()` / `naive_query()` 中的单体函数。同时，现有的 topic 注册方式（所有 schema + 模板代码写在 `topic_registry.go` 中）面临文件膨胀和维护困难。

我们需要同时解决两个问题：
1. 查询流程的 topic 化（新增 6 个 query topic）
2. topic 注册架构的现代化（从硬编码迁移到 YAML + embed）

核心约束：
- 无外部 subscriber 时查询行为必须与当前完全一致（向后兼容）
- 查询阶段有严格的串行依赖：关键词提取 → 扩词 → KG/向量检索 → 重排序 → 回答生成
- 新增/修改 topic 不应需要重新编译 Go 代码
- 部署时仍保持单二进制文件的便利性

## Goals / Non-Goals

**Goals:**
- 将 topic 定义从硬编码迁移到 YAML 文件 + `//go:embed` 打包
- 将查询流程拆解为 6 个独立 topic，每个可被外部 subscriber 替换
- 内置 Native Subscriber 保证开箱即用
- 在 EventBus 仪表盘自动展示新 topic 的协议定义和代码示例
- 支持用户在关键词提取后插入扩词服务（如专有词库）
- 版本化目录 `topics/v1/`，为未来 schema 变更做准备

**Non-Goals:**
- 不改变 `/query` 和 `/query/stream` 的 API 接口
- 不改变 protobuf 定义
- 不引入查询阶段的 DAG 编排引擎
- 不支持 topic 的动态注册/注销
- 不实现 `_templates/` SDK 生成器（预留目录，本次不做）

## Decisions

### Decision 1: YAML + embed 而非硬编码

**选择**: 每个 topic 独立目录，包含 `schema.yaml`（字段定义）、`metadata.yaml`（策略/权重/描述）、`examples/*.md`（代码示例）。Go 启动时通过 `//go:embed` 打包加载。

**理由**:
- 新增 topic 不改 Go 代码，只需加目录
- YAML 语言无关，前端/Python/文档工具都能直接读取
- examples 是最终代码片段，可独立维护和测试
- embed 打包进二进制，部署仍为单文件

**替代方案**: 硬编码（当前方式）— 已证明不可持续；纯文件读取（无 embed）— 部署时需管理多个文件

### Decision 2: 串行 pipeline 而非 DAG

**选择**: `kg_query()` 中按固定顺序依次 `publish_and_wait` 各 topic。

**理由**: 查询阶段天然有线性依赖，DAG 编排增加复杂度但无收益。KG 检索和向量检索在 mix 模式下用 `asyncio.gather` 并行。

### Decision 3: 扩词 topic 放在关键词提取之后

**选择**: 在 `keyword_extraction` 和 `kg_search` 之间插入 `query_expansion`。

**理由**: 扩词服务需要已提取的关键词作为输入，且扩词结果直接影响后续检索。

### Decision 4: 重排序 topic 独立

**选择**: 将 rerank 拆为独立 topic，而非嵌入 vector_search。

**理由**: Reranker 模型选择最灵活，独立 topic 允许用户单独替换 reranker。

## Risks / Trade-offs

- [YAML 解析开销] → 启动时一次性加载，embed 打包后无 IO，可忽略
- [运行时 schema 校验] → YAML 格式错误只能启动时发现 → 启动时校验 + CI 检查弥补
- [串行 pipeline 延迟] → 6 次 publish_and_wait 串行增加延迟 → KG 和向量检索并行执行可缓解
- [调试复杂度] → 分散到多个 subscriber 后调试更困难 → correlation_id 可串联

## Migration Plan

1. 创建 `server/topics/v1/` 目录结构 + YAML 文件 + examples
2. 重构 `topic_registry.go` 为 YAML 加载器 + embed
3. 迁移现有 insert topic（chunking, ocr）到新架构
4. Go 编译验证
5. 新增 Python subscriber 适配器
6. 重构 `operate.py` 使用 dispatcher
7. 回归测试

## Open Questions

- naive 模式是否也需要拆 topic？（当前设计：naive 只用 vector_search + response 两个 topic）
- 扩词服务是否需要支持多级串联？（当前设计：单级，多个 subscriber 结果通过 APPEND 合并）
