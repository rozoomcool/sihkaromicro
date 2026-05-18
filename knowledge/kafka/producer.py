from __future__ import annotations

import json
import logging

from aiokafka import AIOKafkaProducer

from config import settings

logger = logging.getLogger(__name__)


class StatusProducer:
    def __init__(self) -> None:
        self._producer: AIOKafkaProducer | None = None

    async def start(self) -> None:
        self._producer = AIOKafkaProducer(
            bootstrap_servers=settings.kafka_brokers,
            value_serializer=lambda v: json.dumps(v).encode(),
        )
        await self._producer.start()
        logger.info("Kafka producer started")

    async def stop(self) -> None:
        if self._producer:
            await self._producer.stop()
            logger.info("Kafka producer stopped")

    async def publish_status(
        self,
        job_id: str,
        source_id: int,
        status: str,
        error: str = "",
    ) -> None:
        if not self._producer:
            raise RuntimeError("Producer not started")

        payload = {
            "job_id": job_id,
            "source_id": source_id,
            "status": status,
            "error": error,
        }
        await self._producer.send_and_wait(
            settings.kafka_status_topic,
            value=payload,
            key=job_id.encode(),
        )
        logger.info("Published status=%s for job=%s source=%d", status, job_id, source_id)
