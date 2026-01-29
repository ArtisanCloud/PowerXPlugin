from datetime import datetime

from sqlalchemy import BigInteger, Column, DateTime

from app.entity.models.base import Base


class IAMMemberRole(Base):
    __tablename__ = "iam_member_roles"

    member_id = Column(BigInteger, primary_key=True)
    role_id = Column(BigInteger, primary_key=True)
    created_at = Column(DateTime, default=datetime.utcnow)
