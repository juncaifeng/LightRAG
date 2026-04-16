## ADDED Requirements

### Requirement: YAML-based topic registry

The system SHALL load topic schemas from YAML files in a versioned directory structure `server/topics/v1/`, replacing the current hardcoded approach.

#### Scenario: Topic schema loaded from YAML

- **WHEN** the Go server starts
- **THEN** it scans `server/topics/v1/` directories, reads `schema.yaml` and `metadata.yaml` for each topic, and registers them in the topic registry

#### Scenario: New topic added without code change

- **WHEN** a developer creates a new directory under `server/topics/v1/` with `schema.yaml`, `metadata.yaml`, and `examples/`
- **THEN** the topic is automatically registered on next server start without modifying any Go source code

### Requirement: Topic schema YAML format

Each topic directory SHALL contain `schema.yaml` defining inputs/outputs fields and `metadata.yaml` defining pipeline/stage/description/strategy/weight.

#### Scenario: schema.yaml structure

- **WHEN** `schema.yaml` is read for a topic
- **THEN** it contains `inputs` and `outputs` arrays, each with `name`, `type`, `required`, `description`, `description_en`, and optional `default`

#### Scenario: metadata.yaml structure

- **WHEN** `metadata.yaml` is read for a topic
- **THEN** it contains `name`, `pipeline`, `stage`, `description`, `description_en`, `recommended_strategy`, `recommended_weight`

### Requirement: Code examples as markdown

Each topic directory SHALL contain an `examples/` directory with language-specific code examples in markdown format.

#### Scenario: Examples served via API

- **WHEN** the API returns topic schemas via `/api/topics/schemas`
- **THEN** the `code_templates` field is populated from `examples/*.md` files (keyed by language name, content is the markdown body)

### Requirement: Go embed packaging

The system SHALL use `//go:embed` to package all YAML and markdown files into the Go binary at build time.

#### Scenario: Single binary deployment

- **WHEN** the Go binary is built
- **THEN** it contains all topic definitions and examples embedded, requiring no external files at runtime

### Requirement: Versioned topic directory

Topic definitions SHALL be organized under a versioned directory `topics/v1/` to support future schema format changes.

#### Scenario: Future version coexistence

- **WHEN** a new schema format is introduced in the future
- **THEN** it can be placed under `topics/v2/` without breaking existing `v1/` topics
