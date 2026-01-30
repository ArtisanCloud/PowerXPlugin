from sqlalchemy import Boolean, Column, String, text
from sqlalchemy.dialects.postgresql import JSONB, UUID

from app.entity.models.base import BaseModel


class CustomerAccount(BaseModel):
    __tablename__ = "customer_accounts"

    customer_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    email = Column(String, nullable=True, index=True)
    phone = Column(String, nullable=True, index=True)
    password_hash = Column(String, nullable=True)
    status = Column(String, server_default=text("'active'"), nullable=False)
    metadata_ = Column("metadata", JSONB, server_default=text("'{}'::jsonb"), nullable=True)
    email_verified = Column(Boolean, server_default=text("false"), nullable=False)
    phone_verified = Column(Boolean, server_default=text("false"), nullable=False)
