from datetime import datetime

from sqlalchemy import BigInteger, Boolean, Column, DateTime, JSON, String

from app.entity.models.base import Base


class User(Base):
    __tablename__ = "iam_users"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    email = Column(String, unique=True, nullable=True)
    phone = Column(String, nullable=True)
    display_name = Column(String, nullable=True)
    avatar_url = Column(String, nullable=True)
    status = Column(String, nullable=False, default="active")
    is_root = Column(Boolean, nullable=False, default=False)
    password_hash = Column(String, nullable=False, default="")
    meta = Column(JSON, nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow)
    deleted_at = Column(DateTime, nullable=True)
