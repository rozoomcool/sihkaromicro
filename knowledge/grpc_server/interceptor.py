"""
JWT verification against Keycloak JWKS endpoint.
owner_id is extracted from the 'sub' claim — the same field the Go services use.
"""
from __future__ import annotations

import logging
import time
from typing import Any

import httpx
from jose import JWTError, jwt

from config import settings

logger = logging.getLogger(__name__)

# Simple in-process JWKS cache (TTL = 10 minutes)
_JWKS_CACHE: dict[str, Any] = {}
_JWKS_FETCHED_AT: float = 0.0
_JWKS_TTL = 600


class KeycloakVerifier:
    """Fetches Keycloak public keys and verifies RS256 JWT tokens."""

    def __init__(self) -> None:
        self._jwks_uri = (
            f"{settings.keycloak_url}/realms/{settings.keycloak_realm}"
            "/protocol/openid-connect/certs"
        )

    async def _get_jwks(self) -> dict:
        global _JWKS_CACHE, _JWKS_FETCHED_AT
        now = time.monotonic()
        if _JWKS_CACHE and (now - _JWKS_FETCHED_AT) < _JWKS_TTL:
            return _JWKS_CACHE

        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.get(self._jwks_uri)
            resp.raise_for_status()
            _JWKS_CACHE = resp.json()
            _JWKS_FETCHED_AT = now
            logger.debug("Refreshed Keycloak JWKS from %s", self._jwks_uri)

        return _JWKS_CACHE

    async def verify(self, token: str) -> str:
        """
        Verifies the JWT and returns the subject (owner_id / Keycloak sub).
        Raises ValueError on any verification failure.
        """
        jwks = await self._get_jwks()
        try:
            claims = jwt.decode(
                token,
                jwks,
                algorithms=["RS256"],
                options={"verify_aud": False},
            )
        except JWTError as exc:
            raise ValueError(f"JWT validation failed: {exc}") from exc

        sub = claims.get("sub")
        if not sub:
            raise ValueError("JWT missing 'sub' claim")

        return sub
