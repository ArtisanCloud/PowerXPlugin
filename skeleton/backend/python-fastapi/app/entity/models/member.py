from sqlalchemy import BigInteger, Column, DateTime, String, text
from sqlalchemy.dialects.postgresql import JSONB

from app.entity.models.base import BaseModel


class Member(BaseModel):
    __tablename__ = "iam_members"

    user_id = Column(BigInteger, nullable=False, index=True)
    username = Column(String(64), nullable=False, index=True)
    display_name = Column(String(128), nullable=True)
    avatar_url = Column(String(255), nullable=True)
    status = Column(String(32), nullable=False, server_default=text("'active'"), index=True)
    department_id = Column(BigInteger, nullable=True, index=True)
    meta = Column(JSONB, nullable=True)
    last_login_at = Column(DateTime(timezone=True), nullable=True, index=True)
