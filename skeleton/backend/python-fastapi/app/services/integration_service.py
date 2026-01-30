from app.entity.repository.integration_repository import IntegrationRepository
from app.services._utils import to_list


class IntegrationService:
    def __init__(self, repo: IntegrationRepository | None = None) -> None:
        self._repo = repo or IntegrationRepository()

    def list_subscriptions(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_subscriptions(tenant_uuid))

    def list_attempts(self, subscription_id: str | None = None) -> list:
        return to_list(self._repo.list_attempts(subscription_id))

    def list_secrets(self, tenant_uuid: str | None = None) -> list:
        return to_list(self._repo.list_secrets(tenant_uuid))

    def list_grant_matrix_overrides(self) -> list:
        return to_list(self._repo.list_grant_matrix_overrides())
