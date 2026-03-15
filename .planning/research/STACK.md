# Stack Research: v1.1 Additions

**Domain:** Self-hosted personal finance dashboard -- new capabilities for email alerts, financial projections, crypto aggregation, and alert rule builder
**Researched:** 2026-03-15
**Confidence:** HIGH (go-mail verified via pkg.go.dev and GitHub releases; react-querybuilder verified via npm and official docs; SimpleFIN protocol verified via protocol spec and existing codebase; shopspring/decimal capabilities verified via GitHub issues and pkg.go.dev)

---

## Scope

This research covers ONLY the stack additions needed for v1.1 features. The existing validated stack (Go 1.25, go-chi, React 19, TypeScript 5.9, Tailwind v4, SQLite via modernc.org/sqlite, recharts 3.x, shopspring/decimal, JWT auth, Docker, Nginx) is NOT re-researched.

**New capabilities needed:**
1. SMTP email sending from Go (alert notifications)
2. Financial projection / compound interest math
3. SimpleFIN holdings data for crypto and investment detail
4. Alert rule expression builder UI
5. Projection and drill-down chart visualizations

---

## New Backend Libraries

### Email: go-mail

| Property | Value |
|----------|-------|
| **Library** | `github.com/wneessen/go-mail` |
| **Version** | v0.7.1 |
| **Purpose** | SMTP email sending for alert notifications |
| **Min Go** | Go 1.24+ (project uses 1.25, compatible) |
| **License** | MIT |

**Why go-mail over alternatives:**

- **Not net/smtp:** The Go standard library's `net/smtp` package is frozen by the Go team and will not receive new features or security improvements. go-mail forked and extended it.
- **Not gomail:** gomail is abandoned (last release 2016, archived repo). go-mail is its spiritual successor with active maintenance (v0.7.1 released 2025, CVE-2025-59937 patched).
- **Not a transactional email API (SendGrid, Mailgun):** The user is self-hosted and uses Protonmail. Adding a third-party email API dependency defeats the self-hosted philosophy and requires account signup, API keys, and ongoing cost. Direct SMTP via Protonmail Bridge keeps everything local.

**Key features relevant to this project:**

- STARTTLS support with configurable TLS policies (required for Protonmail Bridge)
- AUTH methods: PLAIN, LOGIN, CRAM-MD5, SCRAM-SHA-1, SCRAM-SHA-256, XOAUTH2
- Concurrency-safe (safe to call from alert-checking goroutine)
- Minimal dependency footprint (mostly Go stdlib)
- S/MIME signing support (v0.6.0+, optional but nice for email authenticity)
- Context-aware sending with timeouts

**Protonmail Bridge integration:**

The user uses Protonmail. Protonmail does not expose standard SMTP -- it requires Protonmail Bridge running locally, which creates a local SMTP server at `127.0.0.1:1025` using STARTTLS. For Docker deployments, a Protonmail Bridge Docker container (e.g., `shenxn/protonmail-bridge` or `VideoCurio/ProtonMailBridgeDocker`) must run alongside the app, exposing SMTP on the Docker network.

SMTP configuration fields needed in the app:
- Host (default: `127.0.0.1` or bridge container hostname)
- Port (default: `1025` for Bridge, `587` for standard SMTP)
- Username (Protonmail email address)
- Password (Bridge-generated password, NOT Protonmail password)
- Auth method (PLAIN for Bridge)
- TLS mode (STARTTLS)
- From address

**Architecture note:** Store SMTP config in SQLite settings table. The email sender should be an interface (`internal/notify/sender.go`) so it can be mocked in tests. The alert-checking goroutine calls the sender when thresholds are crossed.

### Financial Projections: No New Library Needed

**Use shopspring/decimal (already in go.mod at v1.4.0).**

Compound interest projection requires: `FV = PV * (1 + r/n)^(n*t)`

shopspring/decimal provides:
- `Decimal.Mul()` for multiplication
- `Decimal.Div()` for division
- `Decimal.Add()` for accumulation
- `Decimal.Pow()` for integer exponents (works for annual compounding where exponent is whole years)

**Critical limitation:** `Decimal.Pow()` truncates fractional exponents (e.g., `4^2.5` returns `16` not `32`). For sub-annual compounding (monthly, daily), use integer exponents by computing `(1 + r/12)^(12*years)` where the exponent `12*years` is always an integer. Alternatively, use `PowWithPrecision()` for fractional exponents with explicit precision control.

**Recommended approach for projections:**
- Monthly compounding with integer exponents: `(1 + APY/12)^months` -- exponent is always a whole number of months
- Use `shopspring/decimal` throughout -- no float64 anywhere in the pipeline
- For income modeling: monthly income amount * savings allocation percentage, added to each projection period
- No external library needed. The math is straightforward with decimal arithmetic.

**Do NOT use `alpeb/go-finance`:** Last meaningful commit 2020, only 80 GitHub stars, adds unnecessary dependency for calculations that are trivial with shopspring/decimal.

---

## SimpleFIN Protocol: Holdings Data

### What the Protocol Provides

The SimpleFIN protocol includes a `holdings` array on account objects. Each holding has:

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `id` | string | Unique holding identifier | `"abc123"` |
| `created` | int64 | Unix timestamp | `1734719509` |
| `currency` | string | Currency code (may be empty) | `"USD"` or `""` |
| `cost_basis` | string | Original investment cost | `"90000.00"` |
| `description` | string | Full name of the holding | `"VANGUARD INDEX FUNDS S&P 500 ETF USD"` |
| `market_value` | string | Current market value | `"100000.00"` |
| `purchase_price` | string | Price per share at purchase | `"0.00"` |
| `shares` | string | Number of shares/units | `"100.03"` |
| `symbol` | string | Ticker symbol | `"VOO"` |

### What Needs to Change in the Existing Codebase

The current `internal/simplefin/client.go` uses `balances-only=1` query parameter, which suppresses transaction and holdings data. To get holdings:

1. **Remove or make `balances-only` conditional:** When syncing investment/crypto accounts, omit `balances-only=1` to receive the `holdings` array.
2. **Add `Holding` struct** to the SimpleFIN client:
   ```go
   type Holding struct {
       ID            string `json:"id"`
       Created       int64  `json:"created"`
       Currency      string `json:"currency"`
       CostBasis     string `json:"cost_basis"`
       Description   string `json:"description"`
       MarketValue   string `json:"market_value"`
       PurchasePrice string `json:"purchase_price"`
       Shares        string `json:"shares"`
       Symbol        string `json:"symbol"`
   }
   ```
3. **Add `Holdings []Holding` field** to the existing `Account` struct.
4. **Create a `holdings` table** in SQLite to store per-account holding snapshots.

### Crypto Aggregation via Holdings

SimpleFIN does not distinguish "crypto" from "investment" at the protocol level. Crypto accounts (Coinbase, Kraken, etc.) appear as regular accounts with holdings. The aggregation logic (summing all Coinbase wallets into one line) is purely application-side:

- Group accounts by `org.name` (or `org.domain`)
- For institutions flagged as crypto (user-configurable or auto-detected), sum balances across all accounts from that org
- Holdings data provides individual coin/token detail for drill-down

**Vanguard investment detail:** Vanguard accounts include holdings with `symbol` (e.g., "VOO", "VTSAX"), `shares`, and `market_value`. This enables showing per-fund breakdown in net worth drill-down without any additional data source.

### No New Library Needed

The existing `net/http` client in `internal/simplefin/client.go` and `encoding/json` handle the expanded response. No new dependency required -- just struct additions and a schema migration.

---

## New Frontend Libraries

### Alert Rule Builder: react-querybuilder

| Property | Value |
|----------|-------|
| **Library** | `react-querybuilder` |
| **Version** | 8.14.0 |
| **Purpose** | Visual expression builder for alert threshold rules |
| **React compat** | React 18+ (React 19 explicitly tested and supported) |
| **TypeScript** | Full type definitions included |
| **License** | MIT |

**Why react-querybuilder over alternatives:**

- **Not react-awesome-query-builder:** More feature-rich but heavier, complex config format, and its React 19 compatibility status is unclear. react-querybuilder is simpler and explicitly supports React 19.
- **Not a custom implementation:** Building a rule builder from scratch (dropdowns, operators, value inputs, group nesting) is 2-4 weeks of UI work with edge cases. react-querybuilder handles all of this out of the box.
- **Not Syncfusion Query Builder:** Commercial license required. Unnecessary for a single-user self-hosted app.

**How it fits the alert rules feature:**

The alert system needs rules like:
- "If Liquid Cash < $5,000, send email"
- "If Net Worth drops by > 5% in 7 days, send email"
- "If Savings + Checking > $50,000, send email"

react-querybuilder provides:
- **Custom fields:** Define buckets (Liquid, Savings, Investments, Net Worth) and individual accounts as selectable fields
- **Custom operators:** `>`, `<`, `>=`, `<=`, `crosses above`, `crosses below`, `changes by %`
- **Combinators:** AND/OR grouping for compound rules
- **Export to JSON:** The rule tree serializes to JSON, which the Go backend stores and evaluates
- **Fully customizable rendering:** Ships unstyled (or with compatibility packages). Style with Tailwind classes to match the existing UI.

**Integration pattern:**
```tsx
import { QueryBuilder, type RuleGroupType } from 'react-querybuilder';

const fields = [
  { name: 'liquid_cash', label: 'Liquid Cash', inputType: 'number' },
  { name: 'savings', label: 'Savings', inputType: 'number' },
  { name: 'net_worth', label: 'Net Worth', inputType: 'number' },
  // ... dynamically populated from accounts
];

const operators = [
  { name: '>', label: 'greater than' },
  { name: '<', label: 'less than' },
  { name: 'crosses_above', label: 'crosses above' },
  { name: 'crosses_below', label: 'crosses below' },
  { name: 'pct_change_gt', label: 'changes by more than %' },
];
```

The exported JSON rule tree is stored in the `alert_rules` table. The Go backend alert-checker goroutine deserializes and evaluates it against current balances.

### Projection and Drill-Down Charts: No New Library Needed

**Use recharts (already installed at v3.8.0).**

recharts already provides everything needed for v1.1 chart features:

| Chart Need | recharts Component | Technique |
|------------|-------------------|-----------|
| **Net worth drill-down historical** | `ComposedChart` + `Area` + `Line` | Stack multiple `<Area>` components for account breakdown, `<Line>` for net worth total |
| **Projected net worth** | `ComposedChart` with two data series | Solid `<Area>` for historical data, dashed `<Area strokeDasharray="5 5">` for projection |
| **Historical vs projection boundary** | `ReferenceLine` | Vertical reference line at "today" to visually separate actual from projected |
| **Growth rate sparklines** | `LineChart` (mini) | Tiny inline charts in panel cards showing 30-day trend |
| **Per-account drill-down** | `LineChart` or `AreaChart` | Individual account balance history with tooltips |

**Dashed line for projections:** recharts supports `strokeDasharray` on `<Line>` and `<Area>` components. Use `strokeDasharray="5 5"` on the projection series to visually distinguish forecast from historical data. This is a built-in SVG property, not a plugin.

**No need for a separate charting library.** recharts 3.x handles all v1.1 visualization needs.

---

## Installation (New Dependencies Only)

```bash
# Go backend -- add to go.mod
go get github.com/wneessen/go-mail@v0.7.1

# React frontend -- add to package.json
npm install react-querybuilder@^8.14.0
```

That is it. Two new dependencies total.

---

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| **Email sending** | go-mail v0.7.1 | net/smtp (stdlib) | Frozen, no new features, missing modern auth methods, no STARTTLS policy control |
| **Email sending** | go-mail v0.7.1 | SendGrid/Mailgun API | Adds external service dependency, requires account/API key, costs money, defeats self-hosted ethos |
| **Email sending** | go-mail v0.7.1 | gomail (go-gomail/gomail) | Abandoned since 2016, archived repository, no security patches |
| **Email sending** | go-mail v0.7.1 | emersion/go-smtp | Lower-level SMTP library (client+server), more code to write for simple sending. go-mail wraps the sending use case more ergonomically |
| **Projections math** | shopspring/decimal (existing) | alpeb/go-finance | Unmaintained (2020), tiny community, adds dep for trivial math |
| **Projections math** | shopspring/decimal (existing) | math.Pow with float64 | Float precision errors accumulate in financial projections. Unacceptable for a finance app |
| **Rule builder UI** | react-querybuilder 8.x | Custom implementation | 2-4 weeks of edge-case-heavy UI work for operator logic, group nesting, validation |
| **Rule builder UI** | react-querybuilder 8.x | react-awesome-query-builder | Heavier, complex config, unclear React 19 support |
| **Drill-down charts** | recharts 3.x (existing) | Nivo | Would add second charting library to bundle for no reason. recharts already handles all needed chart types |

---

## What NOT to Add

| Temptation | Why Resist | What to Do Instead |
|------------|-----------|-------------------|
| **A transactional email service SDK** (SendGrid, Mailgun, AWS SES) | External dependency, costs money, requires signup, self-hosted user expects local control | Use go-mail with SMTP directly to Protonmail Bridge or any user-configured SMTP server |
| **A financial projection library** (go-finance, etc.) | Compound interest is `PV * (1 + r/n)^(n*t)` -- one function with shopspring/decimal. No library needed | Write a `Projection` function in `internal/projections/` using existing shopspring/decimal |
| **A second charting library** (Chart.js, D3 direct, Nivo) | recharts 3.x already installed, handles all needed chart types including dashed projection lines | Use recharts ComposedChart with strokeDasharray for projections |
| **A notification framework** (gorush, ntfy client) | This project sends low-volume email alerts (~1-10/day max). A notification framework is overkill | Simple `notify.SendEmail()` function using go-mail |
| **A rule engine library** (grule-rule-engine, etc.) | Alert rules are simple threshold comparisons. A full rule engine adds complexity for "if balance < X" logic | Evaluate the react-querybuilder JSON tree with a small Go evaluator (~50-100 lines) |
| **@tanstack/react-query** | Listed in v1.0 research but NOT in current package.json -- the project uses direct fetch. Don't add mid-project unless refactoring data fetching globally | Continue with existing fetch pattern or add only if the alert/projection settings pages create enough fetch complexity to justify it |
| **cron library for alert checking** | Already have robfig/cron or a ticker loop for daily sync. Alert checking should run as part of the same sync cycle, not a separate scheduler | Check alert rules inside the existing sync goroutine after balance updates |

---

## Stack Patterns for v1.1 Features

**Email notification architecture:**
- `internal/notify/sender.go` -- interface: `type Sender interface { Send(to, subject, body string) error }`
- `internal/notify/smtp.go` -- implementation using go-mail, configured from SMTP settings in DB
- `internal/notify/mock.go` -- test double
- Alert checker calls `sender.Send()` when threshold crossed
- "Fire once on cross, once on recovery" state tracked per alert rule in DB (boolean `triggered` column)

**Alert rule evaluation flow:**
1. Daily sync completes, new balances stored
2. Alert checker loads all active rules from `alert_rules` table
3. For each rule, deserialize the react-querybuilder JSON tree
4. Evaluate against current balances using a recursive tree walker
5. Compare result with `last_triggered` state
6. If state changed (not triggered -> triggered, or triggered -> recovered), send email and update state

**Projection calculation flow:**
1. Frontend sends: per-account APY, reinvestment toggles, income amount, savings allocation, time horizon
2. Backend computes monthly projections using shopspring/decimal:
   - For each month: `balance = balance * (1 + monthlyRate)` if reinvesting
   - Add allocated income to applicable accounts
   - Sum all accounts for net worth projection
3. Return array of `{month, projectedBalance}` points
4. Frontend renders with recharts: historical (solid) + projected (dashed)

**Protonmail Bridge Docker pattern:**
```yaml
# docker-compose.yml addition
services:
  protonmail-bridge:
    image: shenxn/protonmail-bridge:latest
    volumes:
      - protonmail-data:/root
    # Only expose SMTP internally on Docker network
    # No host port mapping needed -- app container connects directly
```
The finance app container references `protonmail-bridge:1025` as SMTP host. User runs `docker exec -it protonmail-bridge /bin/bash` once to authenticate with Protonmail.

---

## Version Compatibility (New Additions)

| New Package | Compatible With | Notes |
|-------------|-----------------|-------|
| go-mail v0.7.1 | Go 1.25.0 | Requires Go 1.24+. Project uses 1.25. No conflicts with existing deps (no overlapping SMTP packages) |
| react-querybuilder 8.14.0 | React 19.2.4 | Minimum React 18. Explicitly tested with React 19. TypeScript types included. No peer dep conflicts with existing recharts or react-router-dom |
| react-querybuilder 8.14.0 | Tailwind CSS v4 | Ships unstyled by default -- style with Tailwind classes via `controlClassnames` prop. No CSS framework conflicts |

---

## Sources

- [wneessen/go-mail GitHub releases](https://github.com/wneessen/go-mail/releases) -- v0.7.1 confirmed, CVE-2025-59937 security fix, Go 1.24+ requirement (HIGH confidence)
- [go-mail pkg.go.dev](https://pkg.go.dev/github.com/wneessen/go-mail) -- API reference, auth methods: PLAIN, LOGIN, CRAM-MD5, SCRAM-SHA-1/256, XOAUTH2 (HIGH confidence)
- [go-mail wiki](https://github.com/wneessen/go-mail/wiki) -- Getting started, STARTTLS configuration (HIGH confidence)
- [react-querybuilder npm](https://www.npmjs.com/package/react-querybuilder) -- v8.14.0, React 19 support confirmed (HIGH confidence)
- [react-querybuilder docs](https://react-querybuilder.js.org/) -- TypeScript reference, custom fields/operators, export format (HIGH confidence)
- [SimpleFIN Protocol](https://www.simplefin.org/protocol.html) -- Account and holdings schema (HIGH confidence)
- [SimpleFIN GitHub protocol.md](https://github.com/simplefin/simplefin.github.com/blob/master/protocol.md) -- Authoritative spec source (HIGH confidence)
- [Protonmail Bridge SMTP settings](https://proton.me/support/imap-smtp-and-pop3-setup) -- 127.0.0.1:1025, STARTTLS, Bridge-generated password (HIGH confidence)
- [Protonmail Bridge Docker](https://github.com/shenxn/protonmail-bridge-docker) -- Docker container for headless Bridge (MEDIUM confidence -- community-maintained, not official Proton)
- [shopspring/decimal Pow issue #55](https://github.com/shopspring/decimal/issues/55) -- Fractional exponent limitation confirmed, PowWithPrecision workaround (HIGH confidence)
- [shopspring/decimal pkg.go.dev](https://pkg.go.dev/github.com/shopspring/decimal) -- v1.4.0, PowWithPrecision method documented (HIGH confidence)
- [recharts Dashed Line Chart example](https://recharts.github.io/en-US/examples/DashedLineChart/) -- strokeDasharray support confirmed (HIGH confidence)

---

*Stack research for: v1.1 feature additions (email, projections, crypto aggregation, alert rule builder)*
*Researched: 2026-03-15*
