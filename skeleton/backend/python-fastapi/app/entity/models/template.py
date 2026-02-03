from sqlalchemy import Column, DateTime, String, Text, text

from app.entity.models.base import BaseModel


class Template(BaseModel):
    __tablename__ = "template"

    name = Column(String(255), nullable=False)
    description = Column(Text, nullable=True)
    content = Column(Text, nullable=False)
    status = Column(String(50), nullable=False, server_default=text("'draft'"))
    review_status = Column(String(50), nullable=False, server_default=text("'pending'"))
    review_comment = Column(Text, nullable=True)
    reviewed_by = Column(String(100), nullable=True)
    reviewed_at = Column(DateTime(timezone=True), nullable=True)
    publish_channel = Column(String(120), nullable=True)
    published_at = Column(DateTime(timezone=True), nullable=True)
    cleanup_reason = Column(String(255), nullable=True)
    cleaned_at = Column(DateTime(timezone=True), nullable=True)
