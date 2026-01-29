from dataclasses import dataclass
import os
from typing import Any, Tuple

import yaml


@dataclass(frozen=True)
class Settings:
    app_name: str = "PowerX Plugin (FastAPI)"
    api_prefix: str = "/api/v1"
    database_url: str = "sqlite:///./dev.db"
    db_schema: str | None = None
    db_echo: bool = os.getenv("POWERX_DB_ECHO", "0") == "1"


def _resolve_config_candidates() -> list[str]:
    candidates: list[str] = []
    raw_config_path = os.getenv("CONFIG_PATH", "")
    if raw_config_path:
        config_path = os.path.expanduser(raw_config_path)
        if config_path.endswith((".yaml", ".yml")):
            candidates.append(config_path)
        else:
            candidates.extend(
                [
                    os.path.join(config_path, "host-values.yaml"),
                    os.path.join(config_path, "config.yaml"),
                ]
            )
    candidates.extend(
        [
            "./config/host-values.yaml",
            "./config/config.yaml",
            "./config.yaml",
            "./backend/etc/config.yaml",
            "./skeleton/backend/etc/config.yaml",
            "./etc/config.yaml",
            "../config/host-values.yaml",
            "../config/config.yaml",
            "../etc/config.yaml",
            "../backend/etc/config.yaml",
        ]
    )
    return candidates


def _load_config() -> Tuple[dict[str, Any], str | None]:
    for path in _resolve_config_candidates():
        if not os.path.exists(path):
            continue
        try:
            with open(path, "r", encoding="utf-8") as fh:
                data = yaml.safe_load(fh) or {}
                if isinstance(data, dict):
                    return data, path
        except OSError:
            continue
    return {}, None


def _apply_schema(dsn: str, schema: str | None) -> str:
    if not dsn or not schema:
        return dsn
    if "search_path=" in dsn:
        return dsn
    sep = "&" if "?" in dsn else "?"
    return f"{dsn}{sep}options=-csearch_path={schema}"


def _normalize_postgres_dsn(dsn: str) -> str:
    if not dsn:
        return dsn
    if dsn.startswith("postgres://"):
        return "postgresql+psycopg2://" + dsn[len("postgres://") :]
    return dsn


def _database_url_from_config(cfg: dict[str, Any]) -> str:
    db_cfg = cfg.get("database") or {}
    if not isinstance(db_cfg, dict):
        return ""
    dsn = _normalize_postgres_dsn(db_cfg.get("dsn") or "")
    schema = db_cfg.get("schema")
    return _apply_schema(dsn, schema)


def get_settings() -> Settings:
    cfg, _ = _load_config()
    server_cfg = cfg.get("server") or {}
    api_prefix = (
        server_cfg.get("api_prefix")
        if isinstance(server_cfg, dict)
        else None
    ) or "/api/v1"
    database_url = _database_url_from_config(cfg)
    db_cfg = cfg.get("database") or {}
    db_schema = db_cfg.get("schema") if isinstance(db_cfg, dict) else None
    env_db = os.getenv("POWERX_DB_URL") or os.getenv("DATABASE_URL")
    if env_db:
        database_url = _normalize_postgres_dsn(env_db)
    if not database_url:
        database_url = "sqlite:///./dev.db"
    return Settings(
        api_prefix=api_prefix,
        database_url=database_url,
        db_schema=db_schema,
    )
