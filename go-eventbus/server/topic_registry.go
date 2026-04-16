package server

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed topics/v1/**/*
var topicFiles embed.FS

// FieldSchema describes a single input or output field.
type FieldSchema struct {
	Name         string `json:"name" yaml:"name"`
	Type         string `json:"type" yaml:"type"`
	Required     bool   `json:"required" yaml:"required"`
	Description  string `json:"description" yaml:"description"`     // zh
	DescriptionEn string `json:"description_en" yaml:"description_en"` // en
	Default      string `json:"default,omitempty" yaml:"default,omitempty"`
}

// TopicSchema describes the complete protocol definition for a topic.
type TopicSchema struct {
	Name               string            `json:"name"`
	Pipeline           string            `json:"pipeline"`
	Stage              string            `json:"stage"`
	Description        string            `json:"description"`     // zh
	DescriptionEn      string            `json:"description_en"`  // en
	Inputs             []FieldSchema     `json:"inputs"`
	Outputs            []FieldSchema     `json:"outputs"`
	RecommendedStrategy string          `json:"recommended_strategy"`
	RecommendedWeight  int               `json:"recommended_weight"`
	CodeTemplates      map[string]string `json:"code_templates"`
}

// schemaYAML maps to schema.yaml
type schemaYAML struct {
	Inputs  []FieldSchema `yaml:"inputs"`
	Outputs []FieldSchema `yaml:"outputs"`
}

// metadataYAML maps to metadata.yaml
type metadataYAML struct {
	Name                string `yaml:"name"`
	Pipeline            string `yaml:"pipeline"`
	Stage               string `yaml:"stage"`
	Description         string `yaml:"description"`
	DescriptionEn       string `yaml:"description_en"`
	RecommendedStrategy string `yaml:"recommended_strategy"`
	RecommendedWeight   int    `yaml:"recommended_weight"`
}

// builtinSchemas is loaded from embedded YAML files lazily on first access.
var builtinSchemas map[string]TopicSchema
var loadErr error
var schemasLoaded bool

func loadTopicSchemas() {
	if schemasLoaded {
		return
	}
	schemasLoaded = true
	builtinSchemas = make(map[string]TopicSchema)

	loadErr = fs.WalkDir(topicFiles, ".", func(fpath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Only process metadata.yaml to discover topic directories
		if path.Base(fpath) != "metadata.yaml" {
			return nil
		}

		topicDir := path.Dir(fpath)

		// Read metadata.yaml
		metaBytes, err := topicFiles.ReadFile(fpath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", fpath, err)
		}
		var meta metadataYAML
		if err := yaml.Unmarshal(metaBytes, &meta); err != nil {
			return fmt.Errorf("failed to parse %s: %w", fpath, err)
		}

		// Read schema.yaml
		schemaPath := path.Join(topicDir, "schema.yaml")
		schemaBytes, err := topicFiles.ReadFile(schemaPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", schemaPath, err)
		}
		var schema schemaYAML
		if err := yaml.Unmarshal(schemaBytes, &schema); err != nil {
			return fmt.Errorf("failed to parse %s: %w", schemaPath, err)
		}

		// Read examples/*.md as code templates
		codeTemplates := make(map[string]string)
		examplesDir := path.Join(topicDir, "examples")
		fs.WalkDir(topicFiles, examplesDir, func(epath string, eentry fs.DirEntry, eerr error) error {
			if eerr != nil || eentry.IsDir() {
				return nil
			}
			ename := eentry.Name()
			if !strings.HasSuffix(ename, ".md") {
				return nil
			}
			lang := strings.TrimSuffix(ename, ".md")
			content, err := topicFiles.ReadFile(epath)
			if err == nil {
				codeTemplates[lang] = string(content)
			}
			return nil
		})

		// Assemble TopicSchema
		topic := TopicSchema{
			Name:               meta.Name,
			Pipeline:           meta.Pipeline,
			Stage:              meta.Stage,
			Description:        meta.Description,
			DescriptionEn:      meta.DescriptionEn,
			Inputs:             schema.Inputs,
			Outputs:            schema.Outputs,
			RecommendedStrategy: meta.RecommendedStrategy,
			RecommendedWeight:  meta.RecommendedWeight,
			CodeTemplates:      codeTemplates,
		}

		builtinSchemas[meta.Name] = topic
		return nil
	})
}

// GetTopicSchemas returns all registered topic schemas.
func GetTopicSchemas() []TopicSchema {
	loadTopicSchemas()
	if loadErr != nil {
		return nil
	}
	result := make([]TopicSchema, 0, len(builtinSchemas))
	for _, schema := range builtinSchemas {
		result = append(result, schema)
	}
	return result
}

// GetTopicSchema returns a single topic schema by name, or nil if not found.
func GetTopicSchema(name string) *TopicSchema {
	loadTopicSchemas()
	if loadErr != nil {
		return nil
	}
	s, ok := builtinSchemas[name]
	if !ok {
		return nil
	}
	return &s
}
