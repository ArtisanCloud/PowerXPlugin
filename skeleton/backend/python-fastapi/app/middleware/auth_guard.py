from typing import Callable

from fastapi import Request, HTTPException


async def auth_guard_middleware(request: Request, call_next: Callable):
    # TODO: replace with real auth verification
    if request.url.path.startswith("/admin/user/auth"):
        return await call_next(request)

    token = request.headers.get("authorization")
    if not token:
        raise HTTPException(status_code=401, detail="unauthorized")

    response = await call_next(request)
    return response
