from datetime import datetime

from sqlalchemy import BigInteger, Column, DateTime, String

from app.entity.models.base import Base


class Tenant(Base):
    __tablename__ = "iam_tenants"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    uuid = Column(String, unique=True, nullable=False)
    key = Column(String, unique=True, nullable=False)
    name = Column(String, nullable=False)
    status = Column(String, nullable=False, default="active")
    plan = Column(String, nullable=False, default="free")
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow)
    deleted_at = Column(DateTime, nullable=True)
