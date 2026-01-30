# Quickstart — Customer Auth Modes

1. **Select mode**
   - Set `customerAuth.mode=local` for Skeleton or `delegate` for host integration.
   - Provide required config keys: `delegate_endpoint`, `jwt_*`, cache TTL, etc.

2. **Run migrations (Skeleton)**
   ```bash
   POWERX_RUN_MIGRATE=1 make migrate
   ```
   Ensures `customer_accounts` (local customer credentials) exists.

3. **Start services**
   ```bash
   make dev # backend + web-admin
   ```

4. **Register/Login (Skeleton)**
   ```bash
   curl -X POST http://127.0.0.1:8078/api/v1/mini-app/auth/register \
     -H "Content-Type: application/json" \
     -H "X-PowerX-Tenant: <tid>" \
     -d '{"email":"demo@example.com","password":"P@ssword1!"}'

   curl -X POST http://127.0.0.1:8078/api/v1/mini-app/auth/login \
     -H "Content-Type: application/json" \
     -d '{"login":"demo@example.com","password":"P@ssword1!"}'
   ```
   - 多租户提示：当同一 `login`（邮箱/手机号）在多个租户存在且登录未指定 `tenant_uuid` 时，接口返回 `409` + `TENANT_SELECTION_REQUIRED`，并在 `error.details.tenants[]` 给出候选租户列表；客户端选择后再次登录即可。
   - 使用返回的 JWT 调用 `/mini-app/*`（例如 `/mini-app/ping`）。`X-PowerX-Tenant` 可省略（tenant 会从 token 注入），若显式携带则必须与 token tenant 一致。

5. **Delegated validation**
   - Point `customerAuth.delegate_endpoint` to the host CRM validation URL.
   - Provide tenant-scoped STS credentials if the host requires signed requests.
   - Optional: enable cache by setting `customerAuth.cache_ttl_seconds`.

6. **Testing**
   ```bash
   mkdir -p skeleton/backend/go-gin/.cache/go-build
   GOCACHE=$(pwd)/skeleton/backend/go-gin/.cache/go-build go test ./skeleton/backend/go-gin/tests/unit ./skeleton/backend/go-gin/tests/integration/... -count=1
   npm test -- mini-app-auth
   ```

7. **Observability**
   - Monitor logs for `customer.auth` events (register/login/validation failure).
   - Verify metrics / traces remain stable.
