package server

import (
	"reflect"
	"strings"
	"sync"

	"google.golang.org/protobuf/reflect/protoreflect"

	sessionpb "github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go/services/session"
)

// ServiceSchema describes the complete protocol definition for a gRPC service.
type ServiceSchema struct {
	Name          string          `json:"name"`
	Package       string          `json:"package"`
	Description   string          `json:"description"`
	DescriptionEn string          `json:"description_en"`
	Methods       []MethodSchema  `json:"methods"`
	Messages      []MessageSchema `json:"messages"`
}

// MethodSchema describes a single RPC method in a service.
type MethodSchema struct {
	Name          string `json:"name"`
	InputType     string `json:"input_type"`
	OutputType    string `json:"output_type"`
	Description   string `json:"description"`
	DescriptionEn string `json:"description_en"`
}

// MessageSchema describes a proto message used by a service.
type MessageSchema struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Fields      []FieldSchema `json:"fields"`
}

// allServiceMessages maps proto message short names to their Go reflect type.
// These are message types from proto/services/ (Request/Response types).
var allServiceMessages map[string]reflect.Type

// serviceMu protects lazy loading of service schemas.
var serviceMu sync.Once

// builtinServiceSchemas is populated lazily.
var builtinServiceSchemas []ServiceSchema

func init() {
	allServiceMessages = make(map[string]reflect.Type)
	for _, msg := range []any{
		// session service — core types
		(*sessionpb.Session)(nil),
		(*sessionpb.Message)(nil),
		// session service — request/response types
		(*sessionpb.CreateSessionRequest)(nil),
		(*sessionpb.GetSessionRequest)(nil),
		(*sessionpb.UpdateSessionRequest)(nil),
		(*sessionpb.DeleteSessionRequest)(nil),
		(*sessionpb.DeleteResponse)(nil),
		(*sessionpb.ListSessionsRequest)(nil),
		(*sessionpb.ListSessionsResponse)(nil),
		(*sessionpb.AppendMessageRequest)(nil),
		(*sessionpb.AppendMessageResponse)(nil),
		(*sessionpb.ListMessagesRequest)(nil),
		(*sessionpb.ListMessagesResponse)(nil),
	} {
		t := reflect.TypeOf(msg).Elem()
		allServiceMessages[t.Name()] = t
	}
}

// discoverServices scans registered service proto messages, groups them by
// proto file, and extracts service/method definitions from file descriptors.
func discoverServices() []ServiceSchema {
	type msgInfo struct {
		name     string
		goType   reflect.Type
		filePath string
	}

	// Collect all messages with their source file
	var allMsgs []msgInfo
	for name, goType := range allServiceMessages {
		instance := reflect.New(goType).Interface()
		if pr, ok := instance.(interface{ ProtoReflect() protoreflect.Message }); ok {
			desc := pr.ProtoReflect().Descriptor()
			filePath := string(desc.ParentFile().Path())
			allMsgs = append(allMsgs, msgInfo{name: name, goType: goType, filePath: filePath})
		}
	}

	// Group messages by proto file
	fileMessages := make(map[string][]msgInfo)
	for _, m := range allMsgs {
		fileMessages[m.filePath] = append(fileMessages[m.filePath], m)
	}

	var schemas []ServiceSchema

	for _, msgs := range fileMessages {
		// Get the file descriptor from any message in this file
		var fileDesc protoreflect.FileDescriptor
		for _, m := range msgs {
			instance := reflect.New(m.goType).Interface()
			if pr, ok := instance.(interface{ ProtoReflect() protoreflect.Message }); ok {
				fileDesc = pr.ProtoReflect().Descriptor().ParentFile()
				break
			}
		}
		if fileDesc == nil {
			continue
		}

		// Build MessageSchema for each message in this file
		msgSchemas := make([]MessageSchema, 0, len(msgs))
		for _, m := range msgs {
			fields := extractFieldsFromProto(m.goType)
			desc := extractMsgDescription(m.goType)
			msgSchemas = append(msgSchemas, MessageSchema{
				Name:        m.name,
				Description: desc,
				Fields:      fields,
			})
		}

		// Extract service definitions from file descriptor
		services := fileDesc.Services()
		for i := 0; i < services.Len(); i++ {
			svcDesc := services.Get(i)
			svcName := string(svcDesc.Name())

			// Derive short service name: "SessionService" → "session"
			shortName := strings.TrimSuffix(svcName, "Service")
			shortName = camelToSnake(shortName)

			// Extract methods
			methods := svcDesc.Methods()
			methodSchemas := make([]MethodSchema, 0, methods.Len())
			for j := 0; j < methods.Len(); j++ {
				md := methods.Get(j)
				descEn := ""
				if locs := fileDesc.SourceLocations().ByDescriptor(md); locs.LeadingComments != "" {
					descEn = strings.TrimSpace(locs.LeadingComments)
				}
				methodSchemas = append(methodSchemas, MethodSchema{
					Name:          string(md.Name()),
					InputType:     string(md.Input().Name()),
					OutputType:    string(md.Output().Name()),
					Description:   descEn,
					DescriptionEn: descEn,
				})
			}

			// Service description from service comment
			svcDescText := ""
			if locs := fileDesc.SourceLocations().ByDescriptor(svcDesc); locs.LeadingComments != "" {
				svcDescText = strings.TrimSpace(locs.LeadingComments)
			}

			schemas = append(schemas, ServiceSchema{
				Name:          shortName,
				Package:       string(fileDesc.Package()),
				Description:   svcDescText,
				DescriptionEn: svcDescText,
				Methods:       methodSchemas,
				Messages:      msgSchemas,
			})
		}
	}

	return schemas
}

// loadServiceSchemas lazily discovers and caches service schemas.
func loadServiceSchemas() {
	serviceMu.Do(func() {
		builtinServiceSchemas = discoverServices()
	})
}

// GetServiceSchemas returns all registered service schemas.
func GetServiceSchemas() []ServiceSchema {
	loadServiceSchemas()
	return builtinServiceSchemas
}

// GetServiceSchema returns a single service schema by name, or nil if not found.
func GetServiceSchema(name string) *ServiceSchema {
	loadServiceSchemas()
	for i := range builtinServiceSchemas {
		if builtinServiceSchemas[i].Name == name {
			return &builtinServiceSchemas[i]
		}
	}
	return nil
}
