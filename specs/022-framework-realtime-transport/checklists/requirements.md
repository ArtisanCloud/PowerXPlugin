# Requirements Checklist: Framework Realtime Transport

**Feature**: `022-framework-realtime-transport`  
**Date**: 2026-06-11

## Completeness

- [x] User stories cover frontend client, backend transport, scope/envelope, manifest/RBAC, Agent stream-through.
- [x] Requirements define both WS and SSE responsibilities.
- [x] Agent SSE exception is explicitly documented.
- [x] Success criteria are measurable.
- [x] Existing feature boundaries are documented.

## Testability

- [x] Static scan can detect direct WebSocket/EventSource/gin-contrib/sse usage.
- [x] Backend tests identify target packages.
- [x] Frontend tests identify target migration pages.
- [x] Manual host/proxy validation steps are present.

## Scope Control

- [x] Does not reimplement TaskBus/EventBridge from 008.
- [x] Does not replace WS Bus Adapter from 015.
- [x] Does not move Agent Skill business registry from 021.
- [x] Keeps skeleton migrations as proof points for framework infrastructure.
