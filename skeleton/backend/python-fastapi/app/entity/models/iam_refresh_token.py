from datetime import datetime

from sqlalchemy import BigInteger, Boolean, Column, DateTime, String

from app.entity.models.base import Base


class IAMRefreshToken(Base):
    __tablename__ = "iam_refresh_tokens"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    token_hash = Column(String, nullable=False, unique=True)
    user_id = Column(BigInteger, nullable=False)
    tenant_uuid = Column(String, nullable=False)
    member_id = Column(BigInteger, nullable=False)
    expires_at = Column(DateTime, nullable=False)
    revoked = Column(Boolean, nullable=False, default=False)
    created_at = Column(DateTime, default=datetime.utcnow)
