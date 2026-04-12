from sqlalchemy import select

from app.entity.models import ToolGrantRevocation, ToolGrantUsageEvent
from app.entity.repository.base import BaseRepository


class ToolGrantRepository(BaseRepository):
    def list_revocations(self, tenant_uuid: str | None = None, limit: int = 0):
        db = self._session()
        try:
            stmt = select(ToolGrantRevocation)
            if tenant_uuid:
                stmt = stmt.where(ToolGrantRevocation.tenant_uuid == tenant_uuid)
            stmt = stmt.order_by(ToolGrantRevocation.revoked_at.desc())
            if limit and limit > 0:
                stmt = stmt.limit(limit)
            return db.execute(stmt).scalars().all()
        finally:
            db.close()

    def get_revocation(self, revocation_id: str):
        return self.get_by_id(ToolGrantRevocation, revocation_id)

    def list_usage_events(self, tenant_uuid: str | None = None, toolgrant_id: str | None = None, limit: int = 0):
        db = self._session()
        try:
            stmt = select(ToolGrantUsageEvent)
            if tenant_uuid:
                stmt = stmt.where(ToolGrantUsageEvent.tenant_uuid == tenant_uuid)
            if toolgrant_id:
                stmt = stmt.where(ToolGrantUsageEvent.toolgrant_id == toolgrant_id)
            stmt = stmt.order_by(ToolGrantUsageEvent.occurred_at.desc())
            if limit and limit > 0:
                stmt = stmt.limit(limit)
            return db.execute(stmt).scalars().all()
        finally:
            db.close()

    def get_usage_event(self, event_id: str):
        return self.get_by_id(ToolGrantUsageEvent, event_id)

    def create_revocation(self, entity: ToolGrantRevocation):
        return self.add(entity)

    def create_usage_event(self, entity: ToolGrantUsageEvent):
        return self.add(entity)
