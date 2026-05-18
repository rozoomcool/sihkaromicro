"""
Chunking pipeline: section text → parent chunk + semantic child chunks.
"""
from __future__ import annotations

import asyncio
import logging
from dataclasses import dataclass

logger = logging.getLogger(__name__)


@dataclass
class ChunkPair:
    parent_text: str
    parent_position: int
    children: list[tuple[str, int]]  # (text, position_within_parent)


def _semantic_split_sync(sections: list[str]) -> list[ChunkPair]:
    """
    Runs SemanticChunker for each section.
    Executed in a thread executor because LangChain's SemanticChunker
    makes synchronous OpenAI embedding calls internally.
    """
    from langchain_experimental.text_splitter import SemanticChunker
    from langchain_openai import OpenAIEmbeddings

    from config import settings

    embeddings = OpenAIEmbeddings(
        model=settings.embedding_model,
        openai_api_key=settings.openai_api_key,
    )
    splitter = SemanticChunker(
        embeddings=embeddings,
        breakpoint_threshold_type="percentile",
        breakpoint_threshold_amount=90,
    )

    pairs: list[ChunkPair] = []
    for parent_pos, section_text in enumerate(sections):
        docs = splitter.create_documents([section_text])
        children = [
            (doc.page_content.strip(), child_pos)
            for child_pos, doc in enumerate(docs)
            if doc.page_content.strip()
        ]
        if not children:
            children = [(section_text.strip(), 0)]
        pairs.append(
            ChunkPair(
                parent_text=section_text,
                parent_position=parent_pos,
                children=children,
            )
        )
        logger.debug(
            "Section %d → %d child chunks", parent_pos, len(children)
        )

    return pairs


async def build_chunk_pairs(sections: list[str]) -> list[ChunkPair]:
    """Async entry point for the chunking pipeline."""
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _semantic_split_sync, sections)
