"""
Two-stage retrieval:
  1. ANN search in pgvector  → top-N child chunks  (default N=20)
  2. CrossEncoder rerank     → top-K parent chunks  (default K=5)

owner_id is ALWAYS injected by the caller from the verified JWT —
it is never taken from the incoming gRPC request body.
"""
from __future__ import annotations

import logging
from collections import defaultdict
from dataclasses import dataclass

from core.embeddings.base import EmbeddingProvider
from core.reranker.base import Reranker
from core.storage.vector_store import VectorStore

logger = logging.getLogger(__name__)


@dataclass
class RetrievedContext:
    parent_chunk_id: int
    parent_content: str
    source_id: int
    best_child_score: float
    rerank_score: float
    metadata: dict | None


class RetrievalService:
    def __init__(
        self,
        vector_store: VectorStore,
        embedder: EmbeddingProvider,
        reranker: Reranker,
        ann_top_k: int = 20,
        rerank_top_k: int = 5,
    ) -> None:
        self._store = vector_store
        self._embedder = embedder
        self._reranker = reranker
        self._ann_top_k = ann_top_k
        self._rerank_top_k = rerank_top_k

    async def retrieve(
        self,
        query: str,
        owner_id: str,
        project_id: int,
        top_k: int | None = None,
    ) -> list[RetrievedContext]:
        rerank_k = top_k or self._rerank_top_k

        # 1. Embed query
        query_vec = await self._embedder.embed_query(query)

        # 2. ANN search — filtered by owner_id + project_id
        rows = await self._store.search(
            query_embedding=query_vec,
            owner_id=owner_id,
            project_id=project_id,
            top_k=self._ann_top_k,
        )

        if not rows:
            return []

        # 3. Deduplicate by parent_chunk_id, keep best child score
        parent_map: dict[int, dict] = {}
        for row in rows:
            pid = row["parent_chunk_id"]
            if pid not in parent_map or row["score"] > parent_map[pid]["score"]:
                parent_map[pid] = row

        unique_parents = list(parent_map.values())
        parent_texts = [p["parent_content"] for p in unique_parents]

        # 4. Rerank parents with CrossEncoder
        reranked = await self._reranker.rerank(query, parent_texts, top_k=rerank_k)

        results: list[RetrievedContext] = []
        for orig_idx, rerank_score in reranked:
            p = unique_parents[orig_idx]
            import json
            meta = None
            if p.get("parent_metadata"):
                try:
                    meta = json.loads(p["parent_metadata"]) if isinstance(p["parent_metadata"], str) else p["parent_metadata"]
                except Exception:
                    pass
            results.append(
                RetrievedContext(
                    parent_chunk_id=p["parent_chunk_id"],
                    parent_content=p["parent_content"],
                    source_id=p["source_id"],
                    best_child_score=float(p["score"]),
                    rerank_score=rerank_score,
                    metadata=meta,
                )
            )

        logger.info(
            "Retrieved %d context chunks for project %s/%d",
            len(results),
            owner_id,
            project_id,
        )
        return results
