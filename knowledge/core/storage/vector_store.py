from __future__ import annotations

import json
import logging
from dataclasses import dataclass

import asyncpg
import numpy as np
from pgvector.asyncpg import register_vector

logger = logging.getLogger(__name__)


@dataclass
class ParentChunkRecord:
    source_id: int
    project_id: int
    owner_id: str
    content: str
    position: int
    metadata: dict | None = None


@dataclass
class ChildChunkRecord:
    source_id: int
    project_id: int
    owner_id: str
    content: str
    embedding: list[float]
    position: int
    metadata: dict | None = None


async def create_pool(dsn: str) -> asyncpg.Pool:
    async def _init(conn: asyncpg.Connection) -> None:
        await register_vector(conn)

    return await asyncpg.create_pool(dsn, init=_init)


class VectorStore:
    def __init__(self, pool: asyncpg.Pool) -> None:
        self._pool = pool

    async def save_source_chunks(
        self,
        parents: list[ParentChunkRecord],
        children_by_parent: list[list[ChildChunkRecord]],
    ) -> None:
        """Saves parent chunks and their children in a single transaction."""
        async with self._pool.acquire() as conn:
            async with conn.transaction():
                for parent, children in zip(parents, children_by_parent):
                    parent_id = await conn.fetchval(
                        """
                        INSERT INTO parent_chunks
                            (source_id, project_id, owner_id, content, metadata, position)
                        VALUES ($1, $2, $3, $4, $5, $6)
                        RETURNING id
                        """,
                        parent.source_id,
                        parent.project_id,
                        parent.owner_id,
                        parent.content,
                        json.dumps(parent.metadata) if parent.metadata else None,
                        parent.position,
                    )

                    for child in children:
                        vec = np.array(child.embedding, dtype=np.float32)
                        await conn.execute(
                            """
                            INSERT INTO chunks
                                (parent_chunk_id, source_id, project_id, owner_id,
                                 content, embedding, metadata, position)
                            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
                            """,
                            parent_id,
                            child.source_id,
                            child.project_id,
                            child.owner_id,
                            child.content,
                            vec,
                            json.dumps(child.metadata) if child.metadata else None,
                            child.position,
                        )

    async def delete_by_source(self, source_id: int, owner_id: str) -> int:
        """Deletes all chunks for a source. Returns number of parent chunks deleted."""
        async with self._pool.acquire() as conn:
            result = await conn.execute(
                "DELETE FROM parent_chunks WHERE source_id = $1 AND owner_id = $2",
                source_id,
                owner_id,
            )
            # result is "DELETE N"
            return int(result.split()[-1])

    async def search(
        self,
        query_embedding: list[float],
        owner_id: str,
        project_id: int,
        top_k: int = 20,
    ) -> list[dict]:
        """
        Returns top-k child chunks ranked by cosine similarity,
        strictly filtered by owner_id AND project_id.
        """
        vec = np.array(query_embedding, dtype=np.float32)
        async with self._pool.acquire() as conn:
            rows = await conn.fetch(
                """
                SELECT
                    c.id,
                    c.content,
                    c.source_id,
                    c.parent_chunk_id,
                    c.metadata,
                    p.content AS parent_content,
                    p.metadata AS parent_metadata,
                    1 - (c.embedding <=> $1::vector) AS score
                FROM chunks c
                JOIN parent_chunks p ON c.parent_chunk_id = p.id
                WHERE c.owner_id = $2
                  AND c.project_id = $3
                ORDER BY c.embedding <=> $1::vector
                LIMIT $4
                """,
                vec,
                owner_id,
                project_id,
                top_k,
            )
        return [dict(r) for r in rows]
