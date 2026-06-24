# Skill Runtime

`runtime/skills` provides the plugin-side Agent Skill contract:

- manifest registration and validation
- discovery and schema HTTP adapters
- invocation context validation
- executor dispatch and standard error/result envelopes

Business plugins register manifests and executor handlers. The framework owns HTTP shape, fail-fast checks, trace propagation, and stable `skill.*` error codes.
