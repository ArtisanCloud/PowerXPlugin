from __future__ import annotations

from datetime import datetime
from pathlib import Path
from typing import Any
from uuid import uuid4
import hashlib
import json
import logging
import os
from urllib import parse as urlparse
from urllib import request as urlrequest
from urllib.error import URLError, HTTPError

import yaml

from app.config.settings import get_settings
from app.entity.models import Capability
from app.entity.repository.capability_repository import CapabilityRepository

_CAPABILITY_REGISTRY: list[dict[str, Any]] = []
_CAPABILITY_LIFECYCLES: dict[str, dict[str, Any]] = {}
_CAPABILITY_EXPOSURES: dict[str, dict[str, Any]] = {}
_CAPABILITY_QUOTAS: dict[str, list[dict[str, Any]]] = {}
_CAPABILITY_REVIEWS: dict[str, list[dict[str, Any]]] = {}
_logger = logging.getLogger("capability_catalog_service")


def _now_iso() -> str:
    return datetime.utcnow().isoformat() + "Z"


def _derive_module(capability_id: str) -> str:
    parts = [item for item in (capability_id or "").strip().split(".") if item]
    if len(parts) <= 1:
        return capability_id or ""
    return ".".join(parts[:-1])


def _normalize_kind(raw_kind: Any, protocols: dict[str, Any]) -> str:
    kind = str(raw_kind or "").strip()
    if kind:
        return "Workflow" if kind.lower() == "workflow" else "Capability"
    if isinstance(protocols, dict) and "workflow" in protocols:
        return "Workflow"
    return "Capability"


def _normalize_execution(payload: dict[str, Any] | None = None) -> dict[str, Any]:
    source = payload or {}
    mode = str(source.get("mode") or source.get("async_mode") or "sync").strip().lower()
    if mode not in {"sync", "async"}:
        mode = "sync"
    return {
        "mode": mode,
        "callback_url": source.get("callback_url") or "",
        "sse_channel": source.get("sse_channel") or "",
        "status_endpoint": source.get("status_endpoint") or "",
    }


def _to_int(value: Any, default: int) -> int:
    try:
        return int(value)
    except (TypeError, ValueError):
        return default


def _repo_root() -> Path:
    return Path(__file__).resolve().parents[4]


def _catalog_path(root: Path) -> Path:
    return root / "plugin.d" / "capabilities.yaml"


def _safe_load_yaml(path: Path) -> dict[str, Any]:
    try:
        content = path.read_text(encoding="utf-8")
    except OSError:
        return {}
    try:
        data = yaml.safe_load(content) or {}
    except yaml.YAMLError:
        return {}
    return data if isinstance(data, dict) else {}


def _descriptor_checksum(path: Path) -> str:
    try:
        content = path.read_bytes()
    except OSError:
        return ""
    return hashlib.sha256(content).hexdigest()[:16]


def _normalize_source(value: str | None) -> str:
    source = str(value or "").strip().lower()
    if source in {"", "all", "any"}:
        return "all"
    if source == "platform":
        return "corex"
    if source == "plugin":
        return "plugin"
    if source == "corex":
        return "corex"
    return "all"


def _merge_capability_entries(primary: list[dict[str, Any]], secondary: list[dict[str, Any]]) -> list[dict[str, Any]]:
    merged: list[dict[str, Any]] = []
    seen: set[str] = set()
    for bucket in (primary, secondary):
        for item in bucket:
            capability_id = str(item.get("id") or "").strip()
            if not capability_id or capability_id in seen:
                continue
            seen.add(capability_id)
            merged.append(item)
    return merged


def _tenant_uuid_from_jwt(token: str) -> str:
    raw = str(token or "").strip()
    if not raw:
        return ""
    parts = raw.split(".")
    if len(parts) < 2:
        return ""
    payload = parts[1]
    rem = len(payload) % 4
    if rem:
        payload += "=" * (4 - rem)
    try:
        import base64
        decoded = base64.urlsafe_b64decode(payload.encode("utf-8")).decode("utf-8")
        claims = json.loads(decoded)
    except Exception:
        return ""
    if not isinstance(claims, dict):
        return ""
    return str(claims.get("tid") or "").strip()


def _normalize_api_prefix(raw: str | None) -> str:
    value = str(raw or "").strip()
    if not value:
        return "/api/v1"
    if not value.startswith("/"):
        value = "/" + value
    value = "/" + value.strip("/")
    if value == "/":
        return "/api/v1"
    return value


class CapabilityService:
    def __init__(self, repo: CapabilityRepository | None = None) -> None:
        self._repo = repo or CapabilityRepository()
        self._catalog_entries = self._load_catalog_entries()
        self._settings = get_settings()

    def list_sources(self) -> dict[str, Any]:
        return {
            "default": "all",
            "aliases": {
                "all": "all",
                "any": "all",
                "platform": "corex",
            },
            "sources": [
                {"id": "all", "label": "all", "description": "查询全部来源（不传 source 或 source=all）"},
                {"id": "corex", "label": "corex", "description": "PowerX 底座能力"},
                {"id": "plugin", "label": "plugin", "description": "插件/租户注册能力"},
            ],
        }

    def list_capabilities(
        self,
        source: str | None = None,
        bearer_token: str | None = None,
        tenant_uuid: str | None = None,
    ):
        normalized_source = _normalize_source(source)
        if normalized_source == "corex":
            corex_entries = self._list_corex_capabilities(bearer_token=bearer_token, tenant_uuid=tenant_uuid)
            if corex_entries:
                return corex_entries
            _logger.warning("source=corex returned empty, fallback to local catalog")
            return self._list_local_capabilities()
        if normalized_source == "all":
            corex_entries = self._list_corex_capabilities(bearer_token=bearer_token, tenant_uuid=tenant_uuid)
            local_entries = self._list_local_capabilities()
            return _merge_capability_entries(corex_entries, local_entries)
        return self._list_local_capabilities()

    def _list_local_capabilities(self):
        entries = {entry["id"]: dict(entry) for entry in self._catalog_entries if entry.get("id")}

        try:
            repo_items = self._repo.list_capabilities()
        except Exception:
            repo_items = []
        for item in repo_items:
            capability_id = str(item.id or "").strip()
            if not capability_id:
                continue
            current = entries.get(capability_id) or {
                "id": capability_id,
                "descriptor": item.name or capability_id,
                "module": _derive_module(capability_id),
                "kind": "Capability",
                "tags": [],
                "checksum": "",
                "execution": _normalize_execution(),
                "protocols": {},
            }
            current["version"] = item.version or current.get("version") or "1.0.0"
            entries[capability_id] = current

        for record in _CAPABILITY_REGISTRY:
            capability_id = str(record.get("capability_id") or "").strip()
            if not capability_id:
                continue
            metadata = record.get("metadata") or {}
            protocols = record.get("protocols") or {}
            current = entries.get(capability_id) or {}
            current.update(
                {
                    "id": capability_id,
                    "version": metadata.get("version") or current.get("version") or "1.0.0",
                    "descriptor": current.get("descriptor") or capability_id,
                    "module": current.get("module") or _derive_module(capability_id),
                    "kind": _normalize_kind(record.get("type"), protocols),
                    "tags": record.get("tags") or current.get("tags") or [],
                    "checksum": current.get("checksum") or "",
                    "execution": _normalize_execution(record.get("async_config") or {}),
                    "protocols": protocols if isinstance(protocols, dict) else {},
                }
            )
            entries[capability_id] = current

        return [entries[key] for key in sorted(entries.keys())]

    def _list_corex_capabilities(
        self,
        bearer_token: str | None = None,
        tenant_uuid: str | None = None,
    ) -> list[dict[str, Any]]:
        base_url = self._effective_gateway_base_url()
        if not base_url:
            _logger.warning("gateway base url missing, cannot load platform capability catalog")
            return []

        auth_scheme = self._resolve_gateway_auth_scheme()
        credential = self._resolve_gateway_credential(auth_scheme, bearer_token=bearer_token)
        if not credential:
            _logger.warning("gateway credential missing, auth_scheme=%s", auth_scheme)
            return []

        query = urlparse.urlencode({"source": "corex", "page_size": 200})
        request_urls = [f"{base_url}/tenant/capabilities?{query}"]
        if auth_scheme == "bearer":
            request_urls.append(f"{base_url}/admin/platform-capabilities?page=1&page_size=200")
        payload_bytes = self._fetch_platform_payload(
            request_urls=request_urls,
            auth_scheme=auth_scheme,
            credential=credential,
            tenant_uuid=tenant_uuid,
        )
        if payload_bytes is None:
            return []

        try:
            payload = json.loads(payload_bytes.decode("utf-8"))
        except (UnicodeDecodeError, json.JSONDecodeError) as exc:
            _logger.warning("decode platform capability catalog failed: %s", exc)
            return []

        records = self._extract_platform_records(payload)
        result: list[dict[str, Any]] = []
        for record in records:
            item = self._platform_record_to_entry(record)
            if item.get("id"):
                result.append(item)
        if not result:
            _logger.warning("platform capability catalog returned empty records")
        return result

    def _resolve_gateway_auth_scheme(self) -> str:
        raw = (
            os.getenv("PX_GATEWAY_AUTH_SCHEME", "").strip()
            or str(getattr(self._settings, "gateway_auth_scheme", "") or "").strip()
        ).lower()
        if raw in {"apikey", "api_key", "api-key"}:
            return "apikey"
        if raw == "bearer":
            return "bearer"
        api_key = self._read_first_nonempty_env("PX_GATEWAY_API_KEY", "PX_PLUGIN_API_KEY")
        sts_client = self._read_first_nonempty_env("POWERX_STS_CLIENT_ID", "PX_STS_CLIENT_ID")
        if api_key and not sts_client:
            return "apikey"
        return "bearer"

    def _resolve_gateway_credential(self, auth_scheme: str, bearer_token: str | None = None) -> str:
        if auth_scheme == "apikey":
            return self._read_first_nonempty_env("PX_GATEWAY_API_KEY", "PX_PLUGIN_API_KEY")
        token = str(bearer_token or "").strip()
        if token:
            return token
        raise RuntimeError("bearer gateway mode requires STS token provider; static bearer credentials are not supported")

    def _resolve_gateway_tenant(self, auth_scheme: str, credential: str, tenant_uuid: str | None = None) -> str:
        if auth_scheme != "bearer":
            return ""
        token_tenant = _tenant_uuid_from_jwt(credential)
        return token_tenant

    def _fetch_platform_payload(
        self,
        request_urls: list[str],
        auth_scheme: str,
        credential: str,
        tenant_uuid: str | None = None,
    ) -> bytes | None:
        auth_header = f"ApiKey {credential.strip()}" if auth_scheme == "apikey" else f"Bearer {credential.strip()}"
        tenant = self._resolve_gateway_tenant(auth_scheme, credential, tenant_uuid=tenant_uuid)
        last_exc: Exception | None = None
        for request_url in request_urls:
            req = urlrequest.Request(request_url, method="GET")
            req.add_header("Accept", "application/json")
            req.add_header("Authorization", auth_header)
            if tenant:
                req.add_header("tenant_uuid", tenant)
            try:
                with urlrequest.urlopen(req, timeout=8) as resp:
                    return resp.read()
            except (URLError, HTTPError) as exc:
                last_exc = exc
                _logger.warning("load platform capability catalog failed: %s url=%s", exc, request_url)
                continue
        if last_exc is not None:
            _logger.warning("all platform capability endpoints failed")
        return None

    def _read_first_nonempty_env(self, *names: str) -> str:
        for name in names:
            value = os.getenv(name, "").strip()
            if value:
                return value
        return ""

    def _effective_gateway_base_url(self) -> str:
        base_url = str(getattr(self._settings, "gateway_base_url", "") or "").strip().rstrip("/")
        if not base_url:
            return ""
        api_prefix = _normalize_api_prefix(getattr(self._settings, "gateway_api_prefix", "/api/v1"))
        if base_url.endswith(api_prefix):
            return base_url
        return f"{base_url}{api_prefix}"

    def _extract_platform_records(self, payload: Any) -> list[dict[str, Any]]:
        if isinstance(payload, list):
            return [item for item in payload if isinstance(item, dict)]
        if not isinstance(payload, dict):
            return []
        raw = payload.get("data")
        if isinstance(raw, list):
            return [item for item in raw if isinstance(item, dict)]
        if isinstance(raw, dict):
            for key in ("items", "list", "records", "rows"):
                value = raw.get(key)
                if isinstance(value, list):
                    return [item for item in value if isinstance(item, dict)]
        for key in ("items", "list", "records", "rows"):
            value = payload.get(key)
            if isinstance(value, list):
                return [item for item in value if isinstance(item, dict)]
        return []

    def _platform_record_to_entry(self, record: dict[str, Any]) -> dict[str, Any]:
        capability_id = str(
            record.get("capability_id")
            or record.get("capabilityId")
            or record.get("id")
            or ""
        ).strip()
        protocols = self._normalize_platform_protocols(record.get("protocols"))
        tags = record.get("categories")
        if not isinstance(tags, list):
            tags = record.get("tags") if isinstance(record.get("tags"), list) else []
        version = str(
            record.get("plugin_version")
            or record.get("pluginVersion")
            or record.get("version")
            or "1.0.0"
        ).strip()
        execution_mode = str(
            record.get("execution_mode")
            or record.get("executionMode")
            or "sync"
        ).strip().lower() or "sync"
        checksum = str(
            record.get("capabilities_hash")
            or record.get("capabilitiesHash")
            or record.get("protocol_hash")
            or record.get("protocolHash")
            or ""
        ).strip()
        return {
            "id": capability_id,
            "version": version,
            "descriptor": "",
            "module": _derive_module(capability_id),
            "kind": "Workflow" if "workflow" in protocols else "Capability",
            "tags": [str(tag).strip() for tag in tags if str(tag).strip()],
            "checksum": checksum,
            "execution": _normalize_execution({"mode": execution_mode}),
            "protocols": protocols,
        }

    def _normalize_platform_protocols(self, value: Any) -> dict[str, Any]:
        if isinstance(value, dict):
            return value
        if not isinstance(value, list):
            return {}
        grouped: dict[str, list[dict[str, Any]]] = {}
        for item in value:
            if not isinstance(item, dict):
                continue
            channel = str(item.get("channel") or "rest").strip().lower() or "rest"
            payload: dict[str, Any] = {}
            endpoint = str(item.get("endpoint") or "").strip()
            method = str(item.get("method") or "").strip().upper()
            rpc = str(item.get("rpc") or "").strip()
            schema_ref = str(item.get("schema_ref") or item.get("schemaRef") or "").strip()
            tool_ref = str(item.get("tool_ref") or item.get("toolRef") or "").strip()
            if endpoint:
                payload["endpoint"] = endpoint
                if channel in {"rest", "http"}:
                    payload["path"] = endpoint
                else:
                    payload["service"] = endpoint
            if method:
                payload["method"] = method
            if rpc:
                payload["rpc"] = rpc
                if "method" not in payload:
                    payload["method"] = rpc
            if schema_ref:
                payload["schema_ref"] = schema_ref
            if tool_ref:
                payload["tool_ref"] = tool_ref
            if not payload:
                payload["defined"] = True
            grouped.setdefault(channel, []).append(payload)
        result: dict[str, Any] = {}
        for channel, entries in grouped.items():
            result[channel] = entries[0] if len(entries) == 1 else entries
        return result

    def register_template(self):
        namespace = ""
        for entry in self._catalog_entries:
            candidate = _derive_module(str(entry.get("id") or ""))
            if not candidate:
                continue
            namespace = candidate.rsplit(".", 1)[0] if "." in candidate else candidate
            break
        if not namespace:
            namespace = "com.powerx.plugins.base"
        return {
            "namespace": namespace,
            "sensitivity_options": ["low", "medium", "high"],
            "async_modes": ["sync", "async"],
            "tag_suggestions": ["integration", "workflow", "agent", "draft"],
            "field_hints": {
                "name.zh": "必填：展示给国内租户的能力名称",
                "name.en": "必填：用于全球站点的英文名称",
                "summary.zh": "一句话描述能力价值，最多 120 字",
                "summary.en": "One-line summary visible to global tenants",
                "schemas.input": "引用 contracts/schema/input/*.json",
                "schemas.output": "引用 contracts/schema/output/*.json",
            },
            "schema_placeholders": {
                "input": f"contracts/schema/input/{namespace}.sample.json",
                "output": f"contracts/schema/output/{namespace}.sample.json",
            },
            "protocol_samples": {
                "rest_path": "/api/v1/templates",
                "grpc_service": "powerx.template.TemplateService/Create",
                "workflow_template": "contracts/exposure/workflow/template-sample.json",
            },
            "identifier_example": f"{namespace}.template.create",
        }

    def register(self, payload: dict):
        capability_id = uuid4().hex
        name = payload.get("name") or {}
        display_name = name.get("zh") or name.get("en") or payload.get("resource") or "capability"
        record = {
            **payload,
            "capability_id": capability_id,
            "status": "draft" if payload.get("draft", False) else "submitted",
            "created_at": _now_iso(),
            "updated_at": _now_iso(),
        }
        _CAPABILITY_REGISTRY.append(record)
        cap = Capability(
            id=capability_id,
            name=display_name,
            status=record["status"],
            version=payload.get("metadata", {}).get("version") if payload.get("metadata") else None,
            created_at=datetime.utcnow(),
            updated_at=datetime.utcnow(),
        )
        self._repo.create(cap)
        return record

    def validate(self, payload: dict):
        return {"capability_id": payload.get("capability_id") or uuid4().hex, "valid": True, "errors": []}

    def lifecycle_template(self):
        return {
            "change_types": ["upgrade", "deprecate", "rollback", "hotfix"],
            "status_options": ["draft", "pending", "approved", "paused", "completed"],
            "channel_options": ["email", "webhook", "slack"],
        }

    def list_lifecycle(self):
        items = []
        for record in _CAPABILITY_LIFECYCLES.values():
            items.append(self._normalize_lifecycle_record(record))
        return items

    def create_lifecycle(self, payload: dict):
        plan_id = str(payload.get("plan_id") or payload.get("id") or uuid4().hex).strip()
        capability_id = str(payload.get("capability_id") or "").strip()
        channels = payload.get("notification_channels")
        if not isinstance(channels, list):
            channels = []
        windows = payload.get("windows")
        if not isinstance(windows, list):
            windows = []
        record = {
            "id": plan_id,
            "plan_id": plan_id,
            "capability_id": capability_id,
            "change_type": payload.get("change_type") or "upgrade",
            "diff_summary": payload.get("diff_summary") or "",
            "notification_channels": [str(item).strip() for item in channels if str(item).strip()],
            "grace_period_hours": _to_int(payload.get("grace_period_hours"), 72),
            "dual_run_until": payload.get("dual_run_until") or "",
            "rollback_plan": payload.get("rollback_plan") or "",
            "windows": windows,
            "metadata": payload.get("metadata") if isinstance(payload.get("metadata"), dict) else {},
            "plan_id": plan_id,
            "status": payload.get("status") or "draft",
            "created_at": _now_iso(),
            "updated_at": _now_iso(),
        }
        _CAPABILITY_LIFECYCLES[plan_id] = record
        return self._normalize_lifecycle_record(record)

    def update_lifecycle_status(self, plan_id: str, payload: dict):
        key = str(plan_id or "").strip()
        record = _CAPABILITY_LIFECYCLES.get(key, {"id": key, "plan_id": key, "capability_id": ""})
        if payload.get("status"):
            record["status"] = payload["status"]
        if payload.get("notes") is not None:
            record["notes"] = payload.get("notes")
        record["updated_at"] = _now_iso()
        _CAPABILITY_LIFECYCLES[key] = record
        return self._normalize_lifecycle_record(record)

    def exposure_template(self):
        return {
            "channels": ["public", "private"],
            "regions": ["global"],
        }

    def exposure_detail(self, capability_id: str):
        return _CAPABILITY_EXPOSURES.get(capability_id, {"capability_id": capability_id})

    def update_exposure(self, capability_id: str, payload: dict):
        record = {**payload, "capability_id": capability_id, "updated_at": _now_iso()}
        _CAPABILITY_EXPOSURES[capability_id] = record
        return record

    def list_quotas(self, capability_id: str):
        return _CAPABILITY_QUOTAS.get(capability_id, [])

    def update_quotas(self, capability_id: str, payload: dict):
        quotas = payload.get("quotas") or []
        _CAPABILITY_QUOTAS[capability_id] = quotas
        return {"capability_id": capability_id, "quotas": quotas}

    def list_reviews(self, capability_id: str):
        return _CAPABILITY_REVIEWS.get(capability_id, [])

    def resubmit_review(self, capability_id: str, payload: dict | None = None):
        tasks = _CAPABILITY_REVIEWS.get(capability_id, [])
        record = {
            "task_id": uuid4().hex,
            "status": "resubmitted",
            "actor": (payload or {}).get("actor"),
            "note": (payload or {}).get("note"),
            "attachments": (payload or {}).get("attachments") or [],
            "created_at": _now_iso(),
        }
        tasks.append(record)
        _CAPABILITY_REVIEWS[capability_id] = tasks
        return tasks

    def add_review_comment(self, task_id: str, payload: dict | None = None):
        return {
            "task_id": task_id,
            "author": (payload or {}).get("author"),
            "message": (payload or {}).get("message"),
            "attachments": (payload or {}).get("attachments") or [],
            "created_at": _now_iso(),
        }

    def decide_review(self, task_id: str, payload: dict | None = None):
        return {
            "task_id": task_id,
            "actor": (payload or {}).get("actor"),
            "decision": (payload or {}).get("decision"),
            "note": (payload or {}).get("note"),
            "attachments": (payload or {}).get("attachments") or [],
            "updated_at": _now_iso(),
        }

    def _normalize_lifecycle_record(self, record: dict[str, Any]) -> dict[str, Any]:
        plan_id = str(record.get("id") or record.get("plan_id") or "").strip()
        channels = record.get("notification_channels")
        if not isinstance(channels, list):
            channels = []
        windows = record.get("windows")
        if not isinstance(windows, list):
            windows = []
        metadata = record.get("metadata")
        if not isinstance(metadata, dict):
            metadata = {}
        return {
            "id": plan_id,
            "plan_id": plan_id,
            "capability_id": str(record.get("capability_id") or "").strip(),
            "change_type": str(record.get("change_type") or "upgrade"),
            "diff_summary": str(record.get("diff_summary") or ""),
            "notification_channels": [str(item).strip() for item in channels if str(item).strip()],
            "grace_period_hours": _to_int(record.get("grace_period_hours"), 72),
            "dual_run_until": str(record.get("dual_run_until") or ""),
            "rollback_plan": str(record.get("rollback_plan") or ""),
            "windows": windows,
            "status": str(record.get("status") or "draft"),
            "notes": str(record.get("notes") or ""),
            "created_at": record.get("created_at") or _now_iso(),
            "updated_at": record.get("updated_at") or _now_iso(),
            "metadata": metadata,
        }

    def _load_catalog_entries(self) -> list[dict[str, Any]]:
        root = _repo_root()
        catalog = _safe_load_yaml(_catalog_path(root))
        capabilities = catalog.get("capabilities") if isinstance(catalog.get("capabilities"), dict) else {}
        provides = capabilities.get("provides") if isinstance(capabilities.get("provides"), list) else []
        entries: list[dict[str, Any]] = []
        for item in provides:
            if not isinstance(item, dict):
                continue
            capability_id = str(item.get("id") or "").strip()
            if not capability_id:
                continue
            descriptor = str(item.get("descriptor") or "").strip()
            descriptor_path = (root / descriptor) if descriptor else None
            descriptor_data = _safe_load_yaml(descriptor_path) if descriptor_path else {}
            metadata = descriptor_data.get("metadata") if isinstance(descriptor_data.get("metadata"), dict) else {}
            protocols = metadata.get("protocols") if isinstance(metadata.get("protocols"), dict) else {}
            tags = metadata.get("tags")
            if not isinstance(tags, list):
                tags = descriptor_data.get("tags") if isinstance(descriptor_data.get("tags"), list) else []
            entries.append(
                {
                    "id": capability_id,
                    "version": str(item.get("version") or descriptor_data.get("version") or "1.0.0"),
                    "descriptor": descriptor,
                    "module": _derive_module(capability_id),
                    "kind": _normalize_kind(descriptor_data.get("type"), protocols),
                    "tags": [str(tag) for tag in tags if str(tag).strip()],
                    "checksum": _descriptor_checksum(descriptor_path) if descriptor_path else "",
                    "execution": _normalize_execution(metadata.get("execution") if isinstance(metadata.get("execution"), dict) else {}),
                    "protocols": protocols,
                }
            )
        return entries
