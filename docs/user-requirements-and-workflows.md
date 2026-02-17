# User Requirements and Workflows

Pressluft is site-centric. Users manage WordPress projects, not servers. Infrastructure is a capacity layer that stays in the background. All actions are scoped to a single site and must be isolated, reversible, and safe even when multiple sites share one server.

---

## 1. Infrastructure Setup

**User Goal:** Connect cloud infrastructure and make it ready for sites.

**Flow:**
- Connect provider API key.
- Create first server.
- Platform installs and hardens the stack.

**Must-Have Outcomes:**
- Hardened OS and secure defaults.
- Preconfigured, maintainable WordPress stack.
- Server ready to host multiple isolated sites.
- No manual SSH provisioning or firewall setup.

---

## 2. Site Creation

**User Goal:** Launch a new WordPress site instantly, with or without a real domain.

**Flow:**
- Click “Create Site”.
- Assign to an existing server.
- Receive an automatic preview URL.

**Must-Have Outcomes:**
- Temporary platform domain provided by default.
- Automatic TLS on preview domains.
- Isolated system user and PHP pool per site.
- No DNS required during development.

Real domains can be attached later.

---

## 3. Staging, Cloning, and Review

**User Goal:** Create safe working environments for development or client review.

**Flow:**
- Create staging or dev clone.
- Work or review via unique URL.
- Promote changes intentionally.

**Must-Have Outcomes:**
- One-click full clone (files + database).
- Strict rule: Code flows up, content flows down.
- Selective table sync to prevent overwriting orders or form data.
- Automatic backup before overwrite.
- Clear environment labeling to avoid mistakes.
- Support for both Git-based and builder-only workflows.

---

## 4. Deployment and Updates

**User Goal:** Deploy changes without downtime or risk.

**Must-Have Outcomes:**
- Atomic release switching with instant rollback.
- No maintenance mode or white screens.
- Optional light visual regression before promotion.
- Staged updates for core, themes, and plugins.
- Bulk update capability for agencies.

---

## 5. Backups and Recovery

**User Goal:** Restore confidently at any time.

**Must-Have Outcomes:**
- Automated daily off-site backups.
- On-demand snapshots.
- Site-level restore without affecting other sites.
- Clear retention policy.

---

## 6. Migration and Rebalancing

**User Goal:** Move sites safely between servers or from external hosts.

**Must-Have Outcomes:**
- Server-to-server transfer.
- Safe handling of serialized data.
- Zero-downtime cutover.
- Ability to move a single site without impacting others.

Pressluft removes infrastructure anxiety. WordPress work becomes predictable, isolated, and fully reversible.
