## 1. Topic 注册架构重构 — YAML + embed

- [x] 1.1 创建 `server/topics/v1/insert/chunking/` 目录，包含 `schema.yaml`、`metadata.yaml`、`examples/go.md`、`examples/python.md`、`examples/curl.md`
- [x] 1.2 创建 `server/topics/v1/insert/ocr/` 目录，包含同上文件
- [x] 1.3 创建 `server/topics/v1/query/keyword_extraction/` 目录，包含同上文件
- [x] 1.4 创建 `server/topics/v1/query/expansion/` 目录，包含同上文件
- [x] 1.5 创建 `server/topics/v1/query/kg_search/` 目录，包含同上文件
- [x] 1.6 创建 `server/topics/v1/query/vector_search/` 目录，包含同上文件
- [x] 1.7 创建 `server/topics/v1/query/rerank/` 目录，包含同上文件
- [x] 1.8 创建 `server/topics/v1/query/response/` 目录，包含同上文件

## 2. Go 后端 — YAML 加载器 + embed

- [x] 2.1 重构 `server/topic_registry.go`：移除硬编码的 `builtinSchemas` 和模板代码变量
- [x] 2.2 新增 `//go:embed` 声明，打包 `topics/v1/**/*` 目录
- [x] 2.3 新增 YAML 解析逻辑：扫描目录、读取 `schema.yaml` + `metadata.yaml`、读取 `examples/*.md` → 组装 `TopicSchema`
- [x] 2.4 新增 `gopkg.in/yaml.v3` 依赖
- [x] 2.5 Go 编译验证

## 3. Python 后端 — Subscriber 适配器

- [x] 3.1 在 `hooks/adapters.py` 新增 `NativeKeywordExtractionSubscriber`，封装 `extract_keywords_only()`
- [x] 3.2 新增 `NativeQueryExpansionSubscriber`，默认透传（expanded_hl = hl, expanded_ll = ll）
- [x] 3.3 新增 `NativeKGSearchSubscriber`，封装 `_get_node_data()` + `_get_edge_data()`
- [x] 3.4 新增 `NativeVectorSearchSubscriber`，封装 `_get_vector_context()`
- [x] 3.5 新增 `NativeRerankSubscriber`，封装 rerank 逻辑
- [x] 3.6 新增 `NativeResponseSubscriber`，封装 LLM 回答生成

## 4. Python 后端 — 注册与编排

- [x] 4.1 在 `lightrag.py` 的 `__post_init__` 中注册所有 6 个默认 subscriber
- [x] 4.2 重构 `operate.py` 中 `kg_query()`，改为通过 dispatcher 串行调用各 topic
- [x] 4.3 重构 `operate.py` 中 `naive_query()`，通过 dispatcher 调用 vector_search + response
- [x] 4.4 在 `operate.py` 中，mix 模式下 kg_search 和 vector_search 用 `asyncio.gather` 并行执行

## 5. 验证

- [x] 5.1 Go 编译通过，`/api/topics/schemas` 返回 8 个 topic（2 insert + 6 query）
- [ ] 5.2 无外部 subscriber 时，查询结果与重构前一致（回归测试）
- [ ] 5.3 EventBus 仪表盘 Topics 页面展示所有 topic 的 schema + 代码示例
- [ ] 5.4 注册外部 dummy subscriber 到任一 topic，验证结果被正确合并
