from fastapi import Request
from fastapi.responses import JSONResponse
from starlette.responses import Response


_ALLOW_HEADERS = (
    "Content-Type, Content-Length, Accept-Encoding, "
    "X-CSRF-Token, Authorization, accept, origin, Cache-Control, "
    "X-Requested-With, tenant_uuid, X-PowerX-CTX, X-PowerX-CTX-SIG, "
    "X-PowerX-CTX-JWT, X-Request-ID, X-Trace-Id"
)
_ALLOW_METHODS = "POST, OPTIONS, GET, PUT, DELETE, PATCH"
_EXPOSE_HEADERS = "X-Trace-Id, X-Correlation-Id, X-Request-Id"


def _apply_headers(request: Request, response: Response) -> None:
    origin = (request.headers.get("origin") or "").strip() or "*"
    response.headers["Access-Control-Allow-Origin"] = origin
    response.headers["Vary"] = "Origin"
    response.headers["Access-Control-Allow-Credentials"] = "true"
    response.headers["Access-Control-Allow-Headers"] = _ALLOW_HEADERS
    response.headers["Access-Control-Allow-Methods"] = _ALLOW_METHODS
    response.headers["Access-Control-Expose-Headers"] = _EXPOSE_HEADERS


async def cors_middleware(request: Request, call_next):
    if request.method.upper() == "OPTIONS":
        response = Response(status_code=204)
        _apply_headers(request, response)
        return response

    try:
        response = await call_next(request)
    except Exception:
        response = JSONResponse(
            status_code=500,
            content={
                "success": False,
                "message": "internal server error",
                "error": {"code": "INTERNAL_ERROR", "message": "internal server error"},
            },
        )
    _apply_headers(request, response)
    return response
