from __future__ import annotations

import json
import logging
from typing import Any, Optional

from .base import LocalSubscriberAdapter, EventEnvelope, SubscriberReply, MergeStrategy

logger = logging.getLogger(__name__)


class NativeChunkingSubscriber(LocalSubscriberAdapter):
    """
    Wraps the native `chunking_by_token_size` function into a standard EventBus subscriber.
    """
    def __init__(self, chunking_func: Any, topic: str = "rag.insert.chunking"):
        super().__init__(
            topic=topic,
            subscriber_id="native-token-chunker",
            strategy=MergeStrategy.APPEND,
            weight=10
        )
        self.chunking_func = chunking_func

    async def process(self, envelope: EventEnvelope) -> SubscriberReply:
        inputs = envelope.inputs

        content = inputs.get("content")
        tokenizer = inputs.get("tokenizer")
        split_by_character = inputs.get("split_by_character")
        split_by_character_only = inputs.get("split_by_character_only", False)
        chunk_overlap_token_size = inputs.get("chunk_overlap_token_size", 100)
        chunk_token_size = inputs.get("chunk_token_size", 1200)

        if not content or not tokenizer:
            return SubscriberReply(
                outputs={},
                strategy=MergeStrategy.IGNORE,
                error_message="Missing content or tokenizer in chunking inputs"
            )

        try:
            chunks = self.chunking_func(
                tokenizer=tokenizer,
                content=content,
                split_by_character=split_by_character,
                split_by_character_only=split_by_character_only,
                chunk_overlap_token_size=chunk_overlap_token_size,
                chunk_token_size=chunk_token_size
            )

            return SubscriberReply(
                outputs={"chunks": chunks},
                strategy=self.strategy,
                weight=self.weight,
                correlation_id=envelope.correlation_id
            )
        except Exception as e:
            logger.error(f"NativeChunkingSubscriber failed: {e}")
            return SubscriberReply(
                outputs={},
                strategy=MergeStrategy.IGNORE,
                error_code="CHUNKING_FAILED",
                error_message=str(e),
                correlation_id=envelope.correlation_id
            )


class NativeKeywordExtractionSubscriber(LocalSubscriberAdapter):
    """
    Wraps `extract_keywords_only()` for the rag.query.keyword_extraction topic.
    """
    def __init__(self, extract_func: Any, global_config: dict, hashing_kv: Any = None):
        super().__init__(
            topic="rag.query.keyword_extraction",
            subscriber_id="native-keyword-extractor",
            strategy=MergeStrategy.REPLACE,
            weight=10,
        )
        self.extract_func = extract_func
        self.global_config = global_config
        self.hashing_kv = hashing_kv

    async def process(self, envelope: EventEnvelope) -> SubscriberReply:
        from lightrag.base import QueryParam

        query = envelope.inputs.get("query", "")
        if isinstance(query, bytes):
            query = query.decode("utf-8")

        try:
            param = QueryParam()
            hl_keywords, ll_keywords = await self.extract_func(
                query, param, self.global_config, self.hashing_kv
            )

            return SubscriberReply(
                outputs={
                    "hl_keywords": hl_keywords,
                    "ll_keywords": ll_keywords,
                },
                strategy=self.strategy,
                weight=self.weight,
                correlation_id=envelope.correlation_id,
            )
        except Exception as e:
            logger.error(f"NativeKeywordExtractionSubscriber failed: {e}")
            return SubscriberReply(
                outputs={"hl_keywords": [], "ll_keywords": []},
                strategy=MergeStrategy.IGNORE,
                error_code="KEYWORD_EXTRACTION_FAILED",
                error_message=str(e),
                correlation_id=envelope.correlation_id,
            )


class NativeQueryExpansionSubscriber(LocalSubscriberAdapter):
    """
    Default pass-through for rag.query.query_expansion.
    Returns keywords unchanged — external subscribers can APPEND expansions.
    """
    def __init__(self):
        super().__init__(
            topic="rag.query.query_expansion",
            subscriber_id="native-query-expander",
            strategy=MergeStrategy.REPLACE,
            weight=5,
        )

    async def process(self, envelope: EventEnvelope) -> SubscriberReply:
        hl_keywords = envelope.inputs.get("hl_keywords", [])
        ll_keywords = envelope.inputs.get("ll_keywords", [])

        if isinstance(hl_keywords, str):
            hl_keywords = json.loads(hl_keywords)
        if isinstance(ll_keywords, str):
            ll_keywords = json.loads(ll_keywords)

        return SubscriberReply(
            outputs={
                "expanded_hl_keywords": hl_keywords,
                "expanded_ll_keywords": ll_keywords,
            },
            strategy=self.strategy,
            weight=self.weight,
            correlation_id=envelope.correlation_id,
        )


class NativeKGSearchSubscriber(LocalSubscriberAdapter):
    """
    Wraps _get_node_data() + _get_edge_data() for the rag.query.kg_search topic.
    """
    def __init__(self, get_node_data: Any, get_edge_data: Any, global_config: dict,
                 entities_vdb: Any, relationships_vdb: Any, knowledge_graph: Any):
        super().__init__(
            topic="rag.query.kg_search",
            subscriber_id="native-kg-searcher",
            strategy=MergeStrategy.REPLACE,
            weight=10,
        )
        self.get_node_data = get_node_data
        self.get_edge_data = get_edge_data
        self.global_config = global_config
        self.entities_vdb = entities_vdb
        self.relationships_vdb = relationships_vdb
        self.knowledge_graph = knowledge_graph

    async def process(self, envelope: EventEnvelope) -> SubscriberReply:
        from lightrag.base import QueryParam

        hl_keywords = envelope.inputs.get("hl_keywords", [])
        ll_keywords = envelope.inputs.get("ll_keywords", [])
        mode = envelope.inputs.get("mode", "mix")
        top_k = int(envelope.inputs.get("top_k", 60))

        if isinstance(hl_keywords, str):
            hl_keywords = json.loads(hl_keywords)
        if isinstance(ll_keywords, str):
            ll_keywords = json.loads(ll_keywords)

        try:
            param = QueryParam(mode=mode, top_k=top_k)

            # For local/mix: search by ll_keywords
            # For global/mix: search by hl_keywords
            # For hybrid: both
            entities = []
            relations = []

            if mode in ("local", "hybrid", "mix") and ll_keywords:
                e, r = await self.get_node_data(
                    ll_keywords, self.knowledge_graph, self.entities_vdb,
                    self.global_config, param, ll_keywords,
                )
                if e:
                    entities.extend(e)
                if r:
                    relations.extend(r)

            if mode in ("global", "hybrid", "mix") and hl_keywords:
                r, e = await self.get_edge_data(
                    hl_keywords, self.knowledge_graph, self.relationships_vdb,
                    self.global_config, param, hl_keywords,
                )
                if e:
                    entities.extend(e)
                if r:
                    relations.extend(r)

            return SubscriberReply(
                outputs={"entities": entities, "relations": relations},
                strategy=self.strategy,
                weight=self.weight,
                correlation_id=envelope.correlation_id,
            )
        except Exception as e:
            logger.error(f"NativeKGSearchSubscriber failed: {e}")
            return SubscriberReply(
                outputs={"entities": [], "relations": []},
                strategy=MergeStrategy.IGNORE,
                error_code="KG_SEARCH_FAILED",
                error_message=str(e),
                correlation_id=envelope.correlation_id,
            )


class NativeVectorSearchSubscriber(LocalSubscriberAdapter):
    """
    Wraps _get_vector_context() for the rag.query.vector_search topic.
    """
    def __init__(self, get_vector_context: Any, chunks_vdb: Any, global_config: dict):
        super().__init__(
            topic="rag.query.vector_search",
            subscriber_id="native-vector-searcher",
            strategy=MergeStrategy.REPLACE,
            weight=10,
        )
        self.get_vector_context = get_vector_context
        self.chunks_vdb = chunks_vdb
        self.global_config = global_config

    async def process(self, envelope: EventEnvelope) -> SubscriberReply:
        from lightrag.base import QueryParam

        query = envelope.inputs.get("query", "")
        if isinstance(query, bytes):
            query = query.decode("utf-8")
        top_k = int(envelope.inputs.get("top_k", 20))

        try:
            param = QueryParam(chunk_top_k=top_k)
            chunks = await self.get_vector_context(
                query, self.chunks_vdb, param,
            )

            return SubscriberReply(
                outputs={"chunks": chunks or []},
                strategy=self.strategy,
                weight=self.weight,
                correlation_id=envelope.correlation_id,
            )
        except Exception as e:
            logger.error(f"NativeVectorSearchSubscriber failed: {e}")
            return SubscriberReply(
                outputs={"chunks": []},
                strategy=MergeStrategy.IGNORE,
                error_code="VECTOR_SEARCH_FAILED",
                error_message=str(e),
                correlation_id=envelope.correlation_id,
            )


class NativeRerankSubscriber(LocalSubscriberAdapter):
    """
    Default pass-through for rag.query.rerank.
    Returns chunks as-is — external subscribers can replace with reranking logic.
    """
    def __init__(self):
        super().__init__(
            topic="rag.query.rerank",
            subscriber_id="native-reranker",
            strategy=MergeStrategy.REPLACE,
            weight=5,
        )

    async def process(self, envelope: EventEnvelope) -> SubscriberReply:
        chunks = envelope.inputs.get("chunks", [])
        if isinstance(chunks, str):
            chunks = json.loads(chunks)

        return SubscriberReply(
            outputs={"ranked_chunks": chunks},
            strategy=self.strategy,
            weight=self.weight,
            correlation_id=envelope.correlation_id,
        )


class NativeResponseSubscriber(LocalSubscriberAdapter):
    """
    Default response generator for rag.query.response.
    Calls the LLM to generate a response from context.
    """
    def __init__(self, llm_func: Any, global_config: dict):
        super().__init__(
            topic="rag.query.response",
            subscriber_id="native-responder",
            strategy=MergeStrategy.REPLACE,
            weight=10,
        )
        self.llm_func = llm_func
        self.global_config = global_config

    async def process(self, envelope: EventEnvelope) -> SubscriberReply:
        from lightrag.prompt import PROMPTS

        query = envelope.inputs.get("query", "")
        context = envelope.inputs.get("context", "")
        if isinstance(query, bytes):
            query = query.decode("utf-8")
        if isinstance(context, bytes):
            context = context.decode("utf-8")

        response_type = envelope.inputs.get("response_type", "Multiple Paragraphs")
        user_prompt = envelope.inputs.get("user_prompt", "")
        if isinstance(response_type, bytes):
            response_type = response_type.decode("utf-8")
        if isinstance(user_prompt, bytes):
            user_prompt = user_prompt.decode("utf-8")

        try:
            system_prompt = PROMPTS["rag_response"].format(
                response_type=response_type,
                user_prompt=user_prompt or "",
                context_data=context,
            )

            response = await self.llm_func(
                query,
                system_prompt=system_prompt,
                stream=False,
            )

            if isinstance(response, bytes):
                response = response.decode("utf-8")

            return SubscriberReply(
                outputs={"response": response},
                strategy=self.strategy,
                weight=self.weight,
                correlation_id=envelope.correlation_id,
            )
        except Exception as e:
            logger.error(f"NativeResponseSubscriber failed: {e}")
            return SubscriberReply(
                outputs={"response": f"Error: {e}"},
                strategy=MergeStrategy.IGNORE,
                error_code="RESPONSE_FAILED",
                error_message=str(e),
                correlation_id=envelope.correlation_id,
            )
