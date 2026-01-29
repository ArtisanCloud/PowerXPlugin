from logging.config import fileConfig
import os
import sys

from alembic import context
from sqlalchemy import engine_from_config, pool

config = context.config

if config.config_file_name is not None:
    fileConfig(config.config_file_name)

# allow loading app settings from project root
PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
if PROJECT_ROOT not in sys.path:
    sys.path.insert(0, PROJECT_ROOT)

try:
    from app.config.settings import get_settings
except Exception:  # pragma: no cover - fallback if settings import fails
    get_settings = None


def _mask_dsn(dsn: str) -> str:
    if not dsn:
        return dsn
    if "@" not in dsn or "://" not in dsn:
        return dsn
    scheme, rest = dsn.split("://", 1)
    if "@" not in rest:
        return dsn
    creds, tail = rest.split("@", 1)
    if ":" in creds:
        user, _ = creds.split(":", 1)
        creds = f"{user}:***"
    else:
        creds = "***"
    return f"{scheme}://{creds}@{tail}"


def _apply_runtime_settings() -> None:
    if get_settings is None:
        return
    settings = get_settings()
    if settings.database_url:
        config.set_main_option("sqlalchemy.url", settings.database_url)
    if settings.db_schema:
        config.set_main_option("version_table_schema", settings.db_schema)
    masked = _mask_dsn(settings.database_url or "")
    print(f"[alembic] using database url: {masked}")

target_metadata = None

_apply_runtime_settings()


def run_migrations_offline() -> None:
    url = config.get_main_option("sqlalchemy.url")
    context.configure(
        url=url,
        target_metadata=target_metadata,
        literal_binds=True,
        dialect_opts={"paramstyle": "named"},
    )

    with context.begin_transaction():
        context.run_migrations()


def run_migrations_online() -> None:
    connectable = engine_from_config(
        config.get_section(config.config_ini_section),
        prefix="sqlalchemy.",
        poolclass=pool.NullPool,
    )

    with connectable.connect() as connection:
        context.configure(connection=connection, target_metadata=target_metadata)

        with context.begin_transaction():
            context.run_migrations()


if context.is_offline_mode():
    run_migrations_offline()
else:
    run_migrations_online()
