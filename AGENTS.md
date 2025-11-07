# PowerXPlugin Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-10-29

## Active Technologies
- Go 1.24+, TypeScript 5.x (Nuxt 4.2) + Gin 框架、Nuxt 4.2、`@artisan-cloud/plugin-framework-admin` Layer、`@artisan-cloud/plugin-framework-client` (001-powerxplugin-foundation)
- 暂不引入持久层（Skeleton/模板以内存或 mock 为主） (001-powerxplugin-foundation)
- Go 1.24+, Node.js 18+, npm 9+, Bash (POSIX) + Go toolchain (`go test`, `go tool cover`), Playwright 1.48+, Nuxt CLI (`nuxi`), jq/standard UNIX utilities (002-testing-strategy)
- N/A (documentation and scripts only) (002-testing-strategy)
- Go 1.24 (PowerX Core/Dev API), TypeScript 5 + Node.js 18 (px-plugin CLI & tooling), Nuxt 4.2 (Admin) + Gin、`@artisan-cloud/plugin-framework-*`、px-plugin CLI runtime、S3/MinIO 对象存储、Redis/Kafka 链路、Playwright 1.48+ (004-publish-hub-spec)
- Artefacts + integrity 文件放置于对象存储；Marketplace 复用既有 DB 持久化版本/审核记录，本阶段不新增数据库 (004-publish-hub-spec)

## Project Structure

```text
src/
tests/
```

## Commands

npm test && npm run lint

## Code Style

Go 1.24+, TypeScript 5.x (Nuxt 4.2): Follow standard conventions

## Recent Changes
- 004-publish-hub-spec: Added Go 1.24 (PowerX Core/Dev API), TypeScript 5 + Node.js 18 (px-plugin CLI & tooling), Nuxt 4.2 (Admin) + Gin、`@artisan-cloud/plugin-framework-*`、px-plugin CLI runtime、S3/MinIO 对象存储、Redis/Kafka 链路、Playwright 1.48+
- 002-testing-strategy: Added Go 1.24+, Node.js 18+, npm 9+, Bash (POSIX) + Go toolchain (`go test`, `go tool cover`), Playwright 1.48+, Nuxt CLI (`nuxi`), jq/standard UNIX utilities
- 001-powerxplugin-foundation: Added Go 1.24+, TypeScript 5.x (Nuxt 4.2) + Gin 框架、Nuxt 4.2、`@artisan-cloud/plugin-framework-admin` Layer、`@artisan-cloud/plugin-framework-client`

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
