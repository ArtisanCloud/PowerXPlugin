from __future__ import annotations

import json
import logging
from typing import Any, Callable, Iterable

from fastapi import WebSocket, WebSocketDisconnect, FastAPI

from app.middleware.auth_guard import JWTAuthConfig
from app.middleware import auth_guard
from app.services.ws_bus import Event

ALLOWED_TOPICS = {
    "_topic.template.update",
    "_topic.audit.template.updated",
    "_topic.template.validate.completed",
    "_topic.template.batch_clone.completed",
    "_topic.template.update.completed",
}


class _Subscriber:
    async def subscribe(self, topic: str, handler: Callable[[Event], Any]) -> Callable[[], Any]:
        raise NotImplementedError


async def register_ws_routes(app: FastAPI) -> None:
    if app is None:
        return

    @app.websocket("/api/ws")
    async def api_ws_endpoint(websocket: WebSocket):
        await _handle_ws(websocket, app)


async def _handle_ws(websocket: WebSocket, app: FastAPI) -> None:
    hub: _Subscriber | None = getattr(app.state, "ws_bus_hub", None)
    jwt_cfg: JWTAuthConfig | None = getattr(app.state, "jwt_cfg", None)
    if hub is None:
        await websocket.close(code=1011)
        return

    tenant_uuid, ok = _resolve_tenant(websocket, jwt_cfg)
    if not ok and jwt_cfg is not None and not jwt_cfg.optional:
        await websocket.close(code=1008)
        return

    await websocket.accept()
    subs: dict[str, Callable[[], Any]] = {}
    try:
        while True:
            raw = await websocket.receive_text()
            try:
                data = json.loads(raw)
            except json.JSONDecodeError:
                await _send(websocket, {"type": "error", "message": "invalid json"})
                continue
            msg_type = str(data.get("type") or "").strip().lower()
            topics = data.get("topics") or []
            if msg_type == "subscribe":
                await _subscribe(websocket, hub, subs, tenant_uuid, topics)
            elif msg_type == "unsubscribe":
                await _unsubscribe(subs, topics)
            else:
                await _send(websocket, {"type": "error", "message": "unknown message type"})
    except WebSocketDisconnect:
        pass
    finally:
        await _unsubscribe(subs, list(subs.keys()))


def _resolve_tenant(websocket: WebSocket, cfg: JWTAuthConfig | None) -> tuple[str, bool]:
    tenant = (websocket.query_params.get("tenant_uuid") or "").strip()
    authz = (websocket.query_params.get("authorization") or "").strip()
    if not authz:
        authz = (websocket.headers.get("authorization") or "").strip()

    if cfg is None:
        return tenant, bool(tenant)

    if authz.lower().startswith("bearer "):
        raw = authz[7:].strip()
        if raw and cfg.hmac_secret:
            claims = auth_guard._decode_and_verify_jwt(raw, cfg)
            if isinstance(claims, dict):
                tc = auth_guard._claims_to_context(claims)
                if tc.tenant_uuid:
                    return tc.tenant_uuid, True
    if cfg.allow_signed_context and cfg.context_hmac_secret:
        req = _WebSocketRequestAdapter(websocket, authz)
        tc = auth_guard._try_load_signed_context(req, cfg.context_hmac_secret, cfg.max_ctx_age_seconds)
        if tc is not None and tc.tenant_uuid:
            return tc.tenant_uuid, True
    return tenant, bool(tenant)


async def _subscribe(
    websocket: WebSocket,
    hub: _Subscriber,
    subs: dict[str, Callable[[], Any]],
    tenant_uuid: str,
    topics: Iterable[Any],
) -> None:
    for topic in topics:
        clean = str(topic or "").strip()
        if not clean or clean in subs:
            continue
        if clean not in ALLOWED_TOPICS:
            await _send(websocket, {"type": "error", "message": "topic not allowed", "topic": clean})
            continue

        async def _handler(event: Event, topic_name: str = clean):
            if tenant_uuid and event.tenant_uuid and tenant_uuid != event.tenant_uuid:
                return
            await _send(websocket, {"type": "event", "topic": topic_name, "payload": event.payload})

        unsub = await hub.subscribe(clean, _handler)
        subs[clean] = unsub


async def _unsubscribe(subs: dict[str, Callable[[], Any]], topics: Iterable[Any]) -> None:
    for topic in topics:
        clean = str(topic or "").strip()
        if not clean or clean not in subs:
            continue
        try:
            await subs[clean]()
        except TypeError:
            subs[clean]()
        subs.pop(clean, None)


async def _send(websocket: WebSocket, payload: dict[str, Any]) -> None:
    await websocket.send_text(json.dumps(payload, ensure_ascii=False))


class _WebSocketRequestAdapter:
    def __init__(self, websocket: WebSocket, authz: str) -> None:
        self.headers = _HeaderAdapter(websocket, authz)


class _HeaderAdapter:
    def __init__(self, websocket: WebSocket, authz: str) -> None:
        self._headers = websocket.headers
        self._authz = authz

    def get(self, name: str, default: str | None = None) -> str:
        if name.lower() == "authorization" and self._authz:
            return self._authz
        return self._headers.get(name, default or "")
