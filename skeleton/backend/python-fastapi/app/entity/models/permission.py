from datetime import datetime

from sqlalchemy import BigInteger, Column, DateTime, String

from app.entity.models.base import Base


class Permission(Base):
    __tablename__ = "iam_permissions"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    resource = Column(String, nullable=False)
    action = Column(String, nullable=False)
    description = Column(String, nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow)
