package server

import (
	"reflect"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"

	topicspb "github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go/topics"
)

// FieldSchema describes a single input or output field.
type FieldSchema struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Required      bool   `json:"required"`
	Description   string `json:"description"`     // zh
	DescriptionEn string `json:"description_en"` // en
}

// TopicSchema describes the complete protocol definition for a topic.
type TopicSchema struct {
	Name                string        `json:"name"`
	Pipeline            string        `json:"pipeline"`
	Stage               string        `json:"stage"`
	Description         string        `json:"description"`     // zh
	DescriptionEn       string        `json:"description_en"` // en
	Inputs              []FieldSchema `json:"inputs"`
	Outputs             []FieldSchema `json:"outputs"`
	RecommendedStrategy string        `json:"recommended_strategy"`
	RecommendedWeight   int           `json:"recommended_weight"`
}

// inputOutput pairs Input/Output proto messages that form a topic.
type inputOutput struct {
	InputName  string
	OutputName string
	InputType  reflect.Type
	OutputType reflect.Type
	Domain     string // derived from proto file path parent directory (e.g. "rag", "index")
	Pipeline   string // derived from proto file path filename (e.g. "insert", "builder")
}

// allProtoMessages maps proto message short names to their Go reflect type.
var allProtoMessages map[string]reflect.Type

func init() {
	allProtoMessages = make(map[string]reflect.Type)
	for _, msg := range []any{
		(*topicspb.ChunkingInput)(nil),
		(*topicspb.ChunkingOutput)(nil),
		// embedding domain
		(*topicspb.EmbeddingVector)(nil),
		(*topicspb.EmbeddingInput)(nil),
		(*topicspb.EmbeddingOutput)(nil),
		(*topicspb.OcrInput)(nil),
		(*topicspb.OcrOutput)(nil),
		(*topicspb.LoadTextInput)(nil),
		(*topicspb.LoadTextOutput)(nil),
		(*topicspb.LoadPdfInput)(nil),
		(*topicspb.LoadPdfOutput)(nil),
		(*topicspb.LoadDocxInput)(nil),
		(*topicspb.LoadDocxOutput)(nil),
		(*topicspb.KeywordExtractionInput)(nil),
		(*topicspb.KeywordExtractionOutput)(nil),
		(*topicspb.QueryExpansionInput)(nil),
		(*topicspb.QueryExpansionOutput)(nil),
		(*topicspb.VectorSearchInput)(nil),
		(*topicspb.VectorSearchOutput)(nil),
		(*topicspb.KgSearchInput)(nil),
		(*topicspb.KgSearchOutput)(nil),
		(*topicspb.RerankInput)(nil),
		(*topicspb.RerankOutput)(nil),
		(*topicspb.ResponseInput)(nil),
		(*topicspb.ResponseOutput)(nil),
		// index domain
		(*topicspb.IndexBuildInput)(nil),
		(*topicspb.IndexBuildOutput)(nil),
		(*topicspb.RetrieveInput)(nil),
		(*topicspb.RetrieveOutput)(nil),
		(*topicspb.RetrieveResult)(nil),
		// llm domain
		(*topicspb.CompleteInput)(nil),
		(*topicspb.CompleteOutput)(nil),
		(*topicspb.ChatMessage)(nil),
		// kg domain
		(*topicspb.EntityMergeInput)(nil),
		(*topicspb.EntityMergeOutput)(nil),
		(*topicspb.RelationMergeInput)(nil),
		(*topicspb.RelationMergeOutput)(nil),
		(*topicspb.MergeConfig)(nil),
		(*topicspb.EntityData)(nil),
		(*topicspb.RelationData)(nil),
		(*topicspb.CreatedEntity)(nil),
		// rag.insert common types
		(*topicspb.StorageRef)(nil),
		// mcp domain — shared types
		(*topicspb.ServerConfig)(nil),
		(*topicspb.StdioConfig)(nil),
		(*topicspb.HttpConfig)(nil),
		(*topicspb.ServerStatus)(nil),
		(*topicspb.ToolInfo)(nil),
		(*topicspb.ToolCallRequest)(nil),
		(*topicspb.ToolCallResponse)(nil),
		(*topicspb.ToolContent)(nil),
		// mcp domain — server CRUD
		(*topicspb.CreateServerInput)(nil),
		(*topicspb.CreateServerOutput)(nil),
		(*topicspb.GetServerInput)(nil),
		(*topicspb.GetServerOutput)(nil),
		(*topicspb.ListServersInput)(nil),
		(*topicspb.ListServersOutput)(nil),
		(*topicspb.UpdateServerInput)(nil),
		(*topicspb.UpdateServerOutput)(nil),
		(*topicspb.DeleteServerInput)(nil),
		(*topicspb.DeleteServerOutput)(nil),
		// mcp domain — lifecycle
		(*topicspb.StartServerInput)(nil),
		(*topicspb.StartServerOutput)(nil),
		(*topicspb.StopServerInput)(nil),
		(*topicspb.StopServerOutput)(nil),
		(*topicspb.RestartServerInput)(nil),
		(*topicspb.RestartServerOutput)(nil),
		(*topicspb.GetServerStatusInput)(nil),
		(*topicspb.GetServerStatusOutput)(nil),
		// mcp domain — tools
		(*topicspb.ListToolsInput)(nil),
		(*topicspb.ListToolsOutput)(nil),
		(*topicspb.SearchToolsInput)(nil),
		(*topicspb.SearchToolsOutput)(nil),
		(*topicspb.SearchResult)(nil),
		(*topicspb.ToolIndexDocument)(nil),
		(*topicspb.IndexToolsInput)(nil),
		(*topicspb.IndexToolsOutput)(nil),
		(*topicspb.CallToolInput)(nil),
		(*topicspb.CallToolOutput)(nil),
		// mcp domain — batch
		(*topicspb.BatchStartInput)(nil),
		(*topicspb.BatchStartOutput)(nil),
		(*topicspb.BatchStopInput)(nil),
		(*topicspb.BatchStopOutput)(nil),
		// mcp domain — events
		(*topicspb.ServerEventInput)(nil),
		(*topicspb.ServerEventOutput)(nil),
	} {
		t := reflect.TypeOf(msg).Elem()
		allProtoMessages[t.Name()] = t
	}
}

// topicStrategyOverrides allows per-topic strategy overrides when the default
// doesn't fit. Most topics default to APPEND (scatter-gather merge semantics).
var topicStrategyOverrides = map[string]string{
	"rag.insert.chunking":           "FIRST",
	"embedding.embed.embedding":     "FIRST",
	"rag.insert.ocr":                "FIRST",
	"rag.query.keyword_extraction":  "FIRST",
	"rag.query.query_expansion":     "APPEND",
	"rag.query.vector_search":       "APPEND",
	"rag.query.kg_search":           "APPEND",
	"rag.query.rerank":              "REPLACE",
	"rag.query.response":            "FIRST",
	// llm domain
	"llm.completion.complete": "FIRST",
	// kg domain
	"kg.merge.entity":   "FIRST",
	"kg.merge.relation": "FIRST",
	// mcp domain
	"mcp.server.create_server":      "FIRST",
	"mcp.server.get_server":         "FIRST",
	"mcp.server.list_servers":       "APPEND",
	"mcp.server.update_server":      "FIRST",
	"mcp.server.delete_server":      "FIRST",
	"mcp.server.start_server":       "FIRST",
	"mcp.server.stop_server":        "FIRST",
	"mcp.server.restart_server":     "FIRST",
	"mcp.server.get_server_status":  "APPEND",
	"mcp.server.list_tools":         "APPEND",
	"mcp.server.search_tools":       "APPEND",
	"mcp.server.index_tools":        "FIRST",
	"mcp.server.call_tool":          "FIRST",
	"mcp.server.batch_start":        "FIRST",
	"mcp.server.batch_stop":         "FIRST",
	"mcp.server.server_event":       "APPEND",
}

// camelToSnake converts CamelCase to snake_case.
// "ChunkingInput" → "chunking_input", "KgSearch" → "kg_search"
func camelToSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				prev := rune(s[i-1])
				if prev >= 'a' && prev <= 'z' {
					result.WriteRune('_')
				}
			}
			result.WriteRune(r + ('a' - 'A'))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// discoverTopics scans all registered proto messages and pairs XxxInput/XxxOutput
// into topics. Domain is derived from proto file path's parent directory
// (e.g. "topics/rag/insert.proto" → domain="rag", pipeline="insert").
// Stage is derived from message name (ChunkingInput → chunking).
// Topic name: {domain}.{pipeline}.{stage}
func discoverTopics() []inputOutput {
	type msgInfo struct {
		name     string
		goType   reflect.Type
		filePath string // proto source file path
	}

	// Collect all messages with their source file
	var allMsgs []msgInfo
	for name, goType := range allProtoMessages {
		instance := reflect.New(goType).Interface()
		if pr, ok := instance.(interface{ ProtoReflect() protoreflect.Message }); ok {
			desc := pr.ProtoReflect().Descriptor()
			filePath := string(desc.ParentFile().Path())
			allMsgs = append(allMsgs, msgInfo{name: name, goType: goType, filePath: filePath})
		}
	}

	// Build a map of message name → info
	msgMap := make(map[string]msgInfo, len(allMsgs))
	for _, m := range allMsgs {
		msgMap[m.name] = m
	}

	// Find all Input messages and pair with corresponding Output
	var pairs []inputOutput
	for name, info := range msgMap {
		if !strings.HasSuffix(name, "Input") {
			continue
		}

		base := strings.TrimSuffix(name, "Input")
		outputName := base + "Output"
		outputInfo, ok := msgMap[outputName]
		if !ok {
			continue
		}

		// Derive pipeline from proto file path filename
		// e.g. "topics/rag/insert.proto" → "insert"
		fileName := info.filePath
		if idx := strings.LastIndex(fileName, "/"); idx >= 0 {
			fileName = fileName[idx+1:]
		}
		pipeline := strings.TrimSuffix(fileName, ".proto")

		// Derive domain from proto file path parent directory
		// e.g. "topics/rag/insert.proto" → "rag"
		domain := ""
		pathParts := strings.Split(info.filePath, "/")
		if len(pathParts) >= 2 {
			domain = pathParts[len(pathParts)-2]
		}

		pairs = append(pairs, inputOutput{
			InputName:  name,
			OutputName: outputName,
			InputType:  info.goType,
			OutputType: outputInfo.goType,
			Domain:     domain,
			Pipeline:   pipeline,
		})
	}

	return pairs
}

// protoTypeString returns a human-readable type string for a protobuf field descriptor.
func protoTypeString(fd protoreflect.FieldDescriptor) string {
	if fd.IsList() {
		return "repeated " + protoBaseType(fd)
	}
	if fd.IsMap() {
		keyType := protoBaseType(fd.MapKey())
		valType := protoBaseType(fd.MapValue())
		return "map<" + keyType + ", " + valType + ">"
	}
	return protoBaseType(fd)
}

func protoBaseType(fd protoreflect.FieldDescriptor) string {
	if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
		return string(fd.Message().Name())
	}
	return fd.Kind().String()
}

// extractFieldsFromProto reads fields from a proto message type and returns FieldSchema list.
func extractFieldsFromProto(goType reflect.Type) []FieldSchema {
	instance := reflect.New(goType).Interface()
	pr, ok := instance.(interface{ ProtoReflect() protoreflect.Message })
	if !ok {
		return nil
	}

	desc := pr.ProtoReflect().Descriptor()
	fields := desc.Fields()
	result := make([]FieldSchema, 0, fields.Len())

	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)

		// Extract field comment from proto source locations.
		descEn := ""
		if locs := desc.ParentFile().SourceLocations().ByDescriptor(fd); locs.LeadingComments != "" {
			descEn = strings.TrimSpace(locs.LeadingComments)
		}

		result = append(result, FieldSchema{
			Name:          string(fd.Name()),
			Type:          protoTypeString(fd),
			Required:      fd.HasPresence(),
			Description:   descEn,
			DescriptionEn: descEn,
		})
	}
	return result
}

// extractMsgDescription reads the leading comment of a proto message.
func extractMsgDescription(goType reflect.Type) string {
	instance := reflect.New(goType).Interface()
	pr, ok := instance.(interface{ ProtoReflect() protoreflect.Message })
	if !ok {
		return ""
	}
	desc := pr.ProtoReflect().Descriptor()
	if locs := desc.ParentFile().SourceLocations().ByDescriptor(desc); locs.LeadingComments != "" {
		return strings.TrimSpace(locs.LeadingComments)
	}
	return ""
}

// builtinSchemas is populated lazily.
var builtinSchemas map[string]TopicSchema
var schemasLoaded bool

func loadTopicSchemas() {
	if schemasLoaded {
		return
	}
	schemasLoaded = true

	pairs := discoverTopics()
	builtinSchemas = make(map[string]TopicSchema, len(pairs))

	for _, pair := range pairs {
		// Derive stage from Input message name: "ChunkingInput" → "chunking"
		stageBase := strings.TrimSuffix(pair.InputName, "Input")
		stage := camelToSnake(stageBase)

		topicName := pair.Domain + "." + pair.Pipeline + "." + stage

		inputs := extractFieldsFromProto(pair.InputType)
		outputs := extractFieldsFromProto(pair.OutputType)

		// Description from Input message comment
		desc := extractMsgDescription(pair.InputType)

		strategy := topicStrategyOverrides[topicName]
		if strategy == "" {
			strategy = "APPEND"
		}

		schema := TopicSchema{
			Name:                topicName,
			Pipeline:            pair.Pipeline,
			Stage:               stage,
			Description:         desc,
			DescriptionEn:       desc,
			Inputs:              inputs,
			Outputs:             outputs,
			RecommendedStrategy: strategy,
			RecommendedWeight:   10,
		}
		builtinSchemas[topicName] = schema
	}
}

// GetTopicSchemas returns all registered topic schemas.
func GetTopicSchemas() []TopicSchema {
	loadTopicSchemas()
	result := make([]TopicSchema, 0, len(builtinSchemas))
	for _, schema := range builtinSchemas {
		result = append(result, schema)
	}
	return result
}

// GetTopicSchema returns a single topic schema by name, or nil if not found.
func GetTopicSchema(name string) *TopicSchema {
	loadTopicSchemas()
	s, ok := builtinSchemas[name]
	if !ok {
		return nil
	}
	return &s
}
