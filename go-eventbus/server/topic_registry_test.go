package server

import (
	"io/fs"
	"testing"
)

func TestDumpEmbedFS(t *testing.T) {
	err := fs.WalkDir(topicFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			t.Logf("ERR: %s: %v", path, err)
			return nil
		}
		t.Logf("%s (dir=%v)", path, d.IsDir())
		return nil
	})
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}
}

func TestTopicRegistryLoadsAllTopics(t *testing.T) {
	loadTopicSchemas()
	if loadErr != nil {
		t.Fatalf("load error: %v", loadErr)
	}

	schemas := GetTopicSchemas()
	t.Logf("Loaded %d topic schemas", len(schemas))

	if len(schemas) != 8 {
		t.Errorf("expected 8 topics, got %d", len(schemas))
	}

	expectedTopics := []string{
		"rag.insert.chunking",
		"rag.insert.ocr",
		"rag.query.keyword_extraction",
		"rag.query.query_expansion",
		"rag.query.kg_search",
		"rag.query.vector_search",
		"rag.query.rerank",
		"rag.query.response",
	}

	for _, name := range expectedTopics {
		s := GetTopicSchema(name)
		if s == nil {
			t.Errorf("topic %s not found", name)
			continue
		}
		t.Logf("  %s: pipeline=%s stage=%s inputs=%d outputs=%d examples=%d",
			s.Name, s.Pipeline, s.Stage, len(s.Inputs), len(s.Outputs), len(s.CodeTemplates))

		if len(s.Inputs) == 0 {
			t.Errorf("topic %s has no inputs", name)
		}
		if len(s.Outputs) == 0 {
			t.Errorf("topic %s has no outputs", name)
		}
		if len(s.CodeTemplates) == 0 {
			t.Errorf("topic %s has no code examples", name)
		}
	}
}
