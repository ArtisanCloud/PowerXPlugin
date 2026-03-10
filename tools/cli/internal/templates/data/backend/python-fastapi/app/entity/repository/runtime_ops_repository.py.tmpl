from sqlalchemy import select

from app.entity.models import RuntimeSession
from app.entity.repository.db import get_db
from app.entity.repository.base import BaseRepository


class RuntimeOpsRepository(BaseRepository):
    def list_sessions(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(RuntimeSession.tenant_uuid == tenant_uuid)
        return self.list(RuntimeSession, filters)

    def get_session(self, session_id: str):
        return self.get_by_id(RuntimeSession, session_id)

    def create(self, entity: RuntimeSession):
        return self.add(entity)

    def update_state(self, session_id: str, state: str | None, capabilities_hash: str | None = None):
        if not state and not capabilities_hash:
            return None
        db = get_db().session()
        try:
            row = db.execute(select(RuntimeSession).where(RuntimeSession.id == session_id)).scalar_one_or_none()
            if not row:
                return None
            if state:
                row.state = state
            if capabilities_hash is not None:
                row.capabilities_hash = capabilities_hash
            db.commit()
            return row
        finally:
            db.close()
