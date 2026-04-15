import os
import sys
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import asyncio
from lightrag.lightrag import LightRAG, QueryParam
from lightrag.utils import EmbeddingFunc
import numpy as np

async def mock_llm_func(prompt, **kwargs):
    return "Mocked LLM Response"

async def mock_embedding_func(texts, **kwargs):
    return np.zeros((len(texts), 384)).tolist()

async def main():
    print("Testing LightRAG Local Memory Dispatcher Integration...")
    
    # 1. Initialize LightRAG with NO event_bus_url (will use LocalMemoryDispatcher)
    rag = LightRAG(
        working_dir="./rag_storage",
        # Use dummy functions to bypass validation and API calls
        embedding_func=EmbeddingFunc(embedding_dim=384, max_token_size=8192, func=mock_embedding_func),
        llm_model_func=mock_llm_func,
    )
    
    await rag.initialize_storages()
    
    # 2. Insert text. This should trigger the new HookDispatcher for chunking.
    # The NativeChunkingSubscriber will be called in-memory.
    sample_text = "This is a test document. " * 50
    print(f"Inserting document of length {len(sample_text)}...")
    
    # We expect this to run through apipeline_process_enqueue_documents 
    # and hit our new dispatcher.publish_and_wait logic.
    try:
        await rag.ainsert(sample_text)
        print(f"Success! Document inserted via Local Event Bus Dispatcher.")
    except Exception as e:
        print(f"Error during insert: {e}")

if __name__ == "__main__":
    asyncio.run(main())