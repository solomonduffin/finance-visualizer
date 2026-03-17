---
phase: 08-alert-system
plan: 04
subsystem: integration
tags: [react-router, nav, settings, smtp-ui]

# Dependency graph
requires:
  - phase: 08-02
    provides: alert API endpoints, email config endpoints
  - phase: 08-03
    provides: Alerts page, AlertRuleForm, AlertRuleCard components
provides:
  - Alerts route and nav link in App.tsx
  - Email Configuration section in Settings page
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: [SPA route wiring, SMTP form with encrypted password helper]

key-files:
  created: []
  modified:
    - frontend/src/App.tsx
    - frontend/src/pages/Settings.tsx
    - frontend/src/pages/Settings.test.tsx
    - frontend/src/components/AlertRuleForm.tsx

key-decisions:
  - title: crypto.randomUUID fallback for non-secure contexts
    choice: Fallback to Date.now + counter when crypto.randomUUID unavailable
    why: App accessed via Tailscale IP (plain HTTP) which is not a secure context

# Self-Check
## Self-Check: PASSED
- [x] All tasks executed (2/2)
- [x] Each task committed individually
- [x] Tests passing (200/200 frontend, all Go tests green)
- [x] Human verification passed
---

## Summary

Wired the Alerts page into the React router with nav link between "Net Worth" and "Settings". Added SMTP Email Configuration section to the Settings page with 6 fields (host, port, from address, username, password, recipient) plus "Test Email" and "Save" buttons. Fixed crypto.randomUUID crash on non-secure contexts (Tailscale IP access).
