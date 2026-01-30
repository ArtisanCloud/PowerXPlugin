from sqlalchemy import BigInteger, Column, DateTime, String, text
from sqlalchemy.dialects.postgresql import JSONB

from app.entity.models.base import BaseModel


class IAMAuditLog(BaseModel):
    __tablename__ = "iam_audit_logs"

    actor_member_id = Column(BigInteger, nullable=True, index=True)
    action = Column(String(128), nullable=False, index=True)
    resource = Column(String(128), nullable=False, index=True)
    diff = Column(JSONB, nullable=True)
    created_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=text("now()"), nullable=False, onupdate=text("now()"))
