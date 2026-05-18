from abc import ABC, abstractmethod


class Reranker(ABC):
    @abstractmethod
    async def rerank(
        self,
        query: str,
        passages: list[str],
        top_k: int = 5,
    ) -> list[tuple[int, float]]:
        """
        Returns (original_index, score) pairs for the top-k passages,
        sorted by score descending.
        """
