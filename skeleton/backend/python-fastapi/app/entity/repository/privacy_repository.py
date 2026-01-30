from sqlalchemy import select

from app.entity.models import PrivacyConsentToken, PrivacyDataClassification, PrivacyLifecycleEvent
from app.entity.repository.base import BaseRepository


class PrivacyRepository(BaseRepository):
    def list_classifications(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(PrivacyDataClassification.tenant_uuid == tenant_uuid)
        return self.list(PrivacyDataClassification, filters)

    def get_classification(self, classification_id: str):
        return self.get_by_id(PrivacyDataClassification, classification_id)

    def list_consent_tokens(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(PrivacyConsentToken.tenant_uuid == tenant_uuid)
        return self.list(PrivacyConsentToken, filters)

    def get_consent_token(self, token_id: str):
        return self.get_by_id(PrivacyConsentToken, token_id)

    def list_lifecycle_events(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(PrivacyLifecycleEvent.tenant_uuid == tenant_uuid)
        return self.list(PrivacyLifecycleEvent, filters)

    def get_lifecycle_event(self, event_id: str):
        return self.get_by_id(PrivacyLifecycleEvent, event_id)

    def create(self, entity):
        return self.add(entity)
