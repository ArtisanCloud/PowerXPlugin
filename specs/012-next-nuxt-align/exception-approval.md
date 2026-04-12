# Exception Approval Record

- Exception ID: PX-FE-EXCEPTION-012
- Scope: Nuxt -> Next migration for web-admin in feature `012-next-nuxt-align`
- Status: Approved
- Approver: Feature Request Owner (confirmed in current session)
- Approval Date: 2026-03-14
- Decision: Allow Next.js migration path as a scoped exception to default frontend stack constraints for this feature only.
- Constraints:
  - Keep Nuxt behavior as parity baseline.
  - Do not introduce Next-only private backend APIs.
  - Enforce artifact gate and release verification tasks defined in `tasks.md`.
