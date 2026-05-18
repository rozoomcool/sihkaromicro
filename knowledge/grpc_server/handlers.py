"""
gRPC service handlers for RAGService.

Security invariant: owner_id is ALWAYS extracted from the verified JWT token.
It is never read from the request body (owner_id field in proto is ignored).
"""
from __future__ import annotations

import json
import logging

import grpc

from core.retrieval.rag import RAGService
from core.retrieval.search import RetrievalService
from gen import rag_pb2, rag_pb2_grpc
from grpc_server.interceptor import KeycloakVerifier

logger = logging.getLogger(__name__)


class RAGServicer(rag_pb2_grpc.RAGServiceServicer):
    def __init__(
        self,
        retrieval: RetrievalService,
        rag: RAGService,
        verifier: KeycloakVerifier,
    ) -> None:
        self._retrieval = retrieval
        self._rag = rag
        self._verifier = verifier

    # ------------------------------------------------------------------ #
    # Auth helper — extracts owner_id from JWT in gRPC metadata           #
    # ------------------------------------------------------------------ #

    async def _authenticate(self, context: grpc.aio.ServicerContext) -> str | None:
        """Returns owner_id or aborts the RPC with UNAUTHENTICATED."""
        metadata = dict(context.invocation_metadata())
        auth = metadata.get("authorization", "")

        if not auth.startswith("Bearer "):
            await context.abort(grpc.StatusCode.UNAUTHENTICATED, "missing authorization token")
            return None

        token = auth.removeprefix("Bearer ")
        try:
            return await self._verifier.verify(token)
        except ValueError as exc:
            await context.abort(grpc.StatusCode.UNAUTHENTICATED, str(exc))
            return None

    # ------------------------------------------------------------------ #
    # RPC handlers                                                         #
    # ------------------------------------------------------------------ #

    async def Search(
        self,
        request: rag_pb2.SearchRequest,
        context: grpc.aio.ServicerContext,
    ) -> rag_pb2.SearchResponse:
        owner_id = await self._authenticate(context)
        if owner_id is None:
            return rag_pb2.SearchResponse()

        top_k = request.top_k if request.top_k > 0 else 20
        contexts = await self._retrieval.retrieve(
            query=request.query,
            owner_id=owner_id,        # from JWT — not from request.owner_id
            project_id=request.project_id,
            top_k=top_k,
        )

        results = [
            rag_pb2.SearchResult(
                content=ctx.parent_content,
                score=ctx.rerank_score,
                source_id=ctx.source_id,
                metadata=json.dumps(ctx.metadata) if ctx.metadata else "",
            )
            for ctx in contexts
        ]
        return rag_pb2.SearchResponse(results=results)

    async def Generate(
        self,
        request: rag_pb2.GenerateRequest,
        context: grpc.aio.ServicerContext,
    ) -> rag_pb2.GenerateResponse:
        owner_id = await self._authenticate(context)
        if owner_id is None:
            return rag_pb2.GenerateResponse()

        answer, contexts = await self._rag.generate(
            query=request.query,
            owner_id=owner_id,        # from JWT
            project_id=request.project_id,
        )

        sources = [
            rag_pb2.SearchResult(
                content=ctx.parent_content,
                score=ctx.rerank_score,
                source_id=ctx.source_id,
                metadata=json.dumps(ctx.metadata) if ctx.metadata else "",
            )
            for ctx in contexts
        ]
        return rag_pb2.GenerateResponse(answer=answer, sources=sources)

    async def GenerateStream(
        self,
        request: rag_pb2.GenerateRequest,
        context: grpc.aio.ServicerContext,
    ):
        owner_id = await self._authenticate(context)
        if owner_id is None:
            return

        async for text_chunk in self._rag.generate_stream(
            query=request.query,
            owner_id=owner_id,        # from JWT
            project_id=request.project_id,
        ):
            yield rag_pb2.GenerateChunk(text=text_chunk, done=False)

        yield rag_pb2.GenerateChunk(text="", done=True)
