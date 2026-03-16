---
phase: 08-alert-system
plan: 02
subsystem: api
tags: [crud, smtp, chi-router, jwt-secret, sync-hook]

# Dependency graph
requires:
  - phase: 08-01
    provides: internal/alerts package (engine, crypto, evaluator, notifier)
provides:
  - Alert CRUD HTTP handlers (Create, List, Update, Toggle, Delete)
  - Email config handlers (Save, Get, Test)
  - Post-sync alert evaluation hook
  - JWT secret threading through router and sync
affects: [08-04 (settings page SMTP UI)]

# Tech tracking
tech-stack:
  added: []
  patterns: [chi route groups, jwtSecret threading, sync hook for alert evaluation]

key-files:
  created:
    - internal/api/handlers/alerts.go
    - internal/api/handlers/alerts_test.go
    - internal/api/handlers/email.go
    - internal/api/handlers/email_test.go
    - internal/api/jwt_secret.go
  modified:
    - internal/api/router.go
    - internal/api/router_test.go
    - internal/sync/sync.go
    - internal/sync/sync_test.go
    - cmd/server/main.go
    - internal/api/handlers/settings.go
    - internal/api/handlers/settings_test.go

key-decisions:
  - title: JWT secret threading via function parameter
    choice: Pass jwtSecret as parameter to NewRouter and sync functions
    why: Consistent with existing parameter passing pattern; avoids global state

# Self-Check
## Self-Check: PASSED
- [x] All tasks executed (2/2)
- [x] Each task committed individually (3 commits + 1 fix)
- [x] Tests passing (13 new handler tests + all router tests green)
---

## Summary

Built the HTTP API layer for the alert system. Five CRUD endpoints for alert rules (create, list, update, toggle, delete) and three email configuration endpoints (save SMTP config, get config, test email). Wired the post-sync alert evaluation hook so alerts are checked after each Plaid sync. Threaded jwtSecret through the router and sync layer for SMTP credential encryption.
