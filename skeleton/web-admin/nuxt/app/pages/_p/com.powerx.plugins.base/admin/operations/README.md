# Operations Admin Pages

Nuxt pages in this directory expose the Support Playbook, Incident Center, and SLA dashboards. Each page should:

- Use existing layout shells and respect the `/integration/` navigation patterns.
- Source data through composables under `~/stores/operations`.
- Reuse shared checklist and analytics components where possible to keep UI consistent.
