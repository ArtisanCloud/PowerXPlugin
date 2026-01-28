from dataclasses import dataclass
from datetime import datetime
from typing import Any, Optional

from fastapi.encoders import jsonable_encoder
from fastapi.responses import JSONResponse


ERR_CODE_INTERNAL_ERROR = "INTERNAL_ERROR"
ERR_CODE_INVALID_REQUEST = "INVALID_REQUEST"
ERR_CODE_UNAUTHORIZED = "UNAUTHORIZED"
ERR_CODE_FORBIDDEN = "FORBIDDEN"
ERR_CODE_NOT_FOUND = "NOT_FOUND"
ERR_CODE_SERVICE_UNAVAILABLE = "SERVICE_UNAVAILABLE"


@dataclass
class APIError:
    code: str
    message: str
    details: Optional[dict[str, Any]] = None


@dataclass
class APIResponse:
    success: bool
    message: str
    data: Any
    error: Optional[APIError]
    timestamp: str
    request_id: Optional[str]


def ok(
    data: Any = None,
    message: str = "",
    request_id: Optional[str] = None,
    status_code: int = 200,
) -> JSONResponse:
    payload = APIResponse(
        success=True,
        message=message,
        data=data,
        error=None,
        timestamp=datetime.utcnow().isoformat() + "Z",
        request_id=request_id,
    )
    return JSONResponse(status_code=status_code, content=jsonable_encoder(payload))


def fail(
    code: str,
    message: str,
    details: Optional[dict[str, Any]] = None,
    request_id: Optional[str] = None,
    status_code: int = 400,
) -> JSONResponse:
    payload = APIResponse(
        success=False,
        message=message,
        data=None,
        error=APIError(code=code, message=message, details=details),
        timestamp=datetime.utcnow().isoformat() + "Z",
        request_id=request_id,
    )
    return JSONResponse(status_code=status_code, content=jsonable_encoder(payload))
