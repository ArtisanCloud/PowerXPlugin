from __future__ import annotations

import json
import threading
import time
import urllib.error
import urllib.request


class STSExchangeError(RuntimeError):
    pass


def exchange_sts(
    endpoint: str,
    client_id: str,
    client_secret: str,
    audience: str = "",
    scope: str = "",
    ttl_seconds: int = 300,
) -> tuple[str, int]:
    if not endpoint:
        raise STSExchangeError("sts endpoint not configured")
    payload = {
        "client_id": client_id,
        "client_secret": client_secret,
        "audience": audience,
        "scope": scope,
        "ttl": ttl_seconds,
    }
    body = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(
        endpoint,
        data=body,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            raw = resp.read()
    except urllib.error.URLError as exc:
        raise STSExchangeError(str(exc)) from exc
    try:
        data = json.loads(raw.decode("utf-8"))
    except json.JSONDecodeError as exc:
        raise STSExchangeError("invalid sts response") from exc
    if isinstance(data, dict) and "data" in data:
        data = data.get("data") or {}
    if not isinstance(data, dict):
        raise STSExchangeError("invalid sts response")
    token = data.get("access_token") or ""
    expires_in = int(data.get("expires_in") or 0)
    if not token:
        raise STSExchangeError("empty access token from sts")
    return token, expires_in


class TokenManager:
    def __init__(
        self,
        endpoint: str,
        client_id: str,
        client_secret: str,
        audience: str = "",
        scope: str = "",
        ttl_seconds: int = 300,
    ) -> None:
        self.endpoint = endpoint
        self.client_id = client_id
        self.client_secret = client_secret
        self.audience = audience
        self.scope = scope
        self.ttl_seconds = ttl_seconds
        self._lock = threading.RLock()
        self._token: str = ""
        self._expiry: float = 0.0

    def get_token(self) -> str:
        token = self._peek()
        if token:
            return token
        token, _ = self._refresh()
        return token

    def exchange_now(self) -> tuple[str, int]:
        token, expiry = self._refresh()
        expires_in = int(max(0, expiry - time.time()))
        return token, expires_in

    def invalidate(self) -> None:
        with self._lock:
            self._token = ""
            self._expiry = 0.0

    def _peek(self) -> str:
        with self._lock:
            if not self._token:
                return ""
            if time.time() >= (self._expiry - 60):
                return ""
            return self._token

    def _refresh(self) -> tuple[str, float]:
        with self._lock:
            if self._token and time.time() < (self._expiry - 60):
                return self._token, self._expiry
            token, expires_in = exchange_sts(
                self.endpoint,
                self.client_id,
                self.client_secret,
                self.audience,
                self.scope,
                self.ttl_seconds,
            )
            ttl = expires_in or self.ttl_seconds
            self._token = token
            self._expiry = time.time() + ttl
            return self._token, self._expiry


_manager_lock = threading.RLock()
_manager: TokenManager | None = None
_manager_key: tuple[str, str, str, str, str] | None = None


def get_manager(
    endpoint: str,
    client_id: str,
    client_secret: str,
    audience: str = "",
    scope: str = "",
    ttl_seconds: int = 300,
) -> TokenManager:
    global _manager, _manager_key
    key = (endpoint, client_id, client_secret, audience, scope)
    with _manager_lock:
        if _manager is None or _manager_key != key:
            _manager = TokenManager(
                endpoint=endpoint,
                client_id=client_id,
                client_secret=client_secret,
                audience=audience,
                scope=scope,
                ttl_seconds=ttl_seconds,
            )
            _manager_key = key
        return _manager
