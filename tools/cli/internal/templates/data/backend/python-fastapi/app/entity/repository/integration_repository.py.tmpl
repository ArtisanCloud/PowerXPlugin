from app.entity.models import (
    IntegrationChangeApproval,
    IntegrationGrantMatrixOverride,
    IntegrationIdempotencyRecord,
    IntegrationSecret,
    IntegrationWebhookAttempt,
    IntegrationWebhookSubscription,
)
from app.entity.repository.base import BaseRepository


class IntegrationRepository(BaseRepository):
    def list_approvals(self):
        return self.list(IntegrationChangeApproval)

    def get_approval(self, approval_id: str):
        return self.get_by_id(IntegrationChangeApproval, approval_id)

    def update_approval(self, approval_id: str, updates: dict):
        return self.update_by_id(IntegrationChangeApproval, approval_id, updates)

    def list_subscriptions(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(IntegrationWebhookSubscription.tenant_uuid == tenant_uuid)
        return self.list(IntegrationWebhookSubscription, filters)

    def get_subscription(self, subscription_id: str):
        return self.get_by_id(IntegrationWebhookSubscription, subscription_id)

    def create_subscription(self, entity: IntegrationWebhookSubscription):
        return self.add(entity)

    def update_subscription(self, subscription_id: str, updates: dict):
        return self.update_by_id(IntegrationWebhookSubscription, subscription_id, updates)

    def delete_subscription(self, subscription_id: str):
        return self.delete_by_id(IntegrationWebhookSubscription, subscription_id)

    def list_attempts(self, subscription_id: str | None = None):
        filters = []
        if subscription_id:
            filters.append(IntegrationWebhookAttempt.subscription_id == subscription_id)
        return self.list(IntegrationWebhookAttempt, filters)

    def get_attempt(self, attempt_id: str):
        return self.get_by_id(IntegrationWebhookAttempt, attempt_id)

    def create_attempt(self, entity: IntegrationWebhookAttempt):
        return self.add(entity)

    def list_secrets(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(IntegrationSecret.tenant_uuid == tenant_uuid)
        return self.list(IntegrationSecret, filters)

    def get_secret(self, secret_id: str):
        return self.get_by_id(IntegrationSecret, secret_id)

    def create_secret(self, entity: IntegrationSecret):
        return self.add(entity)

    def update_secret(self, secret_id: str, updates: dict):
        return self.update_by_id(IntegrationSecret, secret_id, updates)

    def list_grant_matrix_overrides(self):
        return self.list(IntegrationGrantMatrixOverride)

    def get_grant_matrix_override(self, override_id: str):
        return self.get_by_id(IntegrationGrantMatrixOverride, override_id)

    def create_grant_matrix_override(self, entity: IntegrationGrantMatrixOverride):
        return self.add(entity)

    def list_idempotency_records(self, tenant_uuid: str | None = None):
        filters = []
        if tenant_uuid:
            filters.append(IntegrationIdempotencyRecord.tenant_uuid == tenant_uuid)
        return self.list(IntegrationIdempotencyRecord, filters)

    def get_idempotency_record(self, key: str):
        return self.get_by_id(IntegrationIdempotencyRecord, key)

    def create_idempotency_record(self, entity: IntegrationIdempotencyRecord):
        return self.add(entity)
