from __future__ import annotations

from datetime import datetime
import json
import logging
import os
from urllib import parse as urlparse
from urllib import request as urlrequest
from urllib.error import HTTPError, URLError
from uuid import uuid4

from app.config.settings import get_settings
from app.entity.models import (
    IntegrationChangeApproval,
    IntegrationGrantMatrixOverride,
    IntegrationIdempotencyRecord,
    IntegrationSecret,
    IntegrationWebhookAttempt,
    IntegrationWebhookSubscription,
)
from app.entity.repository.integration_repository import IntegrationRepository
from app.services._utils import to_dict, to_list


def _now() -> datetime:
    return datetime.utcnow()


def _normalize_api_prefix(raw: str | None) -> str:
    value = str(raw or "").strip()
    if not value:
        return "/api/v1"
    if not value.startswith("/"):
        value = "/" + value
    value = "/" + value.strip("/")
    if value == "/":
        return "/api/v1"
    return value


def _auth_scheme_from_header(value: str) -> str:
    raw = (value or "").strip().lower()
    if raw.startswith("bearer "):
        return "bearer"
    if raw.startswith("apikey ") or raw.startswith("api-key ") or raw.startswith("api_key "):
        return "apikey"
    if raw:
        return "unknown"
    return "none"


def _first_nonempty(*items: str) -> str:
    for item in items:
        value = str(item or "").strip()
        if value:
            return value
    return ""


class IntegrationInvokeError(Exception):
    def __init__(
        self,
        message: str,
        *,
        status_code: int = 502,
        trace_id: str = "",
        details: dict | None = None,
        warnings: list[str] | None = None,
    ) -> None:
        super().__init__(message)
        self.status_code = status_code
        self.trace_id = trace_id
        self.details = details or {}
        self.warnings = warnings or []


class IntegrationService:
    def __init__(self, repo: IntegrationRepository | None = None) -> None:
        self._repo = repo or IntegrationRepository()
        self._settings = get_settings()
        self._logger = logging.getLogger("integration_http")

    def list_subscriptions(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_subscriptions(tenant_uuid))

    def create_subscription(self, payload: dict) -> dict:
        entity = IntegrationWebhookSubscription(
            id=payload.get("id"),
            tenant_uuid=payload.get("tenant_uuid") or payload.get("tenantUuid") or "tenant-demo",
            event_type=payload.get("event_type") or "",
            target_url=payload.get("target_url") or "",
            secret=payload.get("secret"),
            retry_policy=payload.get("retry_policy"),
            status=payload.get("status") or "ACTIVE",
            metadata_=payload.get("metadata") or {},
            created_at=_now(),
            updated_at=_now(),
        )
        entity = self._repo.create_subscription(entity)
        return to_dict(entity)

    def update_subscription(self, subscription_id: str, payload: dict) -> dict:
        updates = {}
        for key in ("event_type", "target_url", "secret", "retry_policy", "status", "metadata"):
            if key in payload:
                updates["metadata_" if key == "metadata" else key] = payload.get(key)
        updates["updated_at"] = _now()
        entity = self._repo.update_subscription(subscription_id, updates)
        return to_dict(entity)

    def delete_subscription(self, subscription_id: str) -> dict:
        self._repo.delete_subscription(subscription_id)
        return {"ok": True, "id": subscription_id}

    def list_attempts(self, subscription_id: str | None = None) -> list:
        return to_list(self._repo.list_attempts(subscription_id))

    def replay_attempt(self, attempt_id: str) -> dict:
        attempt = self._repo.get_attempt(attempt_id)
        return {"ok": True, "attempt_id": attempt_id, "subscription_id": getattr(attempt, "subscription_id", None)}

    def get_attempt(self, attempt_id: str) -> dict:
        attempt = self._repo.get_attempt(attempt_id)
        return to_dict(attempt)

    def list_secrets(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_secrets(tenant_uuid))

    def create_secret(self, payload: dict) -> dict:
        entity = IntegrationSecret(
            id=payload.get("id"),
            tenant_uuid=payload.get("tenant_uuid") or payload.get("tenantUuid") or "tenant-demo",
            integration_type=payload.get("integration_type") or "",
            current_secret_ref=payload.get("current_secret_ref"),
            pending_secret_ref=payload.get("pending_secret_ref"),
            rotation_interval_days=payload.get("rotation_interval_days") or 30,
            last_rotated_at=payload.get("last_rotated_at"),
            next_rotation_due_at=payload.get("next_rotation_due_at"),
            status=payload.get("status") or "ACTIVE",
            audit_log=payload.get("audit_log") or [],
            metadata_=payload.get("metadata") or {},
            created_at=_now(),
            updated_at=_now(),
        )
        entity = self._repo.create_secret(entity)
        return to_dict(entity)

    def rotate_secret(self, secret_id: str, payload: dict | None = None) -> dict:
        updates = {
            "status": "ROTATING",
            "pending_secret_ref": (payload or {}).get("pending_secret_ref"),
            "updated_at": _now(),
        }
        entity = self._repo.update_secret(secret_id, updates)
        return to_dict(entity)

    def complete_rotation(self, secret_id: str, payload: dict | None = None) -> dict:
        updates = {
            "status": "ACTIVE",
            "current_secret_ref": (payload or {}).get("current_secret_ref"),
            "pending_secret_ref": None,
            "last_rotated_at": _now(),
            "updated_at": _now(),
        }
        entity = self._repo.update_secret(secret_id, updates)
        return to_dict(entity)

    def revoke_secret(self, secret_id: str, payload: dict | None = None) -> dict:
        updates = {"status": "REVOKED", "updated_at": _now()}
        entity = self._repo.update_secret(secret_id, updates)
        return to_dict(entity)

    def get_secret_audit(self, secret_id: str) -> dict:
        entity = self._repo.get_secret(secret_id)
        audit = getattr(entity, "audit_log", None) if entity else None
        return {"id": secret_id, "items": audit or []}

    def list_grant_matrix_overrides(self) -> list:
        return to_list(self._repo.list_grant_matrix_overrides())

    def submit_grant_matrix(self, payload: dict) -> dict:
        entity = IntegrationGrantMatrixOverride(
            id=payload.get("id") or uuid4().hex,
            scope=payload.get("scope") or "",
            channel=payload.get("channel") or "",
            resource=payload.get("resource") or "",
            action=payload.get("action") or "",
            constraints=payload.get("constraints") or {},
            status=payload.get("status") or "PENDING",
            version=payload.get("version") or 1,
            approved_by=payload.get("approved_by"),
            approved_at=payload.get("approved_at"),
            created_at=_now(),
            updated_at=_now(),
        )
        entity = self._repo.create_grant_matrix_override(entity)
        return to_dict(entity)

    def list_approvals(self) -> list:
        return to_list(self._repo.list_approvals())

    def approve(self, approval_id: str, payload: dict | None = None) -> dict:
        updates = {"status": "APPROVED", "reviewed_by": (payload or {}).get("reviewed_by"), "reviewed_at": _now()}
        entity = self._repo.update_approval(approval_id, updates)
        return to_dict(entity)

    def reject(self, approval_id: str, payload: dict | None = None) -> dict:
        updates = {"status": "REJECTED", "reviewed_by": (payload or {}).get("reviewed_by"), "reviewed_at": _now()}
        entity = self._repo.update_approval(approval_id, updates)
        return to_dict(entity)

    def dispatch(self, payload: dict) -> dict:
        record = IntegrationIdempotencyRecord(
            key=payload.get("idempotency_key") or uuid4().hex,
            tenant_uuid=payload.get("tenant_uuid") or "tenant-demo",
            scope=payload.get("tool_scope"),
            operation="dispatch",
            payload_hash=None,
            response_data={},
            metadata_=payload.get("metadata") or {},
            expires_at=None,
            created_at=_now(),
        )
        self._repo.create_idempotency_record(record)
        return {"status": "ok"}

    def invoke_capability(
        self,
        payload: dict,
        *,
        forward_headers: dict[str, str] | None = None,
        request_id: str | None = None,
    ) -> dict:
        capability_id = str(payload.get("capabilityId") or "").strip()
        action = str(payload.get("action") or "").strip()
        preferred_protocol = str(payload.get("preferredProtocol") or "").strip()
        invoke_payload = payload.get("payload") if isinstance(payload.get("payload"), dict) else {}

        if not capability_id:
            raise IntegrationInvokeError("capabilityId is required", status_code=400)

        gateway_base = self._effective_gateway_base_url()
        if not gateway_base:
            raise IntegrationInvokeError("capability gateway unavailable", status_code=503)

        auth_scheme = self._resolve_gateway_auth_scheme()
        gateway_credential = self._resolve_gateway_credential(auth_scheme)
        if not gateway_credential:
            raise IntegrationInvokeError(f"gateway credential missing (auth_scheme={auth_scheme})", status_code=503)

        endpoint = f"{gateway_base}/tenant/invocations"
        body = {
            "capability_id": capability_id,
            "payload": invoke_payload,
        }
        if action:
            body["action"] = action
        if preferred_protocol:
            body["preferred_protocol"] = preferred_protocol
        body_bytes = json.dumps(body).encode("utf-8")
        req = urlrequest.Request(endpoint, data=body_bytes, method="POST")
        req.add_header("Content-Type", "application/json")
        req.add_header("Accept", "application/json")
        req.add_header("Authorization", self._build_auth_header(auth_scheme, gateway_credential))
        rid = str(request_id or "").strip() or uuid4().hex
        req.add_header("X-Request-ID", rid)

        for key, value in (forward_headers or {}).items():
            if str(value).strip():
                req.add_header(key, str(value).strip())

        self._logger.info(
            "capability invoke dispatch capability_id=%s action=%s preferred_protocol=%s endpoint=%s auth_scheme=%s forwarded_auth=%s",
            capability_id,
            action,
            preferred_protocol,
            endpoint,
            auth_scheme,
            "Authorization" in (forward_headers or {}),
        )

        try:
            with urlrequest.urlopen(req, timeout=10) as resp:
                status_code = resp.status
                raw_bytes = resp.read()
                trace_id = str(resp.headers.get("X-Trace-Id") or "").strip()
        except HTTPError as exc:
            raw_bytes = exc.read() if hasattr(exc, "read") else b""
            details = self._safe_json(raw_bytes)
            trace_id = str(exc.headers.get("X-Trace-Id") if getattr(exc, "headers", None) else "") or ""
            self._logger.error(
                "capability invoke upstream http error status=%s trace_id=%s body=%s",
                exc.code,
                trace_id,
                details,
            )
            raise IntegrationInvokeError(
                "capability invoke failed",
                status_code=exc.code or 502,
                trace_id=trace_id.strip(),
                details=details if isinstance(details, dict) else {"raw": details},
            ) from exc
        except URLError as exc:
            self._logger.error("capability invoke upstream transport error: %s", exc)
            raise IntegrationInvokeError("capability invoke failed", status_code=502, details={"error": str(exc)}) from exc

        envelope = self._safe_json(raw_bytes)
        if not isinstance(envelope, dict):
            envelope = {"raw": envelope}

        response_data = envelope.get("data") if isinstance(envelope.get("data"), dict) else {}
        if not trace_id:
            trace_id = str(envelope.get("trace_id") or response_data.get("trace_id") or "").strip()
        status = str(response_data.get("status") or envelope.get("status") or "").strip() or "ok"
        warnings = response_data.get("warnings") if isinstance(response_data.get("warnings"), list) else []
        payload_data = (
            response_data.get("payload")
            if response_data.get("payload") is not None
            else response_data
        )
        result = {
            "traceId": trace_id or None,
            "status": status,
            "data": payload_data,
            "raw": envelope,
        }
        if warnings:
            result["warnings"] = warnings

        self._logger.info(
            "capability invoke success status_code=%s trace_id=%s status=%s",
            status_code,
            trace_id,
            status,
        )
        return result

    def _effective_gateway_base_url(self) -> str:
        base_url = str(getattr(self._settings, "gateway_base_url", "") or "").strip().rstrip("/")
        if not base_url:
            return ""
        api_prefix = _normalize_api_prefix(getattr(self._settings, "gateway_api_prefix", "/api/v1"))
        if base_url.endswith(api_prefix):
            return base_url
        return f"{base_url}{api_prefix}"

    def _resolve_gateway_auth_scheme(self) -> str:
        raw = os.getenv("PX_GATEWAY_AUTH_SCHEME", "").strip().lower()
        if raw in {"apikey", "api_key", "api-key"}:
            return "apikey"
        if raw == "bearer":
            return "bearer"
        api_key = _first_nonempty(os.getenv("PX_GATEWAY_API_KEY", ""), os.getenv("PX_PLUGIN_API_KEY", ""))
        sts_client = _first_nonempty(os.getenv("POWERX_STS_CLIENT_ID", ""), os.getenv("PX_STS_CLIENT_ID", ""))
        if api_key and not sts_client:
            return "apikey"
        return "bearer"

    def _resolve_gateway_credential(self, auth_scheme: str) -> str:
        if auth_scheme == "apikey":
            return _first_nonempty(os.getenv("PX_GATEWAY_API_KEY", ""), os.getenv("PX_PLUGIN_API_KEY", ""))
        raise RuntimeError("bearer gateway mode requires STS token provider; static bearer credentials are not supported")

    def _build_auth_header(self, auth_scheme: str, credential: str) -> str:
        if auth_scheme == "apikey":
            return f"ApiKey {credential.strip()}"
        return f"Bearer {credential.strip()}"

    def _safe_json(self, raw_bytes: bytes):
        if not raw_bytes:
            return {}
        try:
            return json.loads(raw_bytes.decode("utf-8"))
        except Exception:
            return raw_bytes.decode("utf-8", errors="ignore")
