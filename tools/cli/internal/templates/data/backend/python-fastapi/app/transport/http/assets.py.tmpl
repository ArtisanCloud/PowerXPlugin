from datetime import datetime
from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse

router = APIRouter()


def _request_id(request: Request) -> str | None:
    return request.headers.get("X-Request-ID") or request.headers.get("Request-ID")


def _build_payload(build_id: str) -> dict:
    return {
        "id": build_id,
        "timestamp": int(datetime.utcnow().timestamp() * 1000),
        "matcher": {"static": {}, "wildcard": {}, "dynamic": {}},
        "prerendered": [],
    }


@router.get("/assets/builds/meta")
async def build_meta_default(request: Request):
    payload = _build_payload("dev")
    response = JSONResponse(status_code=200, content=payload)
    request_id = _request_id(request)
    if request_id:
        response.headers["X-Request-Id"] = request_id
    return response


@router.get("/assets/builds/meta/{build_id}")
async def build_meta(request: Request, build_id: str):
    build = (build_id or "dev").removesuffix(".json")
    if not build:
        build = "dev"
    payload = _build_payload(build)
    response = JSONResponse(status_code=200, content=payload)
    request_id = _request_id(request)
    if request_id:
        response.headers["X-Request-Id"] = request_id
    return response
