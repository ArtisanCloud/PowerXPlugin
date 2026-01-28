from sqlalchemy import BigInteger, Column, DateTime, JSON, String

from app.entity.models.base import BaseModel


class Member(BaseModel):
    __tablename__ = "iam_members"

    user_id = Column(BigInteger, nullable=False)
    username = Column(String, nullable=False)
    display_name = Column(String, nullable=True)
    avatar_url = Column(String, nullable=True)
    status = Column(String, nullable=False, default="active")
    department_id = Column(BigInteger, nullable=True)
    meta = Column(JSON, nullable=True)
    last_login_at = Column(DateTime, nullable=True)
