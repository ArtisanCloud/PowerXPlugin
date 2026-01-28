from sqlalchemy import Column, String

from app.entity.models.base import BaseModel


class Role(BaseModel):
    __tablename__ = "iam_roles"

    code = Column(String, nullable=False)
    name = Column(String, nullable=False)
    description = Column(String, nullable=True)
    scope_type = Column(String, nullable=False, default="tenant")
    policy_version = Column(String, nullable=False, default="v1")
