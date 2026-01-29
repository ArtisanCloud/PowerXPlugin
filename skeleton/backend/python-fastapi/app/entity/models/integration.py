from app.entity.models.base import BaseModel


class IntegrationWebhookSubscription(BaseModel):
    __tablename__ = "integration_webhook_subscriptions"


class IntegrationWebhookAttempt(BaseModel):
    __tablename__ = "integration_webhook_attempts"


class IntegrationSecret(BaseModel):
    __tablename__ = "integration_secrets"


class IntegrationGrantMatrixOverride(BaseModel):
    __tablename__ = "integration_grant_matrix_overrides"
