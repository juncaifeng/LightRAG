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
		(*topicspb.EmbeddingInput)(nil),
		(*topicspb.EmbeddingOutput)(nil),
		(*topicspb.OcrInput)(nil),
		(*topicspb.OcrOutput)(nil),
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
	} {
		t := reflect.TypeOf(msg).Elem()
		allProtoMessages[t.Name()] = t
	}
}

// topicStrategyOverrides allows per-topic strategy overrides when the default
// doesn't fit. Most topics default to APPEND (scatter-gather merge semantics).
var topicStrategyOverrides = map[string]string{
	"rag.insert.chunking":           "FIRST",
	"rag.insert.embedding":          "FIRST",
	"rag.insert.ocr":                "FIRST",
	"rag.query.keyword_extraction":  "FIRST",
	"rag.query.query_expansion":     "APPEND",
	"rag.query.vector_search":       "APPEND",
	"rag.query.kg_search":           "APPEND",
	"rag.query.rerank":              "REPLACE",
	"rag.query.response":            "FIRST",
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
