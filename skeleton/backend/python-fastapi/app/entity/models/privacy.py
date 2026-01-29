from app.entity.models.base import BaseModel


class PrivacyDataClassification(BaseModel):
    __tablename__ = "privacy_data_classifications"


class PrivacyConsentToken(BaseModel):
    __tablename__ = "privacy_consent_tokens"


class PrivacyLifecycleEvent(BaseModel):
    __tablename__ = "privacy_lifecycle_events"
