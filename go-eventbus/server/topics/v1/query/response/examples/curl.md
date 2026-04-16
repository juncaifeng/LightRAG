# Publish a response generation event
curl -X POST http://localhost:50051/api/events \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "rag.query.response",
    "inputs": {
      "query": "What is LightRAG?",
      "context": "LightRAG is a RAG framework...",
      "stream": "false",
      "response_type": "Multiple Paragraphs"
    }
  }'
