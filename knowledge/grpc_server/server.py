from __future__ import annotations

import logging

import grpc
import grpc.aio

from config import settings
from gen import rag_pb2_grpc
from grpc_server.handlers import RAGServicer

logger = logging.getLogger(__name__)


async def create_server(servicer: RAGServicer) -> grpc.aio.Server:
    server = grpc.aio.server()
    rag_pb2_grpc.add_RAGServiceServicer_to_server(servicer, server)

    listen_addr = f"[::]:{settings.grpc_port}"
    server.add_insecure_port(listen_addr)
    logger.info("gRPC server configured on %s", listen_addr)
    return server
