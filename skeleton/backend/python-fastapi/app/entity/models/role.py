from sqlalchemy import Column, String, text

from app.entity.models.base import BaseModel


class Role(BaseModel):
    __tablename__ = "iam_roles"

    code = Column(String(64), nullable=False, index=True)
    name = Column(String(128), nullable=False)
    description = Column(String(255), nullable=True)
    scope_type = Column(String(32), nullable=False, server_default=text("'tenant'"), index=True)
    policy_version = Column(String(64), nullable=False, server_default=text("'v1'"))
