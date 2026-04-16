## ADDED Requirements

### Requirement: Query expansion via EventBus

The system SHALL publish the query expansion step as an EventBus topic `rag.query.query_expansion`, allowing external subscribers to expand keywords using proprietary thesaurus services.

#### Scenario: Default subscriber passes through

- **WHEN** a query is published to `rag.query.query_expansion` with `{"hl_keywords": ["AI"], "ll_keywords": ["LightRAG"]}`
- **THEN** the default subscriber returns the same keywords unchanged: `{"expanded_hl_keywords": ["AI"], "expanded_ll_keywords": ["LightRAG"]}`

#### Scenario: External thesaurus subscriber expands keywords

- **WHEN** an external subscriber registers for `rag.query.query_expansion` and processes `{"hl_keywords": ["AI"], "ll_keywords": ["LightRAG"]}`
- **THEN** the subscriber can append synonyms/near-synonyms/terminology to the output: `{"expanded_hl_keywords": ["AI", "artificial intelligence", "machine learning"], "expanded_ll_keywords": ["LightRAG", "RAG", "retrieval augmented generation"]}`

#### Scenario: Multiple expansion subscribers merge results

- **WHEN** multiple subscribers respond to `rag.query.query_expansion` with APPEND strategy
- **THEN** all expanded keywords are merged into a single deduplicated list

#### Scenario: Expansion config controls behavior

- **WHEN** the publisher includes `expansion_config: {"synonym": true, "near_synonym": false, "terminology": true}`
- **THEN** the subscriber only expands using synonym and terminology sources, skipping near-synonyms

### Requirement: Query expansion topic schema

The system SHALL register a topic schema for `rag.query.query_expansion` with defined inputs, outputs, and code examples loaded from YAML files.

#### Scenario: Schema visible in dashboard

- **WHEN** the user opens the Topics page in the EventBus dashboard
- **THEN** `rag.query.query_expansion` is listed with inputs (hl_keywords, ll_keywords, expansion_config), outputs (expanded_hl_keywords, expanded_ll_keywords, expansion_metadata), and Go/Python/curl code examples
