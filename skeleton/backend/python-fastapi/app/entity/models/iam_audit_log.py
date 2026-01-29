from datetime import datetime

from sqlalchemy import BigInteger, Column, DateTime, JSON, String

from app.entity.models.base import BaseModel


class IAMAuditLog(BaseModel):
    __tablename__ = "iam_audit_logs"

    actor_member_id = Column(BigInteger, nullable=True)
    action = Column(String, nullable=False)
    resource = Column(String, nullable=False)
    diff = Column(JSON, nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow)
