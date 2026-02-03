from __future__ import annotations

from typing import Callable

from fastapi import Request


def attach_host_context(request: Request) -> None:
    request.state.host_proxy = request.headers.get("x-powerx-proxy")
    request.state.plugin_id = request.headers.get("x-plugin-id")

    if not request.state.plugin_id:
        path = request.url.path
        if path.startswith("/_p/"):
            parts = path.split("/", 3)
            if len(parts) >= 3:
                request.state.plugin_id = parts[2]


async def host_context_middleware(request: Request, call_next: Callable):
    attach_host_context(request)
    response = await call_next(request)
    return response
