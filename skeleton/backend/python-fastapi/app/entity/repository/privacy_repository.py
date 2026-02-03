from datetime import datetime

from sqlalchemy import or_, select, update

from app.entity.models import PrivacyConsentToken, PrivacyDataClassification, PrivacyLifecycleEvent
from app.entity.repository.base import BaseRepository


class PrivacyRepository(BaseRepository):
    def active_consent_tokens(self, tenant_uuid: str, now: datetime | None = None):
        if not now:
            now = datetime.utcnow()
        db = self._session()
        try:
            return (
                db.execute(
                    select(PrivacyConsentToken)
                    .where(PrivacyConsentToken.tenant_uuid == tenant_uuid)
                    .where(PrivacyConsentToken.status == "ACTIVE")
                    .where(
                        or_(
                            PrivacyConsentToken.expires_at.is_(None),
                            PrivacyConsentToken.expires_at > now,
                        )
                    )
                    .order_by(PrivacyConsentToken.issued_at.desc())
                )
                .scalars()
                .all()
            )
        finally:
            db.close()

    def list_classifications(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(PrivacyDataClassification.tenant_uuid == tenant_uuid)
        return self.list(PrivacyDataClassification, filters)

    def get_classification(self, classification_id: str):
        return self.get_by_id(PrivacyDataClassification, classification_id)

    def list_consent_tokens(
        self,
        tenant_uuid: str | None = None,
        statuses: list[str] | None = None,
    ):
        db = self._session()
        try:
            stmt = select(PrivacyConsentToken)
            if tenant_uuid:
                stmt = stmt.where(PrivacyConsentToken.tenant_uuid == tenant_uuid)
            if statuses:
                stmt = stmt.where(PrivacyConsentToken.status.in_(statuses))
            stmt = stmt.order_by(PrivacyConsentToken.issued_at.desc())
            return db.execute(stmt).scalars().all()
        finally:
            db.close()

    def get_consent_token(self, token_id: str):
        return self.get_by_id(PrivacyConsentToken, token_id)

    def update_consent_token(self, token_id: str, updates: dict):
        return self.update_by_id(PrivacyConsentToken, token_id, updates)

    def revoke_consent_token(self, tenant_uuid: str, token_id: str, updates: dict):
        db = self._session()
        try:
            db.execute(
                update(PrivacyConsentToken)
                .where(PrivacyConsentToken.id == token_id)
                .where(PrivacyConsentToken.tenant_uuid == tenant_uuid)
                .values(**updates)
            )
            db.commit()
            return (
                db.execute(
                    select(PrivacyConsentToken).where(
                        PrivacyConsentToken.id == token_id,
                        PrivacyConsentToken.tenant_uuid == tenant_uuid,
                    )
                )
                .scalars()
                .first()
            )
        finally:
            db.close()

    def list_lifecycle_events(
        self,
        tenant_uuid: str | None = None,
        event_types: list[str] | None = None,
        limit: int = 0,
    ):
        db = self._session()
        try:
            stmt = select(PrivacyLifecycleEvent)
            if tenant_uuid:
                stmt = stmt.where(PrivacyLifecycleEvent.tenant_uuid == tenant_uuid)
            if event_types:
                stmt = stmt.where(PrivacyLifecycleEvent.event_type.in_(event_types))
            stmt = stmt.order_by(PrivacyLifecycleEvent.occurred_at.desc())
            if limit and limit > 0:
                stmt = stmt.limit(limit)
            return db.execute(stmt).scalars().all()
        finally:
            db.close()

    def get_lifecycle_event(self, event_id: str):
        return self.get_by_id(PrivacyLifecycleEvent, event_id)

    def create_classification(self, entity: PrivacyDataClassification):
        return self.add(entity)

    def create_consent_token(self, entity: PrivacyConsentToken):
        return self.add(entity)

    def create_lifecycle_event(self, entity: PrivacyLifecycleEvent):
        return self.add(entity)
