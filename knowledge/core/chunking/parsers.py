"""
Document-aware parsing via unstructured.
Returns a list of text sections (strings) that map to parent chunks.

Supported file_type values (match SourceType in sources.proto):
  pdf, docx, txt, md / markdown, url
"""
from __future__ import annotations

import asyncio
import io
import logging

logger = logging.getLogger(__name__)

_CONTENT_TYPE_MAP = {
    "pdf":      "application/pdf",
    "docx":     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    "txt":      "text/plain",
    "md":       "text/markdown",
    "markdown": "text/markdown",
}


def _parse_file_sync(file_obj: io.IOBase, file_type: str) -> list[str]:
    from unstructured.chunking.title import chunk_by_title
    from unstructured.partition.auto import partition

    from config import settings

    content_type = _CONTENT_TYPE_MAP.get(file_type.lower(), "application/octet-stream")

    elements = partition(
        file=file_obj,
        content_type=content_type,
        include_page_breaks=True,
        strategy="hi_res",
    )

    chunks = chunk_by_title(
        elements,
        max_characters=settings.parent_chunk_max_chars,
        new_after_n_chars=settings.parent_chunk_max_chars - 500,
        combine_text_under_n_chars=200,
    )

    sections = [str(c).strip() for c in chunks if str(c).strip()]
    logger.info("Parsed %d sections from %s document", len(sections), file_type)
    return sections


async def _parse_url(url: str) -> list[str]:
    """Fetch a web page and extract text via unstructured."""
    import httpx

    async with httpx.AsyncClient(timeout=30, follow_redirects=True) as client:
        resp = await client.get(url)
        resp.raise_for_status()
        html_bytes = resp.content

    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _parse_html_sync, html_bytes)


def _parse_html_sync(html_bytes: bytes) -> list[str]:
    from unstructured.chunking.title import chunk_by_title
    from unstructured.partition.html import partition_html

    from config import settings

    elements = partition_html(file=io.BytesIO(html_bytes))
    chunks = chunk_by_title(
        elements,
        max_characters=settings.parent_chunk_max_chars,
        new_after_n_chars=settings.parent_chunk_max_chars - 500,
        combine_text_under_n_chars=200,
    )
    sections = [str(c).strip() for c in chunks if str(c).strip()]
    logger.info("Parsed %d sections from URL", len(sections))
    return sections


async def parse_document(
    file_obj: io.IOBase | None,
    file_type: str,
    source_url: str = "",
) -> list[str]:
    """
    Async entry point.
    - file_type == 'url': fetches source_url and parses HTML
    - otherwise:         runs unstructured in a thread executor
    """
    if file_type.lower() == "url":
        if not source_url:
            raise ValueError("source_url is required for file_type='url'")
        return await _parse_url(source_url)

    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _parse_file_sync, file_obj, file_type)
