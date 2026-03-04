from datetime import datetime
from fastapi import APIRouter, Request

from app.contracts.response import ok

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
    request_id = _request_id(request)
    return ok(_build_payload("dev"), request_id=request_id)


@router.get("/assets/builds/meta/{build_id}")
async def build_meta(request: Request, build_id: str):
    request_id = _request_id(request)
    build = (build_id or "dev").removesuffix(".json")
    if not build:
        build = "dev"
    return ok(_build_payload(build), request_id=request_id)
