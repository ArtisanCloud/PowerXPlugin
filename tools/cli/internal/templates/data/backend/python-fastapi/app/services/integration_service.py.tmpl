from __future__ import annotations

from datetime import datetime
from uuid import uuid4

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


class IntegrationService:
    def __init__(self, repo: IntegrationRepository | None = None) -> None:
        self._repo = repo or IntegrationRepository()

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

    def invoke_capability(self, payload: dict) -> dict:
        return {"status": "ok", "payload": {}, "metadata": {}}
