"""init

Revision ID: 0001_init
Revises: 
Create Date: 2026-01-24
"""

from alembic import op

revision = "0001_init"
down_revision = None
branch_labels = None
depends_on = None


def upgrade():
    from app.entity.models import Base

    bind = op.get_bind()
    Base.metadata.create_all(bind=bind)


def downgrade():
    from app.entity.models import Base

    bind = op.get_bind()
    Base.metadata.drop_all(bind=bind)
