from datetime import datetime

from sqlalchemy import Column, DateTime, String

from app.entity.models.base import BaseModel


class Template(BaseModel):
    __tablename__ = "template"

    name = Column(String, nullable=False)
    description = Column(String, nullable=True)
    content = Column(String, nullable=False)
    status = Column(String, nullable=False, default="draft")
    review_status = Column(String, nullable=False, default="pending")
    review_comment = Column(String, nullable=True)
    reviewed_by = Column(String, nullable=True)
    reviewed_at = Column(DateTime, nullable=True)
    publish_channel = Column(String, nullable=True)
    published_at = Column(DateTime, nullable=True)
    cleanup_reason = Column(String, nullable=True)
    cleaned_at = Column(DateTime, nullable=True)
