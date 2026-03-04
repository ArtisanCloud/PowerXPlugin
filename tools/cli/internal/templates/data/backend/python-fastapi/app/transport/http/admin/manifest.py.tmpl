from fastapi import APIRouter

from app.contracts.response import ok

router = APIRouter(prefix="/admin")


@router.get("/manifest")
async def manifest():
    payload = {
        "id": "com.powerx.plugins.base",
        "name": "Base Template Plugin",
        "version": "0.1.0",
        "description": "A starter plugin that showcases template management capabilities for PowerX",
        "author": "PowerX Team",
        "homepage": "https://powerx.dev/plugins/base",
        "repository": "https://powerx.dev/plugins/base.git",
        "license": "MIT",
        "tags": ["base", "template", "starter"],
        "backend": {
            "entry": "backend/bin/plugin",
            "port": 8078,
            "health": "/healthz",
        },
        "frontend": {
            "entry": "web-admin/.output",
            "routes": {"/admin/*": "index.html"},
            "public_path": "/_p/com.powerx.plugins.base/admin/",
        },
        "menus": [
            {
                "id": "base",
                "title": "menu.base.template",
                "icon": "i-heroicons-clipboard-document-check",
                "path": "/plugins/base",
                "order": 20,
                "children": [
                    {
                        "id": "base.intro",
                        "title": "menu.base.intro",
                        "icon": "i-heroicons-information-circle",
                        "path": "/intro",
                        "order": 1,
                    },
                    {
                        "id": "base.templates",
                        "title": "menu.base.templates.title",
                        "icon": "i-heroicons-clipboard-document-list",
                        "path": "/templates",
                        "order": 2,
                        "children": [
                            {
                                "id": "base.templates.develop",
                                "title": "menu.base.templates.develop",
                                "icon": "i-heroicons-document-text",
                                "path": "/templates/develop",
                                "order": 1,
                            },
                            {
                                "id": "base.templates.crud",
                                "title": "menu.base.templates.crud",
                                "icon": "i-heroicons-wrench",
                                "path": "/templates/crud",
                                "order": 2,
                            },
                        ],
                    },
                ],
                "required_permissions": ["base:template:read"],
            }
        ],
        "permissions": [
            {
                "resource": "base:template",
                "actions": ["read", "create", "update", "delete"],
                "description": "Template management permissions",
            }
        ],
        "agents": [
            {
                "id": "base.assistant",
                "plugin_id": "com.powerx.plugins.base",
                "name": "Base 助理",
                "description": "智能的 Base 模板助手，可以帮助创建与查询模板内容",
                "model": "gpt-4",
                "instructions": "你是一个专业的 Base 模板助手。你可以帮助用户创建、查询和管理模板信息。请始终以友好、专业的方式回应用户的请求。",
                "default_tools": [
                    "base.template.create",
                    "base.template.query",
                ],
                "required_permissions": ["base:template:read"],
            }
        ],
        "tools": [
            {
                "id": "base.template.create",
                "plugin_id": "com.powerx.plugins.base",
                "name": "创建模板",
                "description": "创建一个新的模板记录",
                "transport": "http",
                "endpoint": "/api/v1/templates",
                "method": "POST",
                "rbac_resource": "base:template",
                "input_schema": {
                    "type": "object",
                    "properties": {
                        "name": {"type": "string", "description": "模板名称"},
                        "description": {"type": "string", "description": "模板描述"},
                        "content": {"type": "string", "description": "模板内容"},
                    },
                    "required": ["name", "description", "content"],
                },
                "output_schema": {
                    "type": "object",
                    "properties": {
                        "id": {"type": "integer", "description": "模板ID"},
                        "name": {"type": "string", "description": "模板名称"},
                        "description": {"type": "string", "description": "模板描述"},
                        "content": {"type": "string", "description": "模板内容"},
                    },
                },
                "timeout": 30,
            },
            {
                "id": "base.template.query",
                "plugin_id": "com.powerx.plugins.base",
                "name": "查询模板",
                "description": "查询模板列表",
                "transport": "http",
                "endpoint": "/api/v1/templates",
                "method": "GET",
                "rbac_resource": "base:template",
                "input_schema": {
                    "type": "object",
                    "properties": {
                        "q": {"type": "string", "description": "按名称或描述搜索"},
                        "page": {"type": "integer", "default": 1, "description": "页码"},
                        "limit": {"type": "integer", "default": 20, "description": "每页数量"},
                    },
                },
                "output_schema": {
                    "type": "object",
                    "properties": {
                        "list": {"type": "array", "description": "模板列表"},
                        "total": {"type": "integer", "description": "总数"},
                        "page": {"type": "integer", "description": "页码"},
                        "limit": {"type": "integer", "description": "每页数量"},
                    },
                },
                "timeout": 30,
            },
        ],
    }
    return ok(payload)


@router.get("/rbac")
async def rbac():
    payload = {
        "resources": [
            {
                "name": "base:template",
                "description": "Base 模板管理",
                "actions": [
                    {"name": "read", "description": "查看模板"},
                    {"name": "create", "description": "创建模板"},
                    {"name": "update", "description": "更新模板"},
                    {"name": "delete", "description": "删除模板"},
                ],
            }
        ],
        "roles": [
            {
                "name": "base_master",
                "description": "Base Master 角色",
                "permissions": ["base:template:*"],
            },
            {
                "name": "template_editor",
                "description": "模板编辑角色",
                "permissions": [
                    "base:template:read",
                    "base:template:create",
                    "base:template:update",
                ],
            },
            {
                "name": "template_viewer",
                "description": "模板查看角色",
                "permissions": ["base:template:read"],
            },
        ],
        "permissions": [
            {"resource": "base:template", "action": "read"},
            {"resource": "base:template", "action": "create"},
            {"resource": "base:template", "action": "update"},
            {"resource": "base:template", "action": "delete"},
        ],
    }
    return ok(payload)
