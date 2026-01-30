from app.entity.models import ToolGrantRevocation, ToolGrantUsageEvent
from app.entity.repository.base import BaseRepository


class ToolGrantRepository(BaseRepository):
    def list_revocations(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(ToolGrantRevocation.tenant_uuid == tenant_uuid)
        return self.list(ToolGrantRevocation, filters)

    def get_revocation(self, revocation_id: str):
        return self.get_by_id(ToolGrantRevocation, revocation_id)

    def list_usage_events(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(ToolGrantUsageEvent.tenant_uuid == tenant_uuid)
        return self.list(ToolGrantUsageEvent, filters)

    def get_usage_event(self, event_id: str):
        return self.get_by_id(ToolGrantUsageEvent, event_id)

    def create(self, entity):
        return self.add(entity)
