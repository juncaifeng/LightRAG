## ADDED Requirements

### Requirement: Knowledge graph search via EventBus

The system SHALL publish the KG search step as an EventBus topic `rag.query.kg_search`, allowing external subscribers to replace the default entity/relation retrieval.

#### Scenario: Default local mode subscriber searches entities

- **WHEN** a query is published to `rag.query.kg_search` with `{"ll_keywords": ["LightRAG"], "mode": "local"}`
- **THEN** the default subscriber calls `_get_node_data()` to search entities by vector similarity and returns `{"entities": [...], "relations": [...]}`

#### Scenario: Default global mode subscriber searches relations

- **WHEN** a query is published to `rag.query.kg_search` with `{"hl_keywords": ["RAG framework"], "mode": "global"}`
- **THEN** the default subscriber calls `_get_edge_data()` to search relations by vector similarity and returns `{"entities": [...], "relations": [...]}`

#### Scenario: Default hybrid/mix mode subscriber searches both

- **WHEN** a query is published to `rag.query.kg_search` with `{"hl_keywords": [...], "ll_keywords": [...], "mode": "hybrid"}`
- **THEN** the default subscriber calls both `_get_node_data()` and `_get_edge_data()` and merges results

#### Scenario: External graph database subscriber

- **WHEN** an external subscriber registers for `rag.query.kg_search`
- **THEN** the subscriber can query an external graph database (e.g., Neo4j, FalkorDB) and return entities/relations in the same format

### Requirement: KG search topic schema

The system SHALL register a topic schema for `rag.query.kg_search` with defined inputs, outputs, and code examples loaded from YAML files.

#### Scenario: Schema visible in dashboard

- **WHEN** the user opens the Topics page in the EventBus dashboard
- **THEN** `rag.query.kg_search` is listed with inputs (hl_keywords, ll_keywords, mode, top_k), outputs (entities, relations), and Go/Python/curl code examples
