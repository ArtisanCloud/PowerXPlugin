from dataclasses import dataclass, field
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
    dev_mode: bool = False
    server_dev_mode: bool = False
    server_secret_key: str = ""
    security_toolgrant_secret: str = ""
    context_issuer: str = ""
    context_audience: str = ""
    context_hmac_secret: str = ""
    context_ttl_seconds: int = 900
    grpc_upstream_tenant_uuid: str = ""
    grpc_upstream_sts_client_id: str = ""
    grpc_upstream_sts_client_secret: str = ""
    grpc_upstream_sts_audience: str = ""
    grpc_upstream_sts_scope: str = ""
    sts_endpoint: str = ""


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


def _parse_duration_seconds(value: Any, default_seconds: int) -> int:
    if value is None:
        return default_seconds
    if isinstance(value, (int, float)):
        return int(value)
    if not isinstance(value, str):
        return default_seconds
    raw = value.strip()
    if not raw:
        return default_seconds
    unit = raw[-1].lower()
    num_part = raw[:-1] if unit.isalpha() else raw
    try:
        num = float(num_part)
    except ValueError:
        return default_seconds
    if not unit.isalpha():
        return int(num)
    if unit == "s":
        return int(num)
    if unit == "m":
        return int(num * 60)
    if unit == "h":
        return int(num * 3600)
    if unit == "d":
        return int(num * 86400)
    return default_seconds


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
    dev_mode = bool(cfg.get("dev_mode") or cfg.get("devMode"))
    server_dev_mode = bool(server_cfg.get("dev_mode")) if isinstance(server_cfg, dict) else False
    server_secret_key = server_cfg.get("secret_key") if isinstance(server_cfg, dict) else ""
    security_cfg = cfg.get("security") or {}
    security_toolgrant_secret = (
        security_cfg.get("toolgrant_secret") if isinstance(security_cfg, dict) else ""
    )
    ctx_cfg = cfg.get("context") or {}
    context_issuer = ctx_cfg.get("issuer") if isinstance(ctx_cfg, dict) else ""
    context_audience = ctx_cfg.get("audience") if isinstance(ctx_cfg, dict) else ""
    context_hmac_secret = ctx_cfg.get("hmac_secret") if isinstance(ctx_cfg, dict) else ""
    context_ttl = ctx_cfg.get("ttl") if isinstance(ctx_cfg, dict) else None
    context_ttl_seconds = _parse_duration_seconds(context_ttl, 900)
    grpc_cfg = cfg.get("grpc_upstream") or {}
    grpc_tenant_uuid = grpc_cfg.get("tenant_uuid") if isinstance(grpc_cfg, dict) else ""
    grpc_sts_client_id = grpc_cfg.get("sts_client_id") if isinstance(grpc_cfg, dict) else ""
    grpc_sts_client_secret = grpc_cfg.get("sts_client_secret") if isinstance(grpc_cfg, dict) else ""
    grpc_sts_audience = grpc_cfg.get("sts_audience") if isinstance(grpc_cfg, dict) else ""
    grpc_sts_scope = grpc_cfg.get("sts_scope") if isinstance(grpc_cfg, dict) else ""
    sts_endpoint = os.getenv("POWERX_STS_ENDPOINT", "").strip()
    env_db = os.getenv("POWERX_DB_URL") or os.getenv("DATABASE_URL")
    if env_db:
        database_url = _normalize_postgres_dsn(env_db)
    if not database_url:
        database_url = "sqlite:///./dev.db"
    return Settings(
        api_prefix=api_prefix,
        database_url=database_url,
        db_schema=db_schema,
        dev_mode=dev_mode,
        server_dev_mode=server_dev_mode,
        server_secret_key=server_secret_key or "",
        security_toolgrant_secret=security_toolgrant_secret or "",
        context_issuer=context_issuer or "",
        context_audience=context_audience or "",
        context_hmac_secret=context_hmac_secret or "",
        context_ttl_seconds=context_ttl_seconds,
        grpc_upstream_tenant_uuid=grpc_tenant_uuid or "",
        grpc_upstream_sts_client_id=grpc_sts_client_id or "",
        grpc_upstream_sts_client_secret=grpc_sts_client_secret or "",
        grpc_upstream_sts_audience=grpc_sts_audience or "",
        grpc_upstream_sts_scope=grpc_sts_scope or "",
        sts_endpoint=sts_endpoint or "",
    )
