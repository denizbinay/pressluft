Status: active
Owner: platform
Last Reviewed: 2026-02-20
Depends On: docs/vision-and-purpose.md, docs/spec-index.md
Supersedes: none

# User Requirements and Workflows

Pressluft is site-centric. Users manage WordPress projects and environments, not servers. Infrastructure is an implementation detail. All actions are scoped to a single site and must be isolated, reversible, and safe, even when multiple sites share the same machine.

---

## 1. Installation and Infrastructure

**User Goal:** Install Pressluft on a server and immediately start creating sites.

**Flow:**
- Provision a fresh Ubuntu server on Hetzner.
- Run the provided `install.sh` via curl to install Pressluft.
- The control plane bootstraps and hardens the local runtime and registers the first node.

**Must-Have Outcomes:**
- Hardened OS assumptions and secure defaults.
- Preconfigured, reproducible WordPress stack.
- Server capable of hosting multiple isolated sites.
- No manual server tuning required after installation.

Optional: Advanced users may connect external infrastructure later. This is not required for the MVP.

---

## 2. Site Creation

**User Goal:** Launch a new WordPress site instantly, with or without a custom domain.

**Flow:**
- Open the dashboard and click “Create Site”.
- Receive a working preview URL immediately.

**Must-Have Outcomes:**
- Automatic preview domain (platform-provided or wildcard-based).
- Automatic TLS when domain is configured.
- Isolated system user and PHP process per site.
- No DNS required for initial development.
- Real domains can be attached at any time.

Site creation must feel instant and frictionless.

---

## 3. Environments, Cloning, and Review

**User Goal:** Create safe working copies for development and client review.

**Flow:**
- Create a clone or staging environment.
- Work and share via a unique URL.
- Promote changes back to production intentionally.

**Must-Have Outcomes:**
- One-click full clone (files and database).
- Each environment has isolated runtime paths (release/current + shared state) under a site-keyed filesystem root, plus its own database and URL.
- Clear visual separation between environments.
- Optional expiration for temporary clones.
- Automatic backup before any destructive action.

Environments must be disposable, fast to create, and safe to remove.

---

## 4. Safe Promotion and Drift Protection

**User Goal:** Push changes to production without overwriting critical live data.

**Must-Have Outcomes:**
- Strict rule: changes move up, live data remains protected.
- Preset-based selective sync (e.g., protect content, protect commerce).
- Automatic detection of production changes since clone creation.
- Clear warnings when drift exists.
- Mandatory backup before pushback.
- Atomic promotion with instant rollback.

Users must never fear breaking production.

---

## 5. Deployment and Updates

**User Goal:** Apply updates and changes safely.

**Must-Have Outcomes:**
- Updates applied in a non-production environment first.
- Snapshot before update.
- Instant rollback capability.
- No maintenance mode or visible downtime during promotion.

Git integration is optional but supported. Builder-only workflows must work equally well.

---

## 6. Backups and Recovery

**User Goal:** Restore any site or environment confidently.

**Must-Have Outcomes:**
- Automated scheduled backups.
- On-demand manual snapshots.
- Site-level restore without affecting other sites.
- Clear retention policy.
- Restore to production or any environment.

---

## 7. Migration

**User Goal:** Import existing WordPress sites safely.

**Must-Have Outcomes:**
- Import from external hosts.
- Safe handling of serialized database data.
- Automatic URL rewriting.
- Minimal downtime during cutover.

---

Pressluft removes infrastructure anxiety. WordPress work becomes environment-driven, predictable, isolated, and fully reversible.
