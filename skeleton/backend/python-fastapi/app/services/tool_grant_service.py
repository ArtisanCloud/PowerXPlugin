from __future__ import annotations

from datetime import datetime, timezone

import jwt

from app.config.settings import get_settings
from app.entity.models import ToolGrantRevocation, ToolGrantUsageEvent
from app.entity.repository.tool_grant_repository import ToolGrantRepository
from app.services._utils import to_dict, to_list


def _now() -> datetime:
    return datetime.utcnow()


class ToolGrantService:
    def __init__(self, repo: ToolGrantRepository | None = None) -> None:
        self._repo = repo or ToolGrantRepository()

    def list_revocations(self, tenant_uuid: str | None = None, limit: int = 0) -> list:
        return to_list(self._repo.list_revocations(tenant_uuid, limit))

    def list_usage_events(self, tenant_uuid: str | None = None, toolgrant_id: str | None = None, limit: int = 0) -> list:
        return to_list(self._repo.list_usage_events(tenant_uuid, toolgrant_id, limit))

    def revoke(self, payload: dict) -> dict:
        tenant_uuid = payload.get("tenant_uuid") or ""
        toolgrant_id = payload.get("toolgrant_id") or ""
        reason = payload.get("reason") or ""
        actor = payload.get("requested_by") or payload.get("revoked_by") or "admin"
        ttl_expiry = payload.get("ttl_expiry") or _now()
        entity = ToolGrantRevocation(
            id=payload.get("id"),
            tenant_uuid=tenant_uuid,
            toolgrant_id=toolgrant_id,
            revoked_at=payload.get("revoked_at") or _now(),
            revoked_by=actor,
            reason=reason or None,
            ttl_expiry=ttl_expiry,
            created_at=_now(),
        )
        entity = self._repo.create_revocation(entity)
        usage = ToolGrantUsageEvent(
            tenant_uuid=tenant_uuid,
            toolgrant_id=toolgrant_id,
            event_type="REVOKED",
            capability="",
            agent_id=actor,
            occurred_at=_now(),
            metadata_={"reason": reason},
        )
        self._repo.create_usage_event(usage)
        return to_dict(entity)

    def validate(self, tenant_uuid: str, token: str) -> dict:
        if not token:
            raise ValueError("token missing")
        settings = get_settings()
        secret = (settings.security_toolgrant_secret or "").strip()
        if not secret:
            if settings.dev_mode or settings.server_dev_mode:
                secret = "dev-toolgrant-secret"
            else:
                raise ValueError("toolgrant signing key not configured")
        try:
            payload = jwt.decode(
                token,
                secret,
                algorithms=["HS256"],
                options={"require": ["exp", "iat"], "verify_aud": False},
            )
        except jwt.PyJWTError as exc:
            raise ValueError(str(exc)) from exc
        claim_tenant = payload.get("tenant_uuid") or ""
        if tenant_uuid and claim_tenant and claim_tenant != tenant_uuid:
            raise ValueError("tenant mismatch")
        jti = payload.get("jti") or payload.get("id") or ""
        if jti:
            revocations = self._repo.list_revocations(claim_tenant or tenant_uuid)
            for rev in revocations:
                if rev.toolgrant_id == jti:
                    raise ValueError("toolgrant revoked")
        exp = payload.get("exp")
        if exp:
            exp_dt = datetime.fromtimestamp(exp, tz=timezone.utc)
            if exp_dt <= datetime.now(tz=timezone.utc):
                raise ValueError("token expired")
        return payload
