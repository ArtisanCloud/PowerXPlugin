from sqlalchemy import BigInteger, Boolean, Column, DateTime, String, func, text
from sqlalchemy.dialects.postgresql import JSONB

from app.entity.models.base import Base


class User(Base):
    __tablename__ = "iam_users"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    email = Column(String(255), unique=True, nullable=True)
    phone = Column(String(32), nullable=True, index=True)
    display_name = Column(String(128), nullable=True)
    avatar_url = Column(String(255), nullable=True)
    status = Column(String(32), nullable=False, server_default=text("'active'"))
    is_root = Column(Boolean, nullable=False, server_default=text("false"), index=True)
    password_hash = Column(String(255), nullable=False, server_default=text("''"))
    meta = Column(JSONB, nullable=True)
    created_at = Column(DateTime(timezone=True), default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), default=func.now(), onupdate=func.now(), nullable=False)
    deleted_at = Column(DateTime(timezone=True), nullable=True)
