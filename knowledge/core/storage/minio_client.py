import asyncio
import io
import logging
import os
import tempfile
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

from minio import Minio

from config import settings

logger = logging.getLogger(__name__)


class MinioClient:
    def __init__(self) -> None:
        self._client = Minio(
            settings.minio_endpoint,
            access_key=settings.minio_access_key,
            secret_key=settings.minio_secret_key,
            secure=settings.minio_secure,
        )

    async def get_object_size(self, minio_path: str) -> int:
        loop = asyncio.get_event_loop()
        stat = await loop.run_in_executor(
            None,
            lambda: self._client.stat_object(settings.minio_bucket, minio_path),
        )
        return stat.size

    @asynccontextmanager
    async def download(self, minio_path: str) -> AsyncIterator[io.IOBase]:
        """
        Context manager yielding a readable file-like object.
        Files under max_in_memory_bytes are kept in BytesIO;
        larger files are streamed to a NamedTemporaryFile (cleaned up on exit).
        """
        size = await self.get_object_size(minio_path)

        if size < settings.max_in_memory_bytes:
            data = await self._download_to_memory(minio_path)
            yield io.BytesIO(data)
        else:
            async with self._download_to_disk(minio_path) as fp:
                yield fp

    async def _download_to_memory(self, minio_path: str) -> bytes:
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(None, self._fetch_bytes, minio_path)

    def _fetch_bytes(self, minio_path: str) -> bytes:
        response = self._client.get_object(settings.minio_bucket, minio_path)
        try:
            return response.read()
        finally:
            response.close()
            response.release_conn()

    @asynccontextmanager
    async def _download_to_disk(self, minio_path: str) -> AsyncIterator[io.IOBase]:
        tmp = tempfile.NamedTemporaryFile(delete=False)
        try:
            loop = asyncio.get_event_loop()
            await loop.run_in_executor(None, self._stream_to_file, minio_path, tmp.name)
            tmp.close()
            with open(tmp.name, "rb") as fp:
                yield fp
        finally:
            try:
                os.unlink(tmp.name)
            except OSError:
                pass

    def _stream_to_file(self, minio_path: str, dest: str) -> None:
        response = self._client.get_object(settings.minio_bucket, minio_path)
        try:
            with open(dest, "wb") as fp:
                for chunk in response.stream(amt=8 * 1024 * 1024):
                    fp.write(chunk)
        finally:
            response.close()
            response.release_conn()
