# LightRAG 项目 Code Wiki

## 1. 项目整体架构

LightRAG 是一个简单且快速的“检索增强生成（RAG）”框架，它创新性地结合了**知识图谱（Knowledge Graph）**与**向量检索（Vector Retrieval）**。
该项目的架构设计高度模块化，具有良好的可扩展性，能够灵活接入不同的语言模型（LLM）与底层存储数据库。

项目整体可分为四个核心架构层：
1. **核心控制层 (Core Engine)**：负责调控文档的读取、文本分块（Chunking）、实体/关系抽取（Extraction）、图谱合并（Merging）以及多模式查询策略的执行。
2. **存储抽象层 (Storage/KG)**：提供对底层存储引擎的统一抽象接口。包括图存储（Graph Storage）、向量存储（Vector Storage）以及键值/文档状态存储（KV Storage）。
3. **模型驱动层 (LLM Adapter)**：对接各类大语言模型（LLM）和 Embedding 模型，用于文本向量化、实体提取、问题分析及最终的问答生成。
4. **服务与接口层 (API & WebUI)**：基于 FastAPI 构建的服务端，将核心功能封装为 RESTful API，并提供用于交互的 Web 界面。

---

## 2. 主要模块职责

项目的核心代码集中在 `lightrag/` 目录下，各模块及其子目录的职责分工如下：

- **`lightrag/` (核心逻辑)**
  - 项目的总入口，定义了系统核心处理流水线（Pipeline）。处理文档的摄入（Insert）和知识检索与生成（Query）。
- **`lightrag/kg/` (知识图谱与存储模块)**
  - 实现了所有的存储抽象接口。
  - **图数据库实现**：如 `neo4j_impl.py`, `networkx_impl.py`, `memgraph_impl.py` 等。
  - **向量数据库实现**：如 `milvus_impl.py`, `qdrant_impl.py`, `nano_vector_db_impl.py` 等。
  - **KV数据库实现**：如 `redis_impl.py`, `postgres_impl.py`, `mongo_impl.py`, `json_kv_impl.py` 等。
- **`lightrag/llm/` (语言模型适配器)**
  - 提供了与各大主流 LLM 平台和 Embedding 模型的集成。
  - 包含适配器实现：`openai.py`, `gemini.py`, `ollama.py`, `hf.py` (HuggingFace), `vllm.py` 等。
- **`lightrag/api/` (API 服务端)**
  - 基于 FastAPI 构建的后端服务 (`lightrag_server.py`)，包含路由定义（`routers/`）。提供独立的 REST API 服务，供业务系统接入。
- **`lightrag_webui/` (前端界面)**
  - 项目配套的可视化交互界面代码。
- **`k8s-deploy/` (Kubernetes 部署)**
  - 包含云原生环境下的 Shell 脚本（如 `install_lightrag.sh`），用于将项目部署至 K8s 集群。
- **`examples/` (示例代码)**
  - 包含使用 LightRAG 核心 API 的各种应用示例（如 `lightrag_openai_demo.py`）。

---

## 3. 关键类与函数说明

### 核心类：`LightRAG` ([lightrag/lightrag.py](file:///workspace/lightrag/lightrag.py))
这是框架的“大脑”和唯一主入口类。
- **`__init__`**：初始化工作目录、加载配置参数（如 top_k、token 限制等），并根据用户配置动态实例化底层的存储组件（图、向量、KV）和 LLM 模型组件。
- **`insert(string_or_strings)` / `ainsert(...)`**：
  - **职责**：数据摄入。接收单篇或多篇文档，将其处理并持久化至底层存储。
  - **内部逻辑**：经过防重复处理后，调用分块算法，随后调用 LLM 提取实体和关系，最终合并至图数据库和向量数据库中。
- **`query(query_text, param)` / `aquery(...)`**：
  - **职责**：查询与生成。根据用户的问题生成回答。
  - **内部逻辑**：通过 `QueryParam` 指定检索模式（如 `naive`, `local`, `global`, `hybrid`, `mix` 等）。底层会动态调用对应的图检索和向量检索逻辑，组装上下文后交由 LLM 生成答案。

### 核心基类：Storage Base ([lightrag/base.py](file:///workspace/lightrag/base.py))
定义了存储模块必须实现的标准协议。
- **`BaseKVStorage`**：键值存储基类，用于存储原始文本块、文档处理状态等。
- **`BaseVectorStorage`**：向量存储基类，用于存储文本块、实体描述的 Embedding 向量，支持相似度检索（`query` 方法）。
- **`BaseGraphStorage`**：图存储基类，定义了节点操作（`upsert_node`）、边操作（`upsert_edge`）以及子图获取能力。
- **`QueryParam`**：数据类，用于定义查询时的各类参数设定（模式、top_k 等）。

### 算法与操作：([lightrag/operate.py](file:///workspace/lightrag/operate.py))
包含了 RAG 流水线的具体算法实现。
- **`chunking_by_token_size`**：基于 token 大小对长文本进行合理分块。
- **`extract_entities`**：利用 LLM 驱动 Prompt，从文本块中抽取核心实体和关联关系。
- **`merge_nodes_and_edges`**：将新提取的实体关系与现有的图谱进行消歧与合并。
- **`kg_query` / `naive_query`**：不同模式下（局部图、全局图、纯向量）的具体检索和上下文拼装算法。

---

## 4. 依赖关系

项目的依赖通过 `pyproject.toml` 进行管理，采用了分层设计的依赖体系，方便用户按需安装：

- **核心依赖 (Core)**：
  - 异步与网络：`aiohttp`, `tenacity`, `networkx`
  - 数据处理：`numpy`, `pandas`, `pydantic`
  - 默认存储与模型：`nano-vectordb` (轻量级向量库), `google-genai`, `tiktoken`
- **服务端扩展 (`api`)**：
  - `fastapi`, `uvicorn`, `gunicorn`, `python-multipart`, `PyJWT` 等，以及各种文档解析库（`openpyxl`, `pypdf`, `python-docx`）。
- **离线存储扩展 (`offline-storage`)**：
  - 针对各类数据库的客户端驱动：`redis`, `neo4j`, `pymilvus`, `pymongo`, `asyncpg`, `pgvector`, `qdrant-client` 等。
- **离线模型扩展 (`offline-llm`)**：
  - 多厂商模型 SDK：`openai`, `anthropic`, `ollama`, `zhipuai`, `llama-index` 等。

---

## 5. 项目运行方式

LightRAG 支持多种运行和部署模式，满足从本地开发到企业级集群的多种需求。

### 方式一：本地源码运行 (适合开发与测试)
项目推荐使用 `uv` 或 `pip` 进行包管理和虚拟环境创建。
```bash
# 克隆仓库后进入目录
cd LightRAG

# 使用 uv 安装核心及离线依赖并创建虚拟环境
uv sync --extra test --extra offline

# 激活虚拟环境
source .venv/bin/activate  # Linux/macOS
# .venv\Scripts\activate   # Windows

# 运行基础 Demo
export OPENAI_API_KEY="sk-..."
python examples/lightrag_openai_demo.py
```

### 方式二：API 服务端运行 (Docker & Docker Compose)
如果希望将项目作为后端 API 提供服务，可以使用提供的 Docker 配置。
```bash
# 1. 准备环境变量配置
cp env.example .env
# 编辑 .env 文件配置相关 API Key 和数据库连接信息

# 2. 启动基础服务（Redis, Neo4j, Milvus 等环境依赖）
make env-base

# 3. 构建并启动完整的 LightRAG Server 和 WebUI
docker compose -f docker-compose-full.yml up -d
```
启动后，API 服务通常可通过 `http://localhost:8020/docs` 访问，WebUI 前端可通过 `http://localhost:8000` 访问。

### 方式三：Kubernetes (K8s) 云原生部署
对于生产级别的大规模部署，项目在 `k8s-deploy/` 目录下提供了完整的部署脚本。
```bash
cd k8s-deploy

# 运行安装脚本将基础数据库服务和 LightRAG 部署至 K8s 集群
./install_lightrag.sh
```

### 注意事项
- 更换 Embedding 模型时，必须清空工作目录（默认如 `./dickens`）中的历史数据，否则会导致维度不匹配错误。
- 若需要使用特定存储后端的完整功能，请在安装时指定对应 extras 或手动安装相应依赖（例如 `pip install "lightrag-hku[offline]"`）。