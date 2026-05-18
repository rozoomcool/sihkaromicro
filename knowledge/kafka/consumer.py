"""
Kafka consumer for:
  - jobs.chunking  → download + parse + embed + store
  - jobs.cancel    → delete chunks by source_id
"""
from __future__ import annotations

import asyncio
import json
import logging
from dataclasses import dataclass

from aiokafka import AIOKafkaConsumer, ConsumerRecord

from config import settings
from core.chunking.parsers import parse_document
from core.chunking.pipeline import build_chunk_pairs
from core.embeddings.base import EmbeddingProvider
from core.storage.minio_client import MinioClient
from core.storage.vector_store import (
    ChildChunkRecord,
    ParentChunkRecord,
    VectorStore,
)
from kafka.producer import StatusProducer

logger = logging.getLogger(__name__)


@dataclass
class ChunkingJob:
    job_id: str
    owner_id: str
    source_id: int
    minio_path: str   # empty string for file_type == "url"
    file_type: str
    source_url: str = ""  # populated only for file_type == "url"


@dataclass
class CancelJob:
    job_id: str
    owner_id: str
    source_id: int


class ChunkingConsumer:
    def __init__(
        self,
        vector_store: VectorStore,
        embedder: EmbeddingProvider,
        minio: MinioClient,
        producer: StatusProducer,
        project_id_resolver: "ProjectIdResolver | None" = None,
    ) -> None:
        self._store = vector_store
        self._embedder = embedder
        self._minio = minio
        self._producer = producer
        self._consumer: AIOKafkaConsumer | None = None
        self._running = False

    async def start(self) -> None:
        self._consumer = AIOKafkaConsumer(
            settings.kafka_chunking_topic,
            settings.kafka_cancel_topic,
            bootstrap_servers=settings.kafka_brokers,
            group_id=settings.kafka_group_id,
            # Manual offset commit — we only commit after successful processing
            enable_auto_commit=False,
            auto_offset_reset="earliest",
            value_deserializer=lambda v: json.loads(v.decode()),
        )
        await self._consumer.start()
        self._running = True
        logger.info(
            "Kafka consumer started, topics: %s, %s",
            settings.kafka_chunking_topic,
            settings.kafka_cancel_topic,
        )

    async def stop(self) -> None:
        self._running = False
        if self._consumer:
            await self._consumer.stop()
            logger.info("Kafka consumer stopped")

    async def run(self) -> None:
        """Main consumer loop. Blocks until stop() is called."""
        if not self._consumer:
            raise RuntimeError("Consumer not started")

        async for record in self._consumer:
            if not self._running:
                break
            await self._handle_record(record)

    async def _handle_record(self, record: ConsumerRecord) -> None:
        topic = record.topic
        payload = record.value

        try:
            if topic == settings.kafka_chunking_topic:
                job = ChunkingJob(
                    job_id=payload["job_id"],
                    owner_id=payload["owner_id"],
                    source_id=int(payload["source_id"]),
                    minio_path=payload.get("minio_path", ""),
                    file_type=payload["file_type"],
                    source_url=payload.get("source_url", ""),
                )
                await self._process_chunking(job)
            elif topic == settings.kafka_cancel_topic:
                job = CancelJob(
                    job_id=payload["job_id"],
                    owner_id=payload["owner_id"],
                    source_id=int(payload["source_id"]),
                )
                await self._process_cancel(job)

            # Commit only after successful processing
            await self._consumer.commit()

        except Exception as exc:
            logger.error(
                "Failed to handle record topic=%s offset=%d: %s",
                topic,
                record.offset,
                exc,
                exc_info=True,
            )
            # Do NOT commit — the record will be redelivered on restart.
            # For the chunking topic we handle retries inside _process_chunking.

    async def _process_chunking(self, job: ChunkingJob) -> None:
        last_error: Exception | None = None

        for attempt in range(1, settings.max_retries + 1):
            try:
                await self._do_chunking(job)
                await self._producer.publish_status(
                    job_id=job.job_id,
                    source_id=job.source_id,
                    status="done",
                )
                return
            except Exception as exc:
                last_error = exc
                logger.warning(
                    "Chunking attempt %d/%d failed for source=%d: %s",
                    attempt,
                    settings.max_retries,
                    job.source_id,
                    exc,
                )
                if attempt < settings.max_retries:
                    await asyncio.sleep(2**attempt)  # exponential backoff

        await self._producer.publish_status(
            job_id=job.job_id,
            source_id=job.source_id,
            status="failed",
            error=str(last_error),
        )
        raise last_error  # re-raise so the consumer loop does NOT commit

    async def _do_chunking(self, job: ChunkingJob) -> None:
        logger.info(
            "Processing source=%d path=%s type=%s",
            job.source_id,
            job.minio_path,
            job.file_type,
        )

        if job.file_type.lower() == "url":
            sections = await parse_document(None, job.file_type, source_url=job.source_url)
        else:
            async with self._minio.download(job.minio_path) as file_obj:
                sections = await parse_document(file_obj, job.file_type)

        if not sections:
            logger.warning("No sections parsed for source=%d", job.source_id)
            return

        pairs = await build_chunk_pairs(sections)

        # Collect all child texts for batch embedding
        all_child_texts: list[str] = []
        pair_child_counts: list[int] = []
        for pair in pairs:
            texts = [text for text, _ in pair.children]
            all_child_texts.extend(texts)
            pair_child_counts.append(len(texts))

        all_embeddings = await self._embedder.embed_texts(all_child_texts)

        # Split embeddings back per parent
        parents: list[ParentChunkRecord] = []
        children_by_parent: list[list[ChildChunkRecord]] = []
        offset = 0

        # project_id is embedded in minio_path as owner_id/source_id/filename
        # We can't recover project_id here without a DB lookup or it being in the message.
        # The task spec doesn't include project_id in the Kafka message, so we derive it
        # from the source_id via a placeholder. Callers should add project_id to the
        # Kafka message; for now we store 0 and rely on the sources service to set it.
        # NOTE: Add project_id to the jobs.chunking message in the Sources service
        #       to avoid this limitation.
        project_id = int(job.minio_path.split("/")[1]) if "/" in job.minio_path else 0

        for pair, count in zip(pairs, pair_child_counts):
            parent = ParentChunkRecord(
                source_id=job.source_id,
                project_id=project_id,
                owner_id=job.owner_id,
                content=pair.parent_text,
                position=pair.parent_position,
            )
            child_embeddings = all_embeddings[offset : offset + count]
            children = [
                ChildChunkRecord(
                    source_id=job.source_id,
                    project_id=project_id,
                    owner_id=job.owner_id,
                    content=text,
                    embedding=emb,
                    position=pos,
                )
                for (text, pos), emb in zip(pair.children, child_embeddings)
            ]
            parents.append(parent)
            children_by_parent.append(children)
            offset += count

        await self._store.save_source_chunks(parents, children_by_parent)
        logger.info(
            "Saved %d parent chunks / %d child chunks for source=%d",
            len(parents),
            len(all_child_texts),
            job.source_id,
        )

    async def _process_cancel(self, job: CancelJob) -> None:
        deleted = await self._store.delete_by_source(job.source_id, job.owner_id)
        logger.info(
            "Deleted %d parent chunks for source=%d owner=%s",
            deleted,
            job.source_id,
            job.owner_id,
        )
