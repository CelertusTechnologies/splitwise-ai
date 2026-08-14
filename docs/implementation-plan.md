# Nivra — Market Analysis & Implementation/Deployment Plan

Status: draft plan, written from repo analysis + external market research. Where a decision could go multiple ways, I picked the default marked **(chosen)** and explained why — flag anything you want changed and I'll adjust before we start building.

---

## 1. Competitive landscape

Reviewed: Splitwise (market leader), Tricount (bunq-owned, free, no-account-needed links), Settle Up, plus several 2026 challenger apps and open-source clones (Splito, settle-it, etc.).

### 1.1 Feature matrix

| Feature | Splitwise | Tricount | Settle Up | Nivra today |
|---|---|---|---|---|
| Equal / unequal / % / shares split | ✅ | ✅ | ✅ | ✅ (schema only, no API yet) |
| Groups (trips/home/etc.) | ✅ | ✅ (per-"tricount", link-based) | ✅ | ✅ (schema only) |
| Friends / 1:1 expenses without a group | ✅ | ❌ (always a tricount) | ✅ | ❌ **gap** |
| Debt simplification ("simplify debts") | ✅ (flagship feature, network-flow algorithm) | ➖ basic | ✅ | ❌ **gap** — this is the single most-expected feature |
| Recurring expenses | ✅ | ❌ | ➖ | ❌ **gap** |
| Multi-currency + FX conversion | ✅ (Pro) | ✅ (free) | ✅ | Currency stored per-row, no conversion engine |
| Receipt scanning / OCR | ✅ (Pro) | ❌ | ➖ limited | ❌ (planned V2 per your own architecture doc) |
| Activity feed / notifications | ✅ | ✅ | ✅ | Schema only, no API |
| No-account participants (invite link, guest pays later) | ➖ | ✅ (Tricount's differentiator) | ➖ | ❌ |
| Offline mode + sync | ✅ | ✅ | ✅ | N/A (web-only today) |
| Free tier daily-expense cap + ads | ✅ (3/day free) | ✅ none (fully free) | ✅ (ads, no cap) | You control this — no monetization built yet |

**Takeaway:** your existing DB schema (`backend/migrations/000001_init.up.sql`) already matches or exceeds what public write-ups of Splitwise's own schema describe — it's not the bottleneck. The bottleneck is entirely that groups/expenses/settlements have **zero service/handler code** yet. Debt simplification is the one algorithmic piece competitors treat as their star feature and you have nothing for it yet — that should be a first-class roadmap item, not an afterthought bolted onto settlements.

### 1.2 UI patterns by user role

Public case studies (Splitwise redesigns on Behance/UX Planet) and competitor screenshots converge on the same role model everywhere:

- **Regular member**: bottom-nav (mobile) / sidebar (web) with *Groups*, *Friends*, *Activity*, *Account*. Primary action is a floating "+ Add expense" button, always one tap away. Group detail page shows: expense list (newest first) → balance summary at top ("you owe / you're owed") → "Settle up" CTA.
- **Group admin/owner**: same screens as member plus a settings gear on the group page → manage members (remove/promote), edit group name/photo/currency, regenerate/revoke invite link, archive/delete group. This is inline in the group view, not a separate app — no competitor ships a distinct "admin mode" UI at the group level.
- **Platform admin** (your side, not user-facing): none of the consumer apps expose this — it's an internal-only panel. For Nivra this maps to `users.role = 'admin'` in your schema and should be a separate route tree (e.g. `/admin/*`) gated by JWT role claim, not a toggle inside the consumer UI.
- **First-run/no-account participant** (Tricount's trick): a person added to an expense by email/phone who never signs up sees a read-only, tokenized link showing just their balance — worth adding since it removes Splitwise's biggest complaint ("everyone has to install the app").

### 1.3 Schema patterns confirmed across sources

Every public schema write-up and clone converges on: `users → group_memberships → groups`, `expenses → expense_shares`, a `settlements`/`payments` table, and either a materialized `balances` table or an on-the-fly SUM over `expense_shares`/`settlements`. Your schema already does the latter (computed balances, no stored balance table) — correct choice, avoids write-amplification bugs. No source described a materially different structure worth adopting instead of yours.

---

## 2. Schema changes to make before building on top of it

Your `expenses` table currently has `group_id UUID NOT NULL` — every expense must belong to a group. To support 1:1 "friend" expenses (a top-3 expected feature per §1.1) without a costly later migration:

- **(chosen)** Make `expenses.group_id` nullable, add a `friendships` table (`user_id_a`, `user_id_b`, `status: pending|accepted|blocked`, unique ordered pair), and for direct expenses set `group_id = NULL` + a new `expense_participants` link replacing group membership as the source of truth for "who's involved" when there's no group. Balance queries then union group-scoped and friend-scoped expenses per user pair.
- Add a lightweight `fx_rates(base_currency, quote_currency, rate, as_of_date)` table — needed the moment you support multi-currency conversion, and cheap to add now while the migration file is still small.
- Recurring expenses: no schema change needed yet — defer to Phase 3, model as a `recurring_expense_templates` table that spawns real `expenses` rows on a schedule (worker job), rather than a special-case flag on `expenses` itself.

I'm treating these as day-1 additions rather than a v2 migration since retrofitting nullable `group_id` after expense data exists is far more disruptive than adding it now, before any expense rows exist.

---

## 3. Feature roadmap

**(chosen) Phase priority: core Splitwise parity before any AI work** — this matches what your own `docs/architecture.md` §19 already states ("Do not implement AI in V1"), and there's no working core product yet to layer AI onto.

### Phase 1 — Core product (backend + frontend, no AI)
1. Groups: create/list/detail/edit/archive, invite-code generate+join, member management (promote/remove)
2. Friends: send/accept friend request, 1:1 balance view (needs the schema change above)
3. Expenses: CRUD for all 4 split methods, categories, edit/delete with recalculation
4. **Settlement engine**: balance calculation per group and per friend pair, debt-simplification (network-flow / greedy min-cash-flow algorithm — O(V²), well-documented, see §1.1), record settlements (cash/bank/UPI)
5. Wire the real dashboard: replace `dashboard-shell.tsx`'s hardcoded fixtures with actual API calls (summary, recent activity, balances)
6. Add a Next.js route guard/middleware so `/dashboard` and friends require a valid token (currently anyone can browse it unauthenticated)
7. Notifications: in-app list + mark-read (defer email/SMS/push providers to Phase 2 unless you want them sooner)

### Phase 2 — Production hardening + VPS launch
1. Fix the Go version mismatch: `backend/go.mod` requires `go 1.25`, but `backend/Dockerfile` and `.github/workflows/ci.yml` both build with Go **1.23** — this will break the Docker build the moment any 1.24+ syntax is used. Align all three to one version before deploying.
2. Move frontend tokens from `localStorage` to httpOnly cookies; add refresh-on-401 interceptor to `frontend/lib/api.ts`
3. Real email sending (replace the dev-mode "token returned in API response" behavior) — needs an SMTP/email API credential (see §5)
4. Rate limiting via the already-provisioned Redis; idempotency-key enforcement (table already exists, unused)
5. VPS provisioning, Docker Compose production profile, Nginx + HTTPS, backups, basic monitoring (full detail in §4)
6. First real automated tests — currently there are **zero** anywhere in the repo despite CI being wired to run them

### Phase 3 — AI differentiators (V2, as your own docs specify)
Receipt OCR → itemized split suggestions, natural-language expense entry ("split ₹800 dinner with Raj and Priya"), smart categorization, spending insights. Needs an LLM API key and a storage provider for receipt images — deliberately sequenced last since none of it matters until core expense-tracking works.

---

## 4. VPS deployment plan

Your `docker-compose.yml` already defines postgres/redis/backend/frontend services — this is 70% of a VPS deployment already. What's missing is everything around it: reverse proxy, TLS, hardening, backups, and a deploy flow. Note your `docs/architecture.md` describes a target of AWS ECS Fargate/RDS/CloudFront — deploying to a single VPS instead is a deliberate, sensible divergence for this stage (much cheaper, simpler, fine up to tens of thousands of users), not a mistake to reconcile; migrate to managed cloud later if/when scale demands it.

### 4.1 Sizing
**(chosen) Recommendation: 2 vCPU / 4GB RAM**, NVMe storage, Ubuntu 24.04 LTS. Rationale: Postgres + Redis + Go API + Next.js + Nginx side-by-side realistically need 2-3GB steady-state, plus headroom for `npm run build` if you ever build on-box. This fits a ~$20-24/mo droplet/instance (DigitalOcean, Hetzner CX-family, Linode all offer this tier). If your VPS is smaller (1 vCPU/2GB), it's workable but build the frontend image in CI and just `docker pull`+run on the VPS rather than building on it, and add a swap file.

### 4.2 Software to install on the VPS
- Docker Engine + Docker Compose plugin (official `get.docker.com` script or distro repo)
- Nginx (reverse proxy + TLS termination) — or swap for Caddy if you'd prefer automatic HTTPS with less config; Nginx assumed below since it's the most common/documented path
- Certbot (`python3-certbot-nginx`) for Let's Encrypt certificates — only needed once a domain is pointed at the box
- UFW (firewall: allow 22/tcp, 80/tcp, 443/tcp only; deny everything else, including direct access to 5432/6379/8080/3000 from outside)
- fail2ban (SSH brute-force protection)
- unattended-upgrades (automatic security patches)
- git (to pull the repo, or a CI-based image-push workflow — see §4.4)

### 4.3 Production docker-compose changes needed
- Remove host port publishing for `postgres` (5432) and `redis` (6379) — they should only be reachable on the internal Docker network, never exposed to the public interface
- Add `restart: unless-stopped` to every service
- Add resource limits (`mem_limit`/`cpus`) so one runaway container can't OOM the box
- Real secrets: generate actual random 32+ byte values for `JWT_ACCESS_SECRET`/`JWT_REFRESH_SECRET` (currently placeholder `change-me-...` strings) — this is a hard blocker, not optional, before this touches the internet
- Nginx as an added `nginx` service (or host-level Nginx outside Compose) proxying `/` → frontend:3000, `/api/*` → backend:8080

### 4.4 Deploy flow (simple, appropriate for current scale)
1. SSH to VPS → `git clone`/`git pull` the repo → `docker compose -f docker-compose.prod.yml up -d --build`
2. A small `deploy.sh` script wrapping that pull+rebuild+restart, run manually or via a GitHub Actions SSH step on push to `main`
3. Migrations run as a one-off `docker compose run backend <migrate-up-command>` step before restarting the API container
4. (Later, once traffic justifies it) move to registry-based deploys (build in CI, push image, VPS just pulls) instead of building on the VPS itself

### 4.5 Domain & HTTPS
Works whether or not you have a domain yet:
- **With a domain**: point an `A` record at the VPS IP → Nginx server block for that domain → `certbot --nginx` for auto-renewing HTTPS
- **Without one yet**: serve over the raw IP on HTTP initially (fine for internal testing, **not** for real users — auth tokens over plain HTTP are a real risk), add the domain+Certbot step the moment one is purchased. I'd avoid inviting real users before this step is done.

### 4.6 Backups & monitoring (lightweight version of what `docs/architecture.md` §17 describes for AWS)
- Nightly `pg_dump` cron job → compressed, rotated (7 daily + 4 weekly), copied off-box (e.g. to S3/Backblaze — needs a credential, see §5)
- Uptime check (UptimeRobot free tier or similar) hitting `/health`
- `docker compose logs` + basic `journalctl`/disk-space alerts to start; defer Sentry/Prometheus/OTel until Phase 2 hardening is otherwise done

---

## 5. Access / credentials checklist

Nothing below blocks starting Phase 1 feature work (that's all local/Docker-Compose). These become necessary as noted:

| Needed for | What | When |
|---|---|---|
| VPS deploy | SSH access (IP, user, key or password) to the box | Start of Phase 2 |
| HTTPS | Domain name + registrar/DNS access (or tell me to proceed IP-only for now) | Start of Phase 2, before inviting real users |
| Real auth emails | SMTP or email-API credential (SES/SendGrid/Resend/Postmark — your call) | Start of Phase 2 |
| Receipt attachments | S3-compatible object storage keys (AWS S3, Backblaze B2, or Cloudflare R2) | Whenever attachments/receipts are built (Phase 2 tail / Phase 3) |
| Off-box backups | Same object storage, or a separate bucket | Phase 2 |
| AI features | An LLM provider API key (Anthropic/OpenAI/etc. — no SDK chosen yet) | Phase 3 |
| Optional: SMS/WhatsApp/push | Twilio/MSG91/FCM credentials | Only if you want those channels beyond in-app + email |

---

## 6. Immediate next steps

1. Confirm or correct the defaults marked **(chosen)** above (VPS size, friends-feature schema change, phase ordering) — cheap to change now, expensive after code is built on top
2. Fix the Go 1.23 vs 1.25 version mismatch (§3, Phase 2 item 1) — quick, and will otherwise silently break the first VPS build
3. Start Phase 1 backend work: groups → expenses → settlement engine (in that dependency order, since expenses need groups and settlements need expenses)
4. Once you're ready to hand over VPS access, share SSH details and I'll handle provisioning per §4
