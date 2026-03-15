# Phase 1: Foundation - Context

**Gathered:** 2026-03-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can authenticate into a running app backed by a correct, migration-managed SQLite schema. Includes: SQLite schema with migrations, bcrypt auth, JWT middleware, and Docker dev environment. The schema includes all tables the app will need across all phases — later phases just write code against the existing schema.

</domain>

<decisions>
## Implementation Decisions

### Login Experience
- Password-only login (no username) — single-user self-hosted app
- 30-day JWT session duration — set-and-forget for a personal dashboard
- Password configured via environment variable (PASSWORD or PASSWORD_HASH in .env / docker-compose)
- Rate-limited retries on wrong password: 5 attempts, then 30-second cooldown

### Schema Design
- Full schema created upfront in Phase 1 migration — all tables (accounts, balance_snapshots, settings, sync_log) built now, not incrementally per phase
- Settings table (key-value) for password hash and future config (SimpleFIN token, theme preference)
- Balance snapshots stored as one row per account per day with unique constraint to prevent duplicates
- Account type categorization approach at Claude's discretion

### Docker Dev Workflow
- Air for Go backend hot-reload during development
- Vite dev server with HMR for React frontend, Nginx proxies to it in dev mode
- Separate Docker Compose files: docker-compose.yml (dev) and docker-compose.prod.yml (production)
- Named Docker volume for SQLite database persistence across container rebuilds

### Project Structure
- Go backend: cmd/internal pattern (cmd/server/main.go entrypoint, internal/ for packages)
- Monorepo: Go backend at root, React frontend in frontend/ directory
- Database migrations in top-level migrations/ directory (numbered SQL files, golang-migrate convention)
- Tests colocated with source files (Go: _test.go beside source, React: .test.tsx beside components)

### Claude's Discretion
- Account type categorization approach (enum column vs other)
- Exact login page styling and layout
- Nginx configuration details
- Internal package organization within internal/

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- None — greenfield project, no existing code

### Established Patterns
- None yet — this phase establishes all foundational patterns

### Integration Points
- Password hash stored in settings table, read by auth middleware
- JWT middleware wraps all API routes (established here, used by Phase 3)
- SQLite connection with WAL mode set at open time (established here, used by all phases)
- golang-migrate runs on startup to ensure schema is current

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 01-foundation*
*Context gathered: 2026-03-15*
