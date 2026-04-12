# Implementation Plan: IAM 联邦渠道扫码登录（企微/钉钉/飞书）

**Branch**: `019-iam-federated-channel-login` | **Date**: 2026-04-11 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/019-iam-federated-channel-login/spec.md`

## Summary

实现可扩展的联邦扫码登录框架：
1. provider 抽象与三方渠道适配；
2. 扫码挑战与授权回调链路；
3. 外部身份绑定/映射与 JIT 入库；
4. 角色/部门映射策略；
5. 风控审计与双模式（standalone/delegated）一致身份输出。

## Technical Context

- **Language/Version**: Go 1.24  
- **Primary Dependencies**: IAM service、auth context、RBAC、observability/audit  
- **Storage**: 复用现有 IAM 表并新增 external identity/binding 相关模型  
- **Testing**: unit + integration + callback security tests + e2e login tests  
- **Target Platform**: Linux plugin runtime + Web admin/public login endpoints  
- **Project Type**: backend auth + IAM integration

## Constitution Check

- Host Contract First: PASS（delegated 仍遵循宿主身份契约）  
- Tenant Isolation & Zero Trust: PASS（跨租户边界与回调签名校验）  
- Service-Centric Architecture: PASS（provider 与 binding 通过服务层抽象）  
- Observable & Testable Delivery: PASS（登录链路全程审计与风控事件）

## Project Structure

```text
specs/019-iam-federated-channel-login/
├── spec.md
├── plan.md
└── tasks.md
```

```text
skeleton/backend/go-gin/internal/
├── services/iam/federated/
├── transport/http/public/auth/
├── middleware/
└── observability/auth/

framework/backend/go/
└── iam/federated/
```

## Milestones

1. M1: provider 抽象与回调安全模型完成。  
2. M2: identity binding + JIT 流程完成。  
3. M3: role/department 映射和双模式上下文打通。  
4. M4: 风控审计与回归测试完成。

