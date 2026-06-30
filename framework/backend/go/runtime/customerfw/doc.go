// Package customerfw provides the framework identity boundary for C-end
// customer requests.
//
// The package answers only identity and access questions:
// who the external customer is, which tenant is active, whether the customer
// token is valid, and whether the customer has an active tenant membership.
//
// It carries only generic PowerX Core customer display attributes, such as
// display name, nickname, personal name parts, avatar, locale, and timezone.
// It intentionally does not model SCRM or industry domain concepts such as
// tags, owners, follow-ups, timelines, players, guardians, learners, patients,
// fans, entitlements, benefits, growth levels, or reports. Those concepts
// belong to business plugins. Other plugins should consume those capabilities
// through plugin capability APIs such as an SCRM plugin, not through this
// framework package.
package customerfw
