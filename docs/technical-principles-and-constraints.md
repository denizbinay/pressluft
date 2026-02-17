# Technical Principles and Constraints

### 1. High-Level System Topology

The system is strictly divided into two planes to ensure security and blast-radius containment.

#### **A. The Control Plane (The Brain)**
*   **Role:** Stores state, manages secrets, orchestrates jobs, and exposes the API/UI.
*   **Infrastructure:** A highly available Hetzner Cloud VPS cluster (or dedicated management project).
*   **Components:**
    *   **API/Dashboard:** A monolithic **Go** application (using Fiber/Gin) for speed and type safety.
    *   **State Store:** **PostgreSQL**. The "Single Source of Truth" for site metadata, server inventory, and job history.
    *   **Job Broker:** **Redis** (persistent AOF) handling task queues via **Asynq**.
*   **Network:** Ingress restricted to Cloudflare/Load Balancer IPs. No direct access from Worker Nodes.

#### **B. The Data Plane (The Muscle)**
*   **Role:** Host the WordPress sites. Dumb execution units.
*   **Infrastructure:** Hetzner Cloud Servers (Worker Nodes) connected via a **Private Network (vSwitch/Cloud Network)**.
*   **Components:**
    *   **Agent:** A lightweight **Go binary** running as `root` (systemd service). It polls the Control Plane for work via gRPC/HTTPS (mTLS).
    *   **Stack:** OpenLiteSpeed, MariaDB, Redis, PHP-FPM (via LSAPI).
*   **Network:** **Egress-Only**. Inbound ports 80/443 (Web) and 22 (Bastion only). The Agent dials *out* to the Control Plane.

---

### 2. Site Isolation Model (The "No-Container" Contract)

Since Docker/Kubernetes are forbidden on workers, we leverage **Linux Native Isolation** features to ensure multi-tenant security.

*   **User Separation:**
    *   Every site is assigned a unique Linux user (e.g., `site_ax9z`).
    *   PHP processes run strictly as this user via **LSAPI** (LiteSpeed SAPI).
    *   Strict `chown`/`chmod` ensures `site A` cannot read `site B`'s files.
*   **Filesystem Jails (Namespace Containers):**
    *   We utilize **OpenLiteSpeed’s Native Namespace Containers**. This uses Linux namespaces to provide a read-only view of the OS (`/bin`, `/lib`) and mounts only the specific site’s `/var/www/site_id` as writable.
    *   **Benefit:** Prevents `shell_exec` exploits from traversing the filesystem without the overhead of Docker images.
*   **Resource Guardrails:**
    *   **Systemd Slices / Cgroups:** The Agent spawns site processes within systemd slices to enforce CPU/RAM limits, preventing "noisy neighbor" issues.
*   **Database Isolation:**
    *   Single MariaDB instance per server (for efficiency), but distinct credentials and database names per site.
    *   **Drift Prevention:** The Agent periodically rotates DB passwords and updates `wp-config.php` automatically.

---

### 3. State Machines & Orchestration

To guarantee "boring reliability," every lifecycle action is modeled as a **Finite State Machine (FSM)** stored in Postgres.

#### **A. The Job Orchestration Flow**
1.  **Intent:** User clicks "Deploy".
2.  **State Transition:** DB row transitions from `ACTIVE` to `DEPLOY_PENDING`.
3.  **Job Creation:** Control Plane pushes a serialized job (`DeployJob{SiteID, CommitHash}`) to Redis.
4.  **Execution:**
    *   Worker Agent polls queue, picks up job.
    *   Agent executes atomic steps locally.
    *   Agent streams logs/heartbeats back to Control Plane.
5.  **Completion:** Agent reports `SUCCESS`. Control Plane transitions DB to `ACTIVE`.

#### **B. Failure Handling & Rollbacks**
*   **Idempotency:** All Agent commands must be idempotent. Running `EnsureUser(site_id)` twice results in the same state without error.
*   **Heartbeats:** If an Agent stops heartbeating during a job (e.g., server crash), the Control Plane marks the job `STALLED` and alerts an operator. It does *not* auto-retry destructive actions.
*   **Atomic Deployments:**
    *   Code is deployed to `/var/www/site/releases/<timestamp>`.
    *   Symlink `/var/www/site/current` is switched only after health checks pass (HTTP 200 on localhost).
    *   **Rollback:** If health checks fail, the symlink is never switched. If the site breaks post-switch, a "Rollback" job simply updates the symlink to the previous timestamp.

#### **C. Concurrency Control**
*   **Fencing Tokens:** Every job carries a monotonic version number. The Agent rejects jobs with a version lower than the site's current state to prevent race conditions (e.g., a delayed "Delete" job arriving after a "Restore" job).
*   **Site Locking:** A site in `DEPLOYING` state rejects all other mutation requests (Backup, Clone, Config Change).

---

### 4. Key Workflows (End-to-End)

#### **Infrastructure Setup (Server Provisioning)**
1.  **API Call:** User connects Hetzner API Key.
2.  **Terraform:** Control Plane runs `terraform apply` to provision the VM, Firewalls, and Private Network.
3.  **Ansible:** Runs against the new IP to harden OS (Fail2Ban, UFW), tune Sysctl, and install the **Pressluft Agent**.
4.  **Handshake:** Agent starts, generates a unique mTLS key, and registers itself with the Control Plane.

#### **Site Creation & Cloning**
1.  **Allocation:** Control Plane selects the server with the lowest load average.
2.  **Instruction:** Agent receives payload.
3.  **Execution:**
    *   Creates Linux User.
    *   Creates DB & User (MariaDB).
    *   Configures OLS VHost (via template).
    *   Installs WordPress Core (via WP-CLI).
    *   *For Cloning:* Streams database dump from Source to Target via **Private Network** (bypassing public internet for speed/security). Run `wp search-replace`.

#### **Migrations & Rebalancing**
*   **Floating IPs:** For zero-downtime server migrations, we leverage Hetzner Floating IPs. To move a site, we sync data to the new server, put the site in read-only mode, do a final sync, and switch the OLS listener/Floating IP routing.

---

### 5. Stack Contract & Security Model

*   **Operating System:** Ubuntu 24.04 LTS (Minimal).
*   **Web Server:** OpenLiteSpeed (Stable). Configured via template files managed by the Agent (no manual GUI changes).
*   **Database:** MariaDB 10.11+ (LTS). Tuned for InnoDB performance (Buffer Pool = 70% RAM).
*   **Cache:** Redis (Local socket connection for performance). Isolated via ACLs (key prefixes per site).
*   **WP-CLI:** The **exclusive** method for WordPress operations. No direct DB queries from the Agent.
*   **Backup Boundary:**
    *   **Files:** Restic backups to Hetzner Storage Box (off-site).
    *   **DB:** `mysqldump` with `--single-transaction` streamed to Storage Box.
    *   **Retention:** Defined in Control Plane (e.g., 7 daily, 4 weekly).

---

### 6. Architectural Principles & Forbidden Patterns

These principles must be enforced in code review to maintain the system's integrity.

1.  **The "No SSH" Rule (Production):** The Control Plane **never** SSHs into workers to run jobs. SSH is for emergency break-glass debugging by humans only. The Agent **pulls** jobs.
2.  **Code Up, Content Down:** Deployment tools push code *up* to production. Sync tools pull content (DB/Uploads) *down* to staging. Never overwrite production content with staging data.
3.  **Immutable Configuration:** No "tweaking" configs on the server. If Nginx/OLS config needs changing, update the template in the repo and redeploy the Agent. Drift detection must alert on manual changes.
4.  **No Docker on Workers:** Do not install Docker daemon on worker nodes. It adds networking complexity (bridge networks, NAT) and security overhead that violates the "boring" requirement.
5.  **Strict Egress-Only:** Worker nodes have `ufw default deny incoming`. They only accept established connections.
6.  **Database as a Black Box:** The Agent never runs raw SQL against WP databases. It uses `wp-cli` for everything (user creation, search-replace, option updates) to ensure serialization safety.

This architecture provides a "rigid but robust" foundation. It trades the flexibility of containers for the raw performance and simplicity of bare-metal Linux isolation, perfectly matching the "Pressluft" vision.
