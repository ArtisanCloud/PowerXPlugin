from dataclasses import dataclass
from datetime import datetime
from typing import Any, Optional


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


def ok(data: Any = None, message: str = "") -> APIResponse:
    return APIResponse(
        success=True,
        message=message,
        data=data,
        error=None,
        timestamp=datetime.utcnow().isoformat() + "Z",
        request_id=None,
    )


def fail(code: str, message: str, details: Optional[dict[str, Any]] = None) -> APIResponse:
    return APIResponse(
        success=False,
        message=message,
        data=None,
        error=APIError(code=code, message=message, details=details),
        timestamp=datetime.utcnow().isoformat() + "Z",
        request_id=None,
    )
