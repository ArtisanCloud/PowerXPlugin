from sqlalchemy import BigInteger, Boolean, Column, DateTime, String, text
from sqlalchemy.dialects.postgresql import UUID

from app.entity.models.base import Base


class IAMRefreshToken(Base):
    __tablename__ = "iam_refresh_tokens"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    token_hash = Column(String(128), nullable=False, unique=True)
    user_id = Column(BigInteger, nullable=False, index=True)
    tenant_uuid = Column(UUID(as_uuid=False), nullable=False, index=True)
    member_id = Column(BigInteger, nullable=False)
    expires_at = Column(DateTime(timezone=True), nullable=False, index=True)
    revoked = Column(Boolean, nullable=False, server_default=text("false"))
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
