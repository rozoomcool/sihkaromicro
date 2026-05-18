import asyncio
import logging

from core.reranker.base import Reranker

logger = logging.getLogger(__name__)

_MODEL_NAME = "cross-encoder/ms-marco-MiniLM-L-6-v2"


class CrossEncoderReranker(Reranker):
    def __init__(self) -> None:
        self._model = None  # lazy-loaded on first use

    def _load(self) -> None:
        if self._model is None:
            from sentence_transformers import CrossEncoder
            logger.info("Loading CrossEncoder model %s", _MODEL_NAME)
            self._model = CrossEncoder(_MODEL_NAME)

    def _score_sync(self, query: str, passages: list[str]) -> list[float]:
        self._load()
        pairs = [(query, p) for p in passages]
        scores: list[float] = self._model.predict(pairs).tolist()
        return scores

    async def rerank(
        self,
        query: str,
        passages: list[str],
        top_k: int = 5,
    ) -> list[tuple[int, float]]:
        if not passages:
            return []

        loop = asyncio.get_event_loop()
        scores = await loop.run_in_executor(None, self._score_sync, query, passages)

        indexed = sorted(enumerate(scores), key=lambda x: x[1], reverse=True)
        return indexed[:top_k]
