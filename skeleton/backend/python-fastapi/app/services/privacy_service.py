from app.entity.repository.privacy_repository import PrivacyRepository
from app.services._utils import to_list


class PrivacyService:
    def __init__(self, repo: PrivacyRepository | None = None) -> None:
        self._repo = repo or PrivacyRepository()

    def list_classifications(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_classifications(tenant_uuid))

    def list_consent_tokens(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_consent_tokens(tenant_uuid))

    def list_lifecycle_events(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_lifecycle_events(tenant_uuid))
