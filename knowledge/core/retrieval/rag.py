"""
RAG generation via OpenAI chat completions.
Supports both batch (Generate) and streaming (GenerateStream) responses.
"""
from __future__ import annotations

import logging
from collections.abc import AsyncIterator

from openai import AsyncOpenAI

from config import settings
from core.retrieval.search import RetrievalService, RetrievedContext

logger = logging.getLogger(__name__)

_SYSTEM_PROMPT = (
    "You are a helpful assistant that answers questions strictly based on "
    "the provided document excerpts. If the answer cannot be found in the "
    "excerpts, say so clearly. Do not make up information. "
    "Cite relevant excerpts when appropriate."
)


def _build_messages(query: str, contexts: list[RetrievedContext]) -> list[dict]:
    blocks = [f"[Excerpt {i}]\n{ctx.parent_content}" for i, ctx in enumerate(contexts, 1)]
    user_content = "Document excerpts:\n\n" + "\n\n".join(blocks) + f"\n\n---\n\nQuestion: {query}"
    return [
        {"role": "system", "content": _SYSTEM_PROMPT},
        {"role": "user", "content": user_content},
    ]


class RAGService:
    def __init__(self, retrieval: RetrievalService) -> None:
        self._retrieval = retrieval
        self._client = AsyncOpenAI(api_key=settings.openai_api_key)

    async def generate(
        self,
        query: str,
        owner_id: str,
        project_id: int,
    ) -> tuple[str, list[RetrievedContext]]:
        """Returns (answer_text, source_contexts)."""
        contexts = await self._retrieval.retrieve(query, owner_id, project_id)

        if not contexts:
            return "I could not find relevant information in your documents.", []

        response = await self._client.chat.completions.create(
            model=settings.openai_chat_model,
            max_tokens=2048,
            messages=_build_messages(query, contexts),
        )

        answer = response.choices[0].message.content or ""
        return answer, contexts

    async def generate_stream(
        self,
        query: str,
        owner_id: str,
        project_id: int,
    ) -> AsyncIterator[str]:
        """Yields text chunks from OpenAI streaming API."""
        contexts = await self._retrieval.retrieve(query, owner_id, project_id)

        if not contexts:
            yield "I could not find relevant information in your documents."
            return

        stream = await self._client.chat.completions.create(
            model=settings.openai_chat_model,
            max_tokens=2048,
            messages=_build_messages(query, contexts),
            stream=True,
        )

        async for chunk in stream:
            text = chunk.choices[0].delta.content
            if text:
                yield text
