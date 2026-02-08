from __future__ import annotations

import asyncio
import json
import logging
import uuid
from dataclasses import dataclass
from typing import Any, Callable, Dict

from redis.asyncio import Redis
from redis.asyncio.client import PubSub


@dataclass(frozen=True)
class Event:
    topic: str
    payload: Any
    tenant_uuid: str
    trace_id: str


class MemoryHub:
    def __init__(self) -> None:
        self._lock = asyncio.Lock()
        self._subs: Dict[str, Dict[str, Callable[[Event], Any]]] = {}

    async def publish(self, topic: str, payload: Any, tenant_uuid: str = "", trace_id: str = "") -> None:
        async with self._lock:
            subs = list(self._subs.get(topic, {}).values())
        if not subs:
            return
        event = Event(topic=topic, payload=payload, tenant_uuid=tenant_uuid, trace_id=trace_id)
        for handler in subs:
            if handler is None:
                continue
            if asyncio.iscoroutinefunction(handler):
                asyncio.create_task(handler(event))
            else:
                handler(event)

    async def subscribe(self, topic: str, handler: Callable[[Event], Any]) -> Callable[[], Any]:
        sub_id = uuid.uuid4().hex
        async with self._lock:
            self._subs.setdefault(topic, {})[sub_id] = handler

        async def _unsubscribe() -> None:
            async with self._lock:
                if topic in self._subs and sub_id in self._subs[topic]:
                    del self._subs[topic][sub_id]
                    if not self._subs[topic]:
                        del self._subs[topic]

        return _unsubscribe


class RedisHub:
    def __init__(self, redis_url: str, channel: str = "powerx.wsbus", logger: logging.Logger | None = None) -> None:
        self._redis = Redis.from_url(redis_url, decode_responses=True)
        self._channel = channel or "powerx.wsbus"
        self._instance_id = uuid.uuid4().hex
        self._local = MemoryHub()
        self._logger = logger or logging.getLogger("ws_bus")
        self._pubsub: PubSub | None = None
        self._task: asyncio.Task | None = None

    async def start(self) -> None:
        if self._task is not None:
            return
        self._pubsub = self._redis.pubsub()
        await self._pubsub.subscribe(self._channel)
        self._task = asyncio.create_task(self._run())

    async def close(self) -> None:
        if self._task:
            self._task.cancel()
        if self._pubsub:
            await self._pubsub.close()
        await self._redis.close()

    async def publish(self, topic: str, payload: Any, tenant_uuid: str = "", trace_id: str = "") -> None:
        envelope = {
            "topic": topic,
            "payload": payload,
            "tenant_uuid": tenant_uuid,
            "trace_id": trace_id,
            "origin": self._instance_id,
        }
        data = json.dumps(envelope, ensure_ascii=False)
        await self._redis.publish(self._channel, data)
        await self._local.publish(topic, payload, tenant_uuid=tenant_uuid, trace_id=trace_id)

    async def subscribe(self, topic: str, handler: Callable[[Event], Any]) -> Callable[[], Any]:
        return await self._local.subscribe(topic, handler)

    async def _run(self) -> None:
        if self._pubsub is None:
            return
        async for msg in self._pubsub.listen():
            if msg is None or msg.get("type") != "message":
                continue
            payload = msg.get("data")
            if not payload:
                continue
            try:
                envelope = json.loads(payload)
            except json.JSONDecodeError:
                self._logger.warning("ws bus redis: invalid payload")
                continue
            if envelope.get("origin") == self._instance_id:
                continue
            await self._local.publish(
                envelope.get("topic", ""),
                envelope.get("payload"),
                tenant_uuid=envelope.get("tenant_uuid", ""),
                trace_id=envelope.get("trace_id", ""),
            )
