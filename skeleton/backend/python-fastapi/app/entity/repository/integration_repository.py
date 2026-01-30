from app.entity.models import (
    IntegrationGrantMatrixOverride,
    IntegrationSecret,
    IntegrationWebhookAttempt,
    IntegrationWebhookSubscription,
)
from app.entity.repository.base import BaseRepository


class IntegrationRepository(BaseRepository):
    def list_subscriptions(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(IntegrationWebhookSubscription.tenant_uuid == tenant_uuid)
        return self.list(IntegrationWebhookSubscription, filters)

    def get_subscription(self, subscription_id: str):
        return self.get_by_id(IntegrationWebhookSubscription, subscription_id)

    def list_attempts(self, subscription_id: str | None = None):
        filters = []
        if subscription_id:
            filters.append(IntegrationWebhookAttempt.subscription_id == subscription_id)
        return self.list(IntegrationWebhookAttempt, filters)

    def get_attempt(self, attempt_id: str):
        return self.get_by_id(IntegrationWebhookAttempt, attempt_id)

    def list_secrets(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(IntegrationSecret.tenant_uuid == tenant_uuid)
        return self.list(IntegrationSecret, filters)

    def get_secret(self, secret_id: str):
        return self.get_by_id(IntegrationSecret, secret_id)

    def list_grant_matrix_overrides(self):
        return self.list(IntegrationGrantMatrixOverride)

    def get_grant_matrix_override(self, override_id: str):
        return self.get_by_id(IntegrationGrantMatrixOverride, override_id)

    def create(self, entity):
        return self.add(entity)
