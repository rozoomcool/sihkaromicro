"""
Knowledge Service — entry point.

Runs two concurrent components in the same asyncio event loop:
  1. gRPC server (RAGService) on port 50054
  2. Kafka consumer (jobs.chunking + jobs.cancel)

Shutdown is graceful: SIGTERM/SIGINT waits for the current Kafka message
to finish processing before exiting.
"""
from __future__ import annotations

import asyncio
import logging
import os
import signal
import sys

# Make the project root importable regardless of working directory
sys.path.insert(0, os.path.dirname(__file__))

from config import settings
from core.embeddings.openai_provider import OpenAIEmbeddingProvider
from core.reranker.cross_encoder import CrossEncoderReranker
from core.retrieval.rag import RAGService
from core.retrieval.search import RetrievalService
from core.storage.minio_client import MinioClient
from core.storage.vector_store import VectorStore, create_pool
from grpc_server.handlers import RAGServicer
from grpc_server.interceptor import KeycloakVerifier
from grpc_server.server import create_server
from kafka.consumer import ChunkingConsumer
from kafka.producer import StatusProducer

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger(__name__)


async def run_migrations(pool) -> None:
    migration_path = os.path.join(os.path.dirname(__file__), "migrations", "001_create_tables.sql")
    with open(migration_path) as f:
        sql = f.read()
    async with pool.acquire() as conn:
        await conn.execute(sql)
    logger.info("Migrations applied")


async def main() -> None:
    # ── Database ───────────────────────────────────────────────────────
    logger.info("Connecting to database…")
    pool = await create_pool(settings.db_dsn)
    await run_migrations(pool)

    # ── Shared dependencies ────────────────────────────────────────────
    vector_store = VectorStore(pool)
    embedder = OpenAIEmbeddingProvider()
    reranker = CrossEncoderReranker()
    minio = MinioClient()
    verifier = KeycloakVerifier()

    retrieval = RetrievalService(
        vector_store=vector_store,
        embedder=embedder,
        reranker=reranker,
    )
    rag = RAGService(retrieval=retrieval)

    # ── Kafka ──────────────────────────────────────────────────────────
    producer = StatusProducer()
    await producer.start()

    consumer = ChunkingConsumer(
        vector_store=vector_store,
        embedder=embedder,
        minio=minio,
        producer=producer,
    )
    await consumer.start()

    # ── gRPC server ────────────────────────────────────────────────────
    servicer = RAGServicer(retrieval=retrieval, rag=rag, verifier=verifier)
    server = await create_server(servicer)
    await server.start()
    logger.info("gRPC server started on port %d", settings.grpc_port)

    # ── Graceful shutdown ──────────────────────────────────────────────
    stop_event = asyncio.Event()

    def _on_signal() -> None:
        logger.info("Shutdown signal received")
        stop_event.set()

    loop = asyncio.get_event_loop()
    for sig in (signal.SIGTERM, signal.SIGINT):
        loop.add_signal_handler(sig, _on_signal)

    # Run consumer in background; stop when signal arrives
    consumer_task = asyncio.create_task(consumer.run(), name="kafka-consumer")

    await stop_event.wait()

    logger.info("Shutting down…")
    await consumer.stop()
    consumer_task.cancel()
    try:
        await consumer_task
    except asyncio.CancelledError:
        pass

    await server.stop(grace=5)
    await producer.stop()
    await pool.close()
    logger.info("Shutdown complete")


if __name__ == "__main__":
    asyncio.run(main())
