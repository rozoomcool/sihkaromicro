import logging

from openai import AsyncOpenAI

from config import settings
from core.embeddings.base import EmbeddingProvider

logger = logging.getLogger(__name__)

# Batch size to stay within OpenAI's tokens-per-minute limits
_BATCH_SIZE = 100


class OpenAIEmbeddingProvider(EmbeddingProvider):
    def __init__(self) -> None:
        self._client = AsyncOpenAI(api_key=settings.openai_api_key)
        self._model = settings.embedding_model

    async def embed_texts(self, texts: list[str]) -> list[list[float]]:
        if not texts:
            return []

        all_embeddings: list[list[float]] = []
        for i in range(0, len(texts), _BATCH_SIZE):
            batch = texts[i : i + _BATCH_SIZE]
            response = await self._client.embeddings.create(
                model=self._model,
                input=batch,
            )
            # OpenAI returns embeddings ordered by index
            batch_embeddings = [item.embedding for item in sorted(response.data, key=lambda x: x.index)]
            all_embeddings.extend(batch_embeddings)
            logger.debug("Embedded batch %d-%d", i, i + len(batch))

        return all_embeddings
