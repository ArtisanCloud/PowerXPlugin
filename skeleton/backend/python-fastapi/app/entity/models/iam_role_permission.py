from datetime import datetime

from sqlalchemy import BigInteger, Column, DateTime, String

from app.entity.models.base import Base


class IAMRolePermission(Base):
    __tablename__ = "iam_role_permissions"

    role_id = Column(BigInteger, primary_key=True)
    permission_id = Column(BigInteger, primary_key=True)
    tenant_uuid = Column(String, nullable=False)
    policy_version = Column(String, nullable=False, default="v1")
    created_at = Column(DateTime, default=datetime.utcnow)
