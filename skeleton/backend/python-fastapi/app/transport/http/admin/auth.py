from fastapi import APIRouter

from app.contracts.response import ok
from app.services.auth_service import AuthService

router = APIRouter(prefix="/admin")
service = AuthService()


@router.post("/user/auth/login")
async def login(payload: dict):
    return ok(service.login(payload))


@router.post("/user/auth/register")
async def register(payload: dict):
    return ok(service.register(payload))


@router.post("/user/auth/logout")
async def logout(payload: dict):
    return ok(service.logout(payload))


@router.post("/user/auth/refresh")
async def refresh(payload: dict):
    return ok(service.refresh(payload))


@router.get("/user/auth/me")
async def me():
    return ok(service.me())


@router.put("/user/auth/profile")
async def profile(payload: dict):
    return ok(service.profile(payload))


@router.post("/user/auth/change-password")
async def change_password(payload: dict):
    return ok(service.change_password(payload))


@router.post("/user/auth/reset-password")
async def reset_password(payload: dict):
    return ok(service.reset_password(payload))


@router.post("/user/auth/reset-password/confirm")
async def reset_password_confirm(payload: dict):
    return ok(service.reset_password_confirm(payload))


@router.get("/user/auth/validate")
async def validate():
    return ok(service.validate())


@router.get("/user/auth/permissions")
async def permissions():
    return ok(service.permissions())

@router.get("/user/auth/me/context")
async def me_context():
    return ok({})


@router.get("/user/auth/me/tenants")
async def me_tenants():
    return ok([])


@router.post("/user/auth/me/switch-tenant")
async def switch_tenant(payload: dict):
    return ok(payload)


@router.get("/user/auth/me/roles")
async def me_roles():
    return ok([])


@router.get("/user/auth/me/departments")
async def me_departments():
    return ok([])


@router.post("/user/auth/me/avatar")
async def me_avatar(payload: dict):
    return ok(payload)


@router.post("/user/auth/me/check-permission")
async def check_permission(payload: dict):
    return ok({"has_permission": True})


@router.get("/users")
async def list_users():
    return ok([])


@router.get("/users/{user_id}")
async def get_user(user_id: str):
    return ok({})


@router.post("/users")
async def create_user(payload: dict):
    return ok(payload)


@router.put("/users/{user_id}")
async def update_user(user_id: str, payload: dict):
    return ok(payload)


@router.delete("/users/{user_id}")
async def delete_user(user_id: str):
    return ok({"deleted": True})


@router.post("/users/batch-delete")
async def delete_users(payload: dict):
    return ok({"deleted": True})


@router.patch("/users/{user_id}/status")
async def update_user_status(user_id: str, payload: dict):
    return ok({"updated": True})


@router.get("/roles")
async def list_roles():
    return ok([])


@router.get("/roles/{role_id}")
async def get_role(role_id: str):
    return ok({})


@router.post("/roles")
async def create_role(payload: dict):
    return ok(payload)


@router.put("/roles/{role_id}")
async def update_role(role_id: str, payload: dict):
    return ok(payload)


@router.delete("/roles/{role_id}")
async def delete_role(role_id: str):
    return ok({"deleted": True})


@router.post("/roles/{role_id}/permissions")
async def update_role_permissions(role_id: str, payload: dict):
    return ok(payload)


@router.get("/departments")
async def list_departments():
    return ok([])


@router.get("/departments/{department_id}")
async def get_department(department_id: str):
    return ok({})


@router.post("/departments")
async def create_department(payload: dict):
    return ok(payload)


@router.put("/departments/{department_id}")
async def update_department(department_id: str, payload: dict):
    return ok(payload)


@router.delete("/departments/{department_id}")
async def delete_department(department_id: str):
    return ok({"deleted": True})


@router.get("/departments/tree")
async def get_department_tree():
    return ok([])


@router.get("/permissions")
async def list_permissions():
    return ok([])


@router.get("/permissions/groups")
async def list_permission_groups():
    return ok({})
