package devserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"pressluft/internal/api"
	"pressluft/internal/audit"
	"pressluft/internal/auth"
	"pressluft/internal/backups"
	"pressluft/internal/environments"
	"pressluft/internal/jobs"
	"pressluft/internal/metrics"
	"pressluft/internal/nodes"
	"pressluft/internal/providers"
	"pressluft/internal/sites"
	"pressluft/internal/store"
)

const dashboardHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Pressluft Dashboard</title>
  <style>
    :root {
      color-scheme: dark;
      --ink: #e9f0f7;
      --ink-soft: #9fb1c2;
      --line: #2f3e4c;
      --bg-a: #091421;
      --bg-b: #0d1b2a;
      --surface: #132131e0;
      --surface-2: #101c2a;
      --brand: #1ba784;
      --brand-2: #2c91d9;
      --danger: #ff6b7c;
      --ok: #4dd38c;
      --shadow: 0 24px 48px rgba(0, 0, 0, 0.45);
    }

    * { box-sizing: border-box; }

    body {
      margin: 0;
      min-height: 100vh;
      font-family: "IBM Plex Sans", "Avenir Next", "Segoe UI", sans-serif;
      color: var(--ink);
      background:
        radial-gradient(1200px 700px at -10% -15%, #12334d 0%, transparent 62%),
        radial-gradient(950px 520px at 110% -10%, #114234 0%, transparent 60%),
        linear-gradient(145deg, var(--bg-a), var(--bg-b));
    }

    .wrap {
      max-width: 980px;
      margin: 0 auto;
      padding: 28px 18px 44px;
    }

    .shell {
      background: var(--surface);
      backdrop-filter: blur(6px);
      border: 1px solid #294055;
      border-radius: 18px;
      box-shadow: var(--shadow);
      padding: 22px;
      animation: rise 340ms ease-out;
    }

    .head {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      margin-bottom: 18px;
    }

    .subsite-nav {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
      gap: 8px;
      margin-bottom: 14px;
    }

    .subsite-link {
      display: inline-block;
      border: 1px solid var(--line);
      border-radius: 10px;
      padding: 8px 10px;
      color: var(--ink);
      text-decoration: none;
      background: #0f1a27;
      text-align: center;
      font-weight: 600;
    }

    .subsite-link:hover {
      border-color: #3d5b75;
      background: #122235;
    }

    .subsite-link.active {
      border-color: #2f8ec9;
      background: linear-gradient(125deg, #163149, #183043);
    }

    h1 {
      margin: 0;
      font-size: 1.25rem;
      letter-spacing: 0.02em;
    }

    .muted { color: var(--ink-soft); margin: 2px 0 0; }

    .hidden { display: none !important; }

    .btn {
      border: 0;
      border-radius: 999px;
      padding: 10px 16px;
      font-weight: 600;
      cursor: pointer;
      background: linear-gradient(120deg, var(--brand), var(--brand-2));
      color: #fff;
    }

    .btn.ghost {
      background: #142433;
      border: 1px solid var(--line);
      color: var(--ink);
    }

    .panel {
      border: 1px solid var(--line);
      border-radius: 14px;
      background: var(--surface-2);
      padding: 14px;
    }

    .grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
      gap: 10px;
      margin-bottom: 14px;
    }

    .metric {
      border: 1px solid var(--line);
      border-radius: 12px;
      padding: 12px;
      background: var(--surface-2);
    }

    .metric b { display: block; font-size: 1.3rem; margin-bottom: 4px; }

    form {
      display: grid;
      gap: 10px;
      max-width: 520px;
    }

    label { display: grid; gap: 6px; font-weight: 600; }

    input, select {
      width: 100%;
      border-radius: 10px;
      border: 1px solid var(--line);
      padding: 10px;
      font: inherit;
      color: var(--ink);
      background: #0f1925;
    }

    input:focus-visible, select:focus-visible, .btn:focus-visible {
      outline: 2px solid #59b6ff;
      outline-offset: 2px;
    }

    .split {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
      gap: 14px;
      margin-bottom: 14px;
    }

    .inline {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
      gap: 8px;
    }

    .success {
      color: var(--ok);
      font-weight: 600;
      min-height: 1.2em;
      margin-top: 4px;
    }

    table {
      width: 100%;
      border-collapse: collapse;
      margin-top: 8px;
      font-size: 0.93rem;
    }

    th, td {
      border-top: 1px solid var(--line);
      padding: 10px 8px;
      text-align: left;
    }

    .job-link {
      border: 0;
      padding: 0;
      margin: 0;
      background: transparent;
      color: var(--brand-2);
      font: inherit;
      cursor: pointer;
      text-align: left;
    }

    .job-link:hover { color: #63b9ff; }

    .actions-cell { width: 64px; position: relative; text-align: right; }

    .icon-btn {
      border: 1px solid var(--line);
      border-radius: 10px;
      background: #142433;
      color: var(--ink);
      cursor: pointer;
      min-width: 34px;
      min-height: 30px;
      font-size: 18px;
      line-height: 1;
    }

    .icon-btn:hover { border-color: #3d5b75; }

    .actions-menu {
      position: absolute;
      right: 8px;
      top: 42px;
      z-index: 2;
      min-width: 190px;
      border: 1px solid var(--line);
      border-radius: 10px;
      background: #0f1925;
      padding: 6px;
      display: grid;
      gap: 6px;
      box-shadow: var(--shadow);
    }

    .actions-menu button {
      border: 1px solid var(--line);
      border-radius: 8px;
      background: #152536;
      color: var(--ink);
      text-align: left;
      font: inherit;
      cursor: pointer;
      padding: 8px;
    }

    .actions-menu button:hover { border-color: #3d5b75; }

    th { color: var(--ink-soft); font-size: 0.82rem; text-transform: uppercase; letter-spacing: 0.04em; }

    .status { font-weight: 700; text-transform: capitalize; }
    .status.running { color: var(--brand-2); }
    .status.queued { color: #f0bc53; }
    .status.pending { color: #f0bc53; }
    .status.succeeded { color: var(--ok); }
    .status.completed { color: var(--ok); }
    .status.failed { color: var(--danger); }
    .status.expired { color: var(--ink-soft); }

    .error {
      color: var(--danger);
      font-weight: 600;
      min-height: 1.2em;
      margin-top: 4px;
    }

    .timeline {
      list-style: none;
      margin: 6px 0 0;
      padding: 0;
      display: grid;
      gap: 8px;
    }

    .timeline li {
      border: 1px solid var(--line);
      border-radius: 10px;
      background: #0d1824;
      padding: 10px;
    }

    .timeline strong {
      display: inline-block;
      margin-right: 8px;
      text-transform: capitalize;
    }

    .timeline time {
      color: var(--ink-soft);
      font-size: 0.88rem;
    }

    @keyframes rise {
      from { transform: translateY(8px); opacity: 0; }
      to { transform: translateY(0); opacity: 1; }
    }
  </style>
</head>
<body>
  <main class="wrap">
    <section class="shell">
      <div class="head">
        <div>
          <h1>Pressluft Operator Console</h1>
          <p class="muted">Wave 5.6 nodes-first dashboard realignment is in progress.</p>
        </div>
        <button id="logout" class="btn ghost hidden" type="button">Logout</button>
      </div>

      <section id="auth-panel" class="panel">
        <h2>Sign in</h2>
        <p class="muted">Use the default local operator credentials to unlock dashboard APIs.</p>
        <form id="login-form">
          <label>Email <input id="email" name="email" type="email" required value="admin@pressluft.local"></label>
          <label>Password <input id="password" name="password" type="password" required value="pressluft-dev-password"></label>
          <button class="btn" type="submit">Login</button>
          <p id="auth-error" class="error" aria-live="polite"></p>
        </form>
      </section>

      <section id="dashboard" class="hidden">
        <nav id="subsite-nav" class="subsite-nav" aria-label="Dashboard sections">
          <a class="subsite-link" data-nav="overview" href="/">Overview</a>
          <a class="subsite-link" data-nav="providers" href="/providers">Providers</a>
          <a class="subsite-link" data-nav="nodes" href="/nodes">Nodes</a>
          <a class="subsite-link" data-nav="sites" href="/sites">Sites</a>
          <a class="subsite-link" data-nav="jobs" href="/jobs">Jobs</a>
        </nav>

        <section class="panel" data-subsite="providers" id="subsite-providers" style="margin-bottom: 14px;">
          <div class="head">
            <h2>Providers</h2>
            <p id="providers-title" class="muted">Connect provider bearer credentials before creating nodes.</p>
          </div>
          <form id="provider-connect-form">
            <div class="inline">
              <label>Provider
                <select id="provider-id">
                  <option value="hetzner">Hetzner Cloud</option>
                </select>
              </label>
              <label>API Token
                <input id="provider-api-token" name="api_token" type="password" required placeholder="Bearer token from Hetzner Cloud Console">
              </label>
            </div>
            <button class="btn" type="submit">Connect Provider</button>
            <p id="provider-success" class="success" aria-live="polite"></p>
            <p id="provider-error" class="error" aria-live="polite"></p>
          </form>
          <table>
            <thead>
              <tr>
                <th>Provider</th>
                <th>Status</th>
                <th>Secret</th>
                <th>Capabilities</th>
                <th>Guidance</th>
              </tr>
            </thead>
            <tbody id="providers-body"></tbody>
          </table>
        </section>

        <section class="panel" data-subsite="nodes" id="subsite-nodes" style="margin-bottom: 14px;">
          <div class="head">
            <h2>Nodes</h2>
            <p id="nodes-title" class="muted">Runtime node inventory and local-node readiness.</p>
          </div>
          <p id="local-node-readiness" class="muted">Local node readiness unknown.</p>
          <section class="split" style="margin-top: 10px; margin-bottom: 10px;">
            <article class="panel">
              <h3>Create Node</h3>
              <p class="muted">Create a node from a connected provider.</p>
              <form id="node-provider-form">
                <label>Provider
                  <select id="node-provider-id">
                    <option value="hetzner">Hetzner Cloud</option>
                  </select>
                </label>
                <label>Name <input id="node-name" name="name" placeholder="edge-1"></label>
                <button class="btn" type="submit">Create Node</button>
                <p id="node-create-success" class="success" aria-live="polite"></p>
                <p id="node-create-error" class="error" aria-live="polite"></p>
              </form>
            </article>
          </section>
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Host</th>
                <th>Status</th>
                <th>Local</th>
                <th>Deploy Ready</th>
                <th>Readiness Codes</th>
                <th>Guidance</th>
              </tr>
            </thead>
            <tbody id="nodes-body"></tbody>
          </table>
        </section>

        <section class="panel" data-subsite="overview" id="subsite-overview" style="margin-bottom: 14px;">
          <h2>Overview</h2>
          <p class="muted">Operational metrics across jobs, nodes, and sites.</p>
          <div class="grid" id="metrics"></div>
        </section>

        <section class="split" data-subsite="sites" id="subsite-sites">
          <article class="panel">
            <h2>Create Site</h2>
            <form id="site-form">
              <label>Name <input id="site-name" name="name" required placeholder="Acme Store"></label>
              <label>Slug <input id="site-slug" name="slug" required placeholder="acme-store"></label>
              <button class="btn" type="submit">Create Site</button>
              <p id="site-success" class="success" aria-live="polite"></p>
              <p id="site-error" class="error" aria-live="polite"></p>
            </form>
          </article>
        </section>

        <section class="panel" data-subsite="sites" style="margin-bottom: 14px;">
          <div class="head">
            <h2>Sites</h2>
            <p id="site-count" class="muted">0 sites</p>
          </div>
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Slug</th>
                <th>Status</th>
                <th>Node</th>
                <th>Preview URL</th>
                <th>WordPress</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody id="sites-body"></tbody>
          </table>
        </section>

        <section class="panel" data-subsite="site-detail" id="subsite-site-detail" style="margin-bottom: 14px;">
          <div class="head">
            <h2>Site Detail</h2>
            <p id="site-detail-title" class="muted">Select a site from /sites to manage environments and backups.</p>
          </div>
        </section>

        <section data-subsite="site-detail" id="subsite-environments">
          <article class="panel">
            <h2>Create Environment</h2>
            <form id="environment-form">
              <label id="environment-site-label">Site
                <select id="environment-site" required>
                  <option value="">Select a site</option>
                </select>
              </label>
              <label>Name <input id="environment-name" name="name" required placeholder="Staging"></label>
              <div class="inline">
                <label>Slug <input id="environment-slug" name="slug" required placeholder="staging"></label>
                <label>Type
                  <select id="environment-type">
                    <option value="staging">staging</option>
                    <option value="clone">clone</option>
                  </select>
                </label>
              </div>
              <label>Source Environment
                <select id="environment-source" required>
                  <option value="">Select source environment</option>
                </select>
              </label>
              <label>Promotion Preset
                <select id="environment-preset">
                  <option value="content-protect">content-protect</option>
                  <option value="commerce-protect">commerce-protect</option>
                </select>
              </label>
              <button class="btn" type="submit">Create Environment</button>
              <p id="environment-success" class="success" aria-live="polite"></p>
              <p id="environment-error" class="error" aria-live="polite"></p>
            </form>
          </article>
        </section>

        <section class="panel" data-subsite="site-detail" style="margin-bottom: 14px;">
          <div class="head">
            <h2>Environments</h2>
            <p id="environment-title" class="muted">Select a site to view environments.</p>
          </div>
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Slug</th>
                <th>Type</th>
                <th>Status</th>
                <th>Preview URL</th>
              </tr>
            </thead>
            <tbody id="environments-body"></tbody>
          </table>
        </section>

        <section class="panel" data-subsite="site-detail" id="subsite-backups" style="margin-bottom: 14px;">
          <div class="head">
            <h2>Backups</h2>
            <p id="backup-title" class="muted">Select a site and environment to create and view backups.</p>
          </div>
          <form id="backup-form">
            <label id="backup-site-label">Site
                <select id="backup-site" required>
                  <option value="">Select site</option>
                </select>
            </label>
            <div class="inline">
              <label>Environment
                <select id="backup-environment" required>
                  <option value="">Select environment</option>
                </select>
              </label>
              <label>Scope
                <select id="backup-scope">
                  <option value="full">full</option>
                  <option value="db">db</option>
                  <option value="files">files</option>
                </select>
              </label>
            </div>
            <button class="btn" type="submit">Create Backup</button>
            <p id="backup-success" class="success" aria-live="polite"></p>
            <p id="backup-error" class="error" aria-live="polite"></p>
          </form>
          <form id="restore-form" style="margin-top: 10px;">
            <div class="inline">
              <label>Completed Backup
                <select id="restore-backup" required>
                  <option value="">Select completed backup</option>
                </select>
              </label>
              <label>Confirm
                <select id="restore-confirm" required>
                  <option value="">Choose</option>
                  <option value="yes">I understand this will replace environment content</option>
                </select>
              </label>
            </div>
            <button class="btn" type="submit">Restore Environment</button>
            <p id="restore-success" class="success" aria-live="polite"></p>
            <p id="restore-error" class="error" aria-live="polite"></p>
          </form>
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Scope</th>
                <th>Status</th>
                <th>Retention Until</th>
                <th>Created At</th>
              </tr>
            </thead>
            <tbody id="backups-body"></tbody>
          </table>
        </section>

        <section class="panel" data-subsite="jobs" id="subsite-jobs">
          <div class="head">
            <h2>Recent Jobs</h2>
            <button id="refresh" class="btn ghost" type="button">Refresh</button>
          </div>
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Type</th>
                <th>Status</th>
                <th>Attempts</th>
                <th>Error</th>
              </tr>
            </thead>
            <tbody id="jobs-body"></tbody>
          </table>
        </section>

        <section class="panel" data-subsite="jobs" style="margin-top: 14px;">
          <div class="head">
            <h2>Job Timeline</h2>
            <p id="job-detail-title" class="muted">Select a job from the table.</p>
          </div>
          <ul id="job-timeline" class="timeline">
            <li class="muted">No job selected.</li>
          </ul>
          <p id="job-detail-error" class="error" aria-live="polite"></p>
        </section>
      </section>
    </section>
  </main>

  <script>
    const dom = {
      authPanel: document.getElementById('auth-panel'),
      dashboard: document.getElementById('dashboard'),
      authError: document.getElementById('auth-error'),
      loginForm: document.getElementById('login-form'),
      logoutButton: document.getElementById('logout'),
      refreshButton: document.getElementById('refresh'),
      metricsEl: document.getElementById('metrics'),
      nodesTitle: document.getElementById('nodes-title'),
      localNodeReadiness: document.getElementById('local-node-readiness'),
      nodesBody: document.getElementById('nodes-body'),
      providersTitle: document.getElementById('providers-title'),
      providersBody: document.getElementById('providers-body'),
      providerConnectForm: document.getElementById('provider-connect-form'),
      providerID: document.getElementById('provider-id'),
      providerAPIToken: document.getElementById('provider-api-token'),
      providerSuccess: document.getElementById('provider-success'),
      providerError: document.getElementById('provider-error'),
      nodeProviderForm: document.getElementById('node-provider-form'),
      nodeProviderID: document.getElementById('node-provider-id'),
      nodeName: document.getElementById('node-name'),
      nodeCreateSuccess: document.getElementById('node-create-success'),
      nodeCreateError: document.getElementById('node-create-error'),
      subsiteNav: document.getElementById('subsite-nav'),
      subsiteLinks: Array.from(document.querySelectorAll('[data-nav]')),
      subsiteSections: Array.from(document.querySelectorAll('[data-subsite]')),
      siteForm: document.getElementById('site-form'),
      siteNameInput: document.getElementById('site-name'),
      siteSlugInput: document.getElementById('site-slug'),
      siteError: document.getElementById('site-error'),
      siteSuccess: document.getElementById('site-success'),
      sitesBody: document.getElementById('sites-body'),
      siteCount: document.getElementById('site-count'),
      siteDetailTitle: document.getElementById('site-detail-title'),
      environmentForm: document.getElementById('environment-form'),
      environmentSiteLabel: document.getElementById('environment-site-label'),
      environmentSite: document.getElementById('environment-site'),
      environmentNameInput: document.getElementById('environment-name'),
      environmentSlugInput: document.getElementById('environment-slug'),
      environmentType: document.getElementById('environment-type'),
      environmentSource: document.getElementById('environment-source'),
      environmentPreset: document.getElementById('environment-preset'),
      environmentError: document.getElementById('environment-error'),
      environmentSuccess: document.getElementById('environment-success'),
      environmentsBody: document.getElementById('environments-body'),
      environmentTitle: document.getElementById('environment-title'),
      backupForm: document.getElementById('backup-form'),
      backupSiteLabel: document.getElementById('backup-site-label'),
      backupSite: document.getElementById('backup-site'),
      backupEnvironment: document.getElementById('backup-environment'),
      backupScope: document.getElementById('backup-scope'),
      backupTitle: document.getElementById('backup-title'),
      backupSuccess: document.getElementById('backup-success'),
      backupError: document.getElementById('backup-error'),
      restoreForm: document.getElementById('restore-form'),
      restoreBackup: document.getElementById('restore-backup'),
      restoreConfirm: document.getElementById('restore-confirm'),
      restoreSuccess: document.getElementById('restore-success'),
      restoreError: document.getElementById('restore-error'),
      backupsBody: document.getElementById('backups-body'),
      jobsBody: document.getElementById('jobs-body'),
      jobDetailTitle: document.getElementById('job-detail-title'),
      jobTimeline: document.getElementById('job-timeline'),
      jobDetailError: document.getElementById('job-detail-error'),
    };

    const metricCards = [
      { key: 'jobs_running', label: 'Jobs Running' },
      { key: 'jobs_queued', label: 'Jobs Queued' },
      { key: 'nodes_active', label: 'Nodes Active' },
      { key: 'sites_total', label: 'Sites Total' },
    ];

    const state = {
      selectedJobID: '',
      selectedSiteID: '',
      selectedBackupEnvironmentID: '',
      selectedRestoreBackupID: '',
      requestedSiteDetailID: '',
      requestedDetailFocus: '',
      activeSubsite: 'overview',
      siteEnvironmentsBySite: {},
      backupsByEnvironment: {},
      nodesByID: {},
      providersByID: {},
      wpVersionByEnvironment: {},
    };

    const api = {
      async request(path, options = {}) {
        const response = await fetch('/api' + path, {
          ...options,
          credentials: 'include',
          headers: {
            'Content-Type': 'application/json',
            ...(options.headers || {}),
          },
        });

        let body = null;
        try {
          body = await response.json();
        } catch (_) {}

        if (!response.ok) {
          const statusPrefix = response.status === 400 || response.status === 404 || response.status === 409
            ? String(response.status) + ' '
            : '';
          const err = new Error(statusPrefix + ((body && body.message) || ('request failed: ' + response.status)));
          err.status = response.status;
          err.code = body && body.code;
          throw err;
        }

        return body;
      },
    };

    const util = {
      escapeHTML(value) {
        return String(value || '')
          .replaceAll('&', '&amp;')
          .replaceAll('<', '&lt;')
          .replaceAll('>', '&gt;')
          .replaceAll('"', '&quot;')
          .replaceAll("'", '&#39;');
      },
      formatDisplayTimestamp(value) {
        if (!value) {
          return '-';
        }
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) {
          return '-';
        }
        return date.toLocaleString();
      },
      formatTimelineTimestamp(value) {
        if (!value) {
          return 'pending';
        }
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) {
          return 'pending';
        }
        return date.toLocaleString();
      },
      resetMap(mapRef) {
        Object.keys(mapRef).forEach((key) => delete mapRef[key]);
      },
    };

    const shell = {
      resolveRoute(pathname, search) {
        const siteDetailMatch = pathname.match(/^\/sites\/([^/]+)$/);
        if (siteDetailMatch) {
          const params = new URLSearchParams(search || '');
          const focusParam = params.get('focus') || '';
          const focus = focusParam === 'environment' || focusParam === 'backup' ? focusParam : '';
          return {
            subsite: 'site-detail',
            navSubsite: 'sites',
            siteID: decodeURIComponent(siteDetailMatch[1] || ''),
            focus,
          };
        }

        switch (pathname) {
          case '/':
            return { subsite: 'overview', navSubsite: 'overview', siteID: '', focus: '' };
          case '/providers':
            return { subsite: 'providers', navSubsite: 'providers', siteID: '', focus: '' };
          case '/nodes':
            return { subsite: 'nodes', navSubsite: 'nodes', siteID: '', focus: '' };
          case '/sites':
            return { subsite: 'sites', navSubsite: 'sites', siteID: '', focus: '' };
          case '/jobs':
            return { subsite: 'jobs', navSubsite: 'jobs', siteID: '', focus: '' };
          default:
            return { subsite: 'overview', navSubsite: 'overview', siteID: '', focus: '' };
        }
      },
      applySubsite(pathname, search) {
        const route = shell.resolveRoute(pathname, search);
        state.activeSubsite = route.subsite;
        state.requestedSiteDetailID = route.siteID;
        state.requestedDetailFocus = route.focus;
        if (route.siteID) {
          state.selectedSiteID = route.siteID;
        }

        dom.subsiteSections.forEach((section) => {
          const sectionName = section.getAttribute('data-subsite') || '';
          section.classList.toggle('hidden', sectionName !== state.activeSubsite);
        });
        siteDetailView.syncVisibility();

        dom.subsiteLinks.forEach((link) => {
          const linkName = link.getAttribute('data-nav') || '';
          const isActive = linkName === route.navSubsite;
          link.classList.toggle('active', isActive);
          if (isActive) {
            link.setAttribute('aria-current', 'page');
            return;
          }
          link.removeAttribute('aria-current');
        });
      },
      setAuthed(authed) {
        dom.authPanel.classList.toggle('hidden', authed);
        dom.dashboard.classList.toggle('hidden', !authed);
        dom.logoutButton.classList.toggle('hidden', !authed);
        if (authed) {
          shell.applySubsite(window.location.pathname, window.location.search);
        }
      },
      bindNavigation() {
        dom.subsiteNav.addEventListener('click', (event) => {
          const target = event.target;
          if (!(target instanceof HTMLElement)) {
            return;
          }

          const link = target.closest('a[data-nav]');
          if (!(link instanceof HTMLAnchorElement)) {
            return;
          }

          event.preventDefault();
          const nextPath = link.getAttribute('href') || '/';
          if (window.location.pathname !== nextPath) {
            window.history.pushState({}, '', nextPath);
          }
          shell.applySubsite(nextPath, '');
        });

        window.addEventListener('popstate', () => {
          shell.applySubsite(window.location.pathname, window.location.search);
        });
      },
      navigateToSiteDetail(siteID, focus) {
        if (!siteID) {
          return;
        }
        const base = '/sites/' + encodeURIComponent(siteID);
        const query = focus ? ('?focus=' + encodeURIComponent(focus)) : '';
        const nextPath = base + query;
        if ((window.location.pathname + window.location.search) !== nextPath) {
          window.history.pushState({}, '', nextPath);
        }
        shell.applySubsite(window.location.pathname, window.location.search);
        siteDetailView.applyFocus();
      },
    };

    const overviewView = {
      renderMetrics(metrics) {
        dom.metricsEl.innerHTML = metricCards.map((item) => {
          const value = Number.isFinite(metrics[item.key]) ? metrics[item.key] : 0;
          return '<article class="metric"><b>' + value + '</b><span class="muted">' + item.label + '</span></article>';
        }).join('');
      },
      clear() {
        dom.metricsEl.innerHTML = '';
      },
    };

    const nodesView = {
      cache(nodesList) {
        util.resetMap(state.nodesByID);
        (Array.isArray(nodesList) ? nodesList : []).forEach((node) => {
          if (node && node.id) {
            state.nodesByID[node.id] = node;
          }
        });
      },
      render(nodesList) {
        const nodes = Array.isArray(nodesList) ? nodesList : [];
        nodesView.cache(nodes);
        dom.nodesTitle.textContent = nodes.length === 1 ? '1 node registered.' : String(nodes.length) + ' nodes registered.';

        const providerNode = nodes.find((node) => node && node.is_local === false);
        if (!providerNode) {
          dom.localNodeReadiness.textContent = 'Provider-backed node not registered. Site create will fail until a provider node is ready.';
        } else {
          const readiness = providerNode.readiness || {};
          const reasonCodes = Array.isArray(readiness.reason_codes) ? readiness.reason_codes : [];
          const guidance = Array.isArray(readiness.guidance) ? readiness.guidance : [];
          if (readiness.is_ready) {
            dom.localNodeReadiness.textContent = 'Provider-backed node is present and deploy-ready.';
          } else {
            const details = reasonCodes.length > 0 ? reasonCodes.join(', ') : ('status: ' + providerNode.status);
            const help = guidance.length > 0 ? (' ' + guidance.join(' ')) : '';
            dom.localNodeReadiness.textContent = 'Provider-backed node is not ready (' + details + ').' + help;
          }
        }

        if (nodes.length === 0) {
          dom.nodesBody.innerHTML = '<tr><td colspan="7" class="muted">No nodes registered yet.</td></tr>';
          return;
        }

        dom.nodesBody.innerHTML = nodes.map((node) => {
          const status = String(node.status || 'unknown');
          const host = node.public_ip || node.hostname || (node.is_local ? 'localhost' : '-');
          const readiness = node.readiness || {};
          const reasonCodes = Array.isArray(readiness.reason_codes) ? readiness.reason_codes : [];
          const guidance = Array.isArray(readiness.guidance) ? readiness.guidance : [];
          const deployReady = readiness.is_ready ? 'yes' : 'no';
          const local = node.is_local ? 'yes' : 'no';
          const readinessCodesLabel = reasonCodes.length > 0 ? reasonCodes.join(', ') : '-';
          const guidanceLabel = guidance.length > 0 ? guidance.join(' ') : '-';
          return [
            '<tr>',
            '<td>' + util.escapeHTML(node.name || '-') + '</td>',
            '<td>' + util.escapeHTML(host) + '</td>',
            '<td><span class="status ' + util.escapeHTML(status) + '">' + util.escapeHTML(status) + '</span></td>',
            '<td>' + local + '</td>',
            '<td>' + deployReady + '</td>',
            '<td>' + util.escapeHTML(readinessCodesLabel) + '</td>',
            '<td>' + util.escapeHTML(guidanceLabel) + '</td>',
            '</tr>',
          ].join('');
        }).join('');
      },
      clear() {
        dom.nodesTitle.textContent = 'Runtime node inventory and provider readiness.';
        dom.localNodeReadiness.textContent = 'Provider-backed node readiness unknown.';
        dom.nodesBody.innerHTML = '';
        nodesView.clearMessages();
      },
      clearMessages() {
        dom.nodeCreateSuccess.textContent = '';
        dom.nodeCreateError.textContent = '';
      },
      bind() {
        dom.nodeProviderForm.addEventListener('submit', async (event) => {
          event.preventDefault();
          nodesView.clearMessages();

          const payload = { provider_id: dom.nodeProviderID.value };
          const name = dom.nodeName.value.trim();
          if (name) {
            payload.name = name;
          }

          try {
            const accepted = await api.request('/nodes', {
              method: 'POST',
              body: JSON.stringify(payload),
            });
            await dashboardController.load();
            dom.nodeCreateSuccess.textContent = 'Create Node accepted (' + (accepted.job_id || '-') + ').';
          } catch (err) {
            dom.nodeCreateError.textContent = err.message || 'Create Node failed';
          }
        });
      },
    };

    const providersView = {
      cache(providersList) {
        util.resetMap(state.providersByID);
        (Array.isArray(providersList) ? providersList : []).forEach((provider) => {
          if (provider && provider.provider_id) {
            state.providersByID[provider.provider_id] = provider;
          }
        });
      },
      render(providersList) {
        const providers = Array.isArray(providersList) ? providersList : [];
        providersView.cache(providers);
        if (providers.length === 0) {
          dom.providersTitle.textContent = 'No providers available.';
          dom.providersBody.innerHTML = '<tr><td colspan="5" class="muted">No providers available.</td></tr>';
          return;
        }

        const connected = providers.filter((provider) => provider && provider.status === 'connected').length;
        if (connected === 0) {
          dom.providersTitle.textContent = 'No connected providers. Node creation remains blocked until a provider is connected.';
        } else {
          dom.providersTitle.textContent = String(connected) + ' provider' + (connected === 1 ? '' : 's') + ' connected for node workflows.';
        }

        dom.providersBody.innerHTML = providers.map((provider) => {
          const status = String(provider.status || 'unknown');
          const capabilities = Array.isArray(provider.capabilities) ? provider.capabilities.join(', ') : '-';
          const guidance = Array.isArray(provider.guidance) ? provider.guidance.join(' ') : '-';
          const secret = provider.secret_configured ? 'configured' : 'missing';
          return [
            '<tr>',
            '<td>' + util.escapeHTML(provider.display_name || provider.provider_id || '-') + '</td>',
            '<td><span class="status ' + util.escapeHTML(status) + '">' + util.escapeHTML(status) + '</span></td>',
            '<td>' + util.escapeHTML(secret) + '</td>',
            '<td>' + util.escapeHTML(capabilities || '-') + '</td>',
            '<td>' + util.escapeHTML(guidance || '-') + '</td>',
            '</tr>',
          ].join('');
        }).join('');
      },
      clearMessages() {
        dom.providerSuccess.textContent = '';
        dom.providerError.textContent = '';
      },
      clear() {
        providersView.clearMessages();
        dom.providersTitle.textContent = 'Connect provider bearer credentials before creating nodes.';
        dom.providersBody.innerHTML = '';
        util.resetMap(state.providersByID);
      },
      bind() {
        dom.providerConnectForm.addEventListener('submit', async (event) => {
          event.preventDefault();
          providersView.clearMessages();

          try {
            const provider = await api.request('/providers', {
              method: 'POST',
              body: JSON.stringify({
                provider_id: dom.providerID.value,
                api_token: dom.providerAPIToken.value,
              }),
            });
            dom.providerSuccess.textContent = (provider.display_name || provider.provider_id || 'Provider') + ' connected.';
            dom.providerAPIToken.value = '';
            await dashboardController.load();
          } catch (err) {
            dom.providerError.textContent = err.message || 'Provider connect failed';
          }
        });
      },
    };

    const siteDetailView = {
      syncVisibility() {
        const isDetail = state.activeSubsite === 'site-detail';
        dom.environmentSiteLabel.classList.toggle('hidden', isDetail);
        dom.backupSiteLabel.classList.toggle('hidden', isDetail);
      },
      renderTitle(sites) {
        siteDetailView.syncVisibility();
        if (state.activeSubsite !== 'site-detail') {
          dom.siteDetailTitle.textContent = 'Select a site from /sites to manage environments and backups.';
          return;
        }
        const site = (Array.isArray(sites) ? sites : []).find((entry) => entry && entry.id === state.selectedSiteID);
        if (!site) {
          dom.siteDetailTitle.textContent = 'Requested site was not found. Return to /sites and pick another site.';
          return;
        }
        dom.siteDetailTitle.textContent = 'Managing environments and backups for ' + (site.name || site.slug || site.id || 'selected site') + '.';
      },
      applyFocus() {
        if (state.activeSubsite !== 'site-detail') {
          return;
        }
        if (state.requestedDetailFocus === 'environment') {
          dom.environmentForm.scrollIntoView({ behavior: 'smooth', block: 'start' });
        }
        if (state.requestedDetailFocus === 'backup') {
          dom.backupForm.scrollIntoView({ behavior: 'smooth', block: 'start' });
        }
      },
    };

    const sitesView = {
      renderSiteOptions(sites) {
        const options = ['<option value="">Select a site</option>'].concat(
          sites.map((site) => '<option value="' + util.escapeHTML(site.id) + '">' + util.escapeHTML(site.name) + ' (' + util.escapeHTML(site.slug) + ')</option>')
        ).join('');
        dom.environmentSite.innerHTML = options;
        dom.backupSite.innerHTML = options;

        if (!state.selectedSiteID && sites.length > 0) {
          state.selectedSiteID = sites[0].id || '';
        }
        if (state.selectedSiteID) {
          const exists = sites.some((site) => site.id === state.selectedSiteID);
          if (!exists) {
            if (state.activeSubsite === 'site-detail' && state.requestedSiteDetailID) {
              state.selectedSiteID = state.requestedSiteDetailID;
            } else {
              state.selectedSiteID = sites.length > 0 ? (sites[0].id || '') : '';
            }
          }
        }
        dom.environmentSite.value = state.selectedSiteID || '';
        dom.backupSite.value = state.selectedSiteID || '';
      },
      renderSitesTable(sites) {
        if (sites.length === 0) {
          dom.sitesBody.innerHTML = '<tr><td colspan="7" class="muted">No sites available yet. Create your first site to unlock environments and backups.</td></tr>';
          return;
        }

        dom.sitesBody.innerHTML = sites.map((site) => {
          const status = String(site.status || 'unknown');
          const siteEnvironments = state.siteEnvironmentsBySite[site.id] || [];
          const primaryEnvironment = siteEnvironments.find((environment) => environment.id === site.primary_environment_id) || null;
          const preview = primaryEnvironment ? (primaryEnvironment.preview_url || '-') : '-';
          const nodeID = primaryEnvironment ? (primaryEnvironment.node_id || '') : '';
          const node = nodeID ? state.nodesByID[nodeID] : null;
          const nodeLabel = node ? ((node.name || node.id || 'node') + (node.is_local ? ' (local)' : '')) : (nodeID || '-');
          const wpVersionEntry = primaryEnvironment ? state.wpVersionByEnvironment[primaryEnvironment.id] : null;
          let wpVersion = '-';
          if (wpVersionEntry && wpVersionEntry.version) {
            wpVersion = wpVersionEntry.version;
          } else if (wpVersionEntry && wpVersionEntry.error) {
            wpVersion = 'error: ' + wpVersionEntry.error;
          }
          return [
            '<tr>',
            '<td>' + util.escapeHTML(site.name || '-') + '</td>',
            '<td>' + util.escapeHTML(site.slug || '-') + '</td>',
            '<td><span class="status ' + util.escapeHTML(status) + '">' + util.escapeHTML(status) + '</span></td>',
            '<td>' + util.escapeHTML(nodeLabel) + '</td>',
            '<td>' + util.escapeHTML(preview) + '</td>',
            '<td>' + util.escapeHTML(wpVersion) + '</td>',
            '<td class="actions-cell">',
            '<button class="icon-btn" type="button" data-site-actions-toggle="' + util.escapeHTML(site.id || '') + '" aria-haspopup="true" aria-expanded="false">...</button>',
            '<div class="actions-menu hidden" data-site-actions-menu="' + util.escapeHTML(site.id || '') + '">',
            '<button type="button" data-site-action="open-detail" data-site-id="' + util.escapeHTML(site.id || '') + '">Open details</button>',
            '<button type="button" data-site-action="create-environment" data-site-id="' + util.escapeHTML(site.id || '') + '">Create environment</button>',
            '<button type="button" data-site-action="create-backup" data-site-id="' + util.escapeHTML(site.id || '') + '">Create backup</button>',
            '</div>',
            '</td>',
            '</tr>',
          ].join('');
        }).join('');
      },
      render(sites) {
        const list = Array.isArray(sites) ? sites : [];
        dom.siteCount.textContent = String(list.length) + (list.length === 1 ? ' site' : ' sites');
        sitesView.renderSiteOptions(list);
        sitesView.renderSitesTable(list);
        siteDetailView.renderTitle(list);
      },
      clearMessages() {
        dom.siteError.textContent = '';
        dom.siteSuccess.textContent = '';
      },
      clear() {
        siteDetailView.syncVisibility();
        dom.sitesBody.innerHTML = '';
        dom.siteCount.textContent = '0 sites';
        dom.environmentSite.innerHTML = '<option value="">Select a site</option>';
        dom.backupSite.innerHTML = '<option value="">Select site</option>';
        sitesView.clearMessages();
      },
      bind() {
        document.addEventListener('click', (event) => {
          const target = event.target;
          if (!(target instanceof HTMLElement)) {
            return;
          }
          const isToggle = target.closest('button[data-site-actions-toggle]');
          const isAction = target.closest('button[data-site-action]');
          if (isToggle || isAction) {
            return;
          }
          document.querySelectorAll('[data-site-actions-menu]').forEach((entry) => {
            entry.classList.add('hidden');
          });
          document.querySelectorAll('[data-site-actions-toggle]').forEach((entry) => {
            entry.setAttribute('aria-expanded', 'false');
          });
        });

        dom.sitesBody.addEventListener('click', (event) => {
          const target = event.target;
          if (!(target instanceof HTMLElement)) {
            return;
          }

          const toggle = target.closest('button[data-site-actions-toggle]');
          if (toggle) {
            const siteID = toggle.getAttribute('data-site-actions-toggle') || '';
            const menu = dom.sitesBody.querySelector('[data-site-actions-menu="' + siteID + '"]');
            if (!menu) {
              return;
            }
            const expanded = toggle.getAttribute('aria-expanded') === 'true';
            document.querySelectorAll('[data-site-actions-menu]').forEach((entry) => entry.classList.add('hidden'));
            document.querySelectorAll('[data-site-actions-toggle]').forEach((entry) => entry.setAttribute('aria-expanded', 'false'));
            if (!expanded) {
              menu.classList.remove('hidden');
              toggle.setAttribute('aria-expanded', 'true');
            }
            return;
          }

          const action = target.closest('button[data-site-action]');
          if (!action) {
            return;
          }

          const siteID = action.getAttribute('data-site-id') || '';
          if (!siteID) {
            return;
          }

          const actionName = action.getAttribute('data-site-action') || '';
          switch (actionName) {
            case 'open-detail':
              shell.navigateToSiteDetail(siteID, '');
              return;
            case 'create-environment':
              shell.navigateToSiteDetail(siteID, 'environment');
              return;
            case 'create-backup':
              shell.navigateToSiteDetail(siteID, 'backup');
              return;
            default:
              return;
          }
        });

        dom.siteForm.addEventListener('submit', async (event) => {
          event.preventDefault();
          sitesView.clearMessages();

          try {
            await api.request('/sites', {
              method: 'POST',
              body: JSON.stringify({
                name: dom.siteNameInput.value,
                slug: dom.siteSlugInput.value,
              }),
            });
            dom.siteSuccess.textContent = 'Site create accepted.';
            await dashboardData.loadSiteEnvironmentBackupData();
          } catch (err) {
            dom.siteError.textContent = err.message || 'Site create failed';
          }
        });
      },
    };

    const environmentsView = {
      hasRequestedMissingSite() {
        if (state.activeSubsite !== 'site-detail' || !state.requestedSiteDetailID) {
          return false;
        }
        return !Object.prototype.hasOwnProperty.call(state.siteEnvironmentsBySite, state.selectedSiteID);
      },
      renderSourceOptions() {
        const envs = state.siteEnvironmentsBySite[state.selectedSiteID] || [];
        dom.environmentSource.innerHTML = ['<option value="">Select source environment</option>'].concat(
          envs.map((environment) => '<option value="' + util.escapeHTML(environment.id) + '">' + util.escapeHTML(environment.name) + ' (' + util.escapeHTML(environment.slug) + ')</option>')
        ).join('');
      },
      renderTable() {
        const envs = state.siteEnvironmentsBySite[state.selectedSiteID] || [];
        const requestedButMissing = environmentsView.hasRequestedMissingSite();
        if (requestedButMissing) {
          dom.environmentsBody.innerHTML = '<tr><td colspan="5" class="muted">Requested site not found. Return to /sites and choose a valid site.</td></tr>';
          dom.environmentTitle.textContent = 'Requested site not found.';
          environmentsView.renderSourceOptions();
          return;
        }

        if (!state.selectedSiteID) {
          dom.environmentsBody.innerHTML = '<tr><td colspan="5" class="muted">Create or select a site to view environments.</td></tr>';
          dom.environmentTitle.textContent = 'Select a site to view environments.';
          environmentsView.renderSourceOptions();
          return;
        }

        dom.environmentTitle.textContent = 'Showing environments for selected site.';
        if (envs.length === 0) {
          dom.environmentsBody.innerHTML = '<tr><td colspan="5" class="muted">No environments available for selected site.</td></tr>';
          environmentsView.renderSourceOptions();
          return;
        }

        dom.environmentsBody.innerHTML = envs.map((environment) => {
          const status = String(environment.status || 'unknown');
          const preview = environment.preview_url || '-';
          return [
            '<tr>',
            '<td>' + util.escapeHTML(environment.name || '-') + '</td>',
            '<td>' + util.escapeHTML(environment.slug || '-') + '</td>',
            '<td>' + util.escapeHTML(environment.environment_type || '-') + '</td>',
            '<td><span class="status ' + util.escapeHTML(status) + '">' + util.escapeHTML(status) + '</span></td>',
            '<td>' + util.escapeHTML(preview) + '</td>',
            '</tr>',
          ].join('');
        }).join('');

        environmentsView.renderSourceOptions();
      },
      clearMessages() {
        dom.environmentError.textContent = '';
        dom.environmentSuccess.textContent = '';
      },
      clear() {
        dom.environmentsBody.innerHTML = '';
        dom.environmentTitle.textContent = 'Select a site to view environments.';
        dom.environmentSource.innerHTML = '<option value="">Select source environment</option>';
        dom.environmentSiteLabel.classList.add('hidden');
        environmentsView.clearMessages();
      },
      bind() {
        dom.environmentForm.addEventListener('submit', async (event) => {
          event.preventDefault();
          environmentsView.clearMessages();

          const siteID = state.selectedSiteID || dom.environmentSite.value;
          if (environmentsView.hasRequestedMissingSite()) {
            dom.environmentError.textContent = '404 requested site not found';
            return;
          }
          if (!siteID) {
            dom.environmentError.textContent = '400 site is required';
            return;
          }

          try {
            await api.request('/sites/' + encodeURIComponent(siteID) + '/environments', {
              method: 'POST',
              body: JSON.stringify({
                name: dom.environmentNameInput.value,
                slug: dom.environmentSlugInput.value,
                type: dom.environmentType.value,
                source_environment_id: dom.environmentSource.value || null,
                promotion_preset: dom.environmentPreset.value,
              }),
            });
            dom.environmentSuccess.textContent = 'Environment create accepted.';
            state.selectedSiteID = siteID;
            await dashboardData.loadSiteEnvironmentBackupData();
          } catch (err) {
            dom.environmentError.textContent = err.message || 'Environment create failed';
          }
        });
      },
    };

    const backupsView = {
      hasRequestedMissingSite() {
        if (state.activeSubsite !== 'site-detail' || !state.requestedSiteDetailID) {
          return false;
        }
        return !Object.prototype.hasOwnProperty.call(state.siteEnvironmentsBySite, state.selectedSiteID);
      },
      renderEnvironmentOptions() {
        const envs = state.siteEnvironmentsBySite[state.selectedSiteID] || [];
        dom.backupEnvironment.innerHTML = ['<option value="">Select environment</option>'].concat(
          envs.map((environment) => '<option value="' + util.escapeHTML(environment.id) + '">' + util.escapeHTML(environment.name) + ' (' + util.escapeHTML(environment.slug) + ')</option>')
        ).join('');

        if (!state.selectedBackupEnvironmentID && envs.length > 0) {
          state.selectedBackupEnvironmentID = envs[0].id || '';
        }
        if (state.selectedBackupEnvironmentID) {
          const exists = envs.some((environment) => environment.id === state.selectedBackupEnvironmentID);
          if (!exists) {
            state.selectedBackupEnvironmentID = envs.length > 0 ? (envs[0].id || '') : '';
          }
        }

        dom.backupEnvironment.value = state.selectedBackupEnvironmentID || '';
	      backupsView.renderRestoreBackupOptions();
      },
      renderRestoreBackupOptions() {
        const backups = state.backupsByEnvironment[state.selectedBackupEnvironmentID] || [];
        const completed = backups.filter((backup) => backup && backup.status === 'completed');
        dom.restoreBackup.innerHTML = ['<option value="">Select completed backup</option>'].concat(
          completed.map((backup) => '<option value="' + util.escapeHTML(backup.id) + '">' + util.escapeHTML(backup.id) + ' (' + util.escapeHTML(backup.backup_scope || '-') + ')</option>')
        ).join('');

        if (!state.selectedRestoreBackupID && completed.length > 0) {
          state.selectedRestoreBackupID = completed[0].id || '';
        }
        if (state.selectedRestoreBackupID) {
          const exists = completed.some((backup) => backup.id === state.selectedRestoreBackupID);
          if (!exists) {
            state.selectedRestoreBackupID = completed.length > 0 ? (completed[0].id || '') : '';
          }
        }
        dom.restoreBackup.value = state.selectedRestoreBackupID || '';
      },
      renderTable() {
        const envs = state.siteEnvironmentsBySite[state.selectedSiteID] || [];
        const requestedButMissing = backupsView.hasRequestedMissingSite();
        if (requestedButMissing) {
          dom.backupTitle.textContent = 'Requested site not found.';
          dom.backupsBody.innerHTML = '<tr><td colspan="5" class="muted">Requested site not found. Return to /sites and choose a valid site.</td></tr>';
          backupsView.renderEnvironmentOptions();
          return;
        }

        if (!state.selectedSiteID) {
          dom.backupTitle.textContent = 'Select a site and environment to create and view backups.';
          dom.backupsBody.innerHTML = '<tr><td colspan="5" class="muted">Select a site and environment to view backups.</td></tr>';
          backupsView.renderEnvironmentOptions();
          return;
        }

        backupsView.renderEnvironmentOptions();
        if (!state.selectedBackupEnvironmentID) {
          dom.backupTitle.textContent = 'Select an environment to create and view backups.';
          dom.backupsBody.innerHTML = '<tr><td colspan="5" class="muted">No environments available for selected site.</td></tr>';
          return;
        }

        const selectedEnvironment = envs.find((environment) => environment.id === state.selectedBackupEnvironmentID);
        const selectedLabel = selectedEnvironment ? selectedEnvironment.name : state.selectedBackupEnvironmentID;
        dom.backupTitle.textContent = 'Showing backups for ' + selectedLabel + '.';

        const backups = state.backupsByEnvironment[state.selectedBackupEnvironmentID] || [];
        if (backups.length === 0) {
          dom.backupsBody.innerHTML = '<tr><td colspan="5" class="muted">No backups available for selected environment.</td></tr>';
          return;
        }

        dom.backupsBody.innerHTML = backups.map((backup) => {
          const status = String(backup.status || 'unknown');
          return [
            '<tr>',
            '<td>' + util.escapeHTML(backup.id || '-') + '</td>',
            '<td>' + util.escapeHTML(backup.backup_scope || '-') + '</td>',
            '<td><span class="status ' + util.escapeHTML(status) + '">' + util.escapeHTML(status) + '</span></td>',
            '<td>' + util.escapeHTML(util.formatDisplayTimestamp(backup.retention_until)) + '</td>',
            '<td>' + util.escapeHTML(util.formatDisplayTimestamp(backup.created_at)) + '</td>',
            '</tr>',
          ].join('');
        }).join('');
      },
      clearMessages() {
        dom.backupError.textContent = '';
        dom.backupSuccess.textContent = '';
	      dom.restoreError.textContent = '';
	      dom.restoreSuccess.textContent = '';
      },
      clear() {
        dom.backupsBody.innerHTML = '';
        dom.backupTitle.textContent = 'Select a site and environment to create and view backups.';
        dom.backupSite.innerHTML = '<option value="">Select site</option>';
        dom.backupEnvironment.innerHTML = '<option value="">Select environment</option>';
        dom.restoreBackup.innerHTML = '<option value="">Select completed backup</option>';
        dom.restoreConfirm.value = '';
        dom.backupSiteLabel.classList.add('hidden');
        backupsView.clearMessages();
      },
      bind() {
        dom.backupEnvironment.addEventListener('change', () => {
          state.selectedBackupEnvironmentID = dom.backupEnvironment.value;
          backupsView.renderTable();
        });

	      dom.restoreBackup.addEventListener('change', () => {
	        state.selectedRestoreBackupID = dom.restoreBackup.value;
	      });

        dom.backupForm.addEventListener('submit', async (event) => {
          event.preventDefault();
          backupsView.clearMessages();

          const environmentID = dom.backupEnvironment.value;
          if (backupsView.hasRequestedMissingSite()) {
            dom.backupError.textContent = '404 requested site not found';
            return;
          }
          if (!environmentID) {
            dom.backupError.textContent = '400 environment is required';
            return;
          }

          try {
            await api.request('/environments/' + encodeURIComponent(environmentID) + '/backups', {
              method: 'POST',
              body: JSON.stringify({ backup_scope: dom.backupScope.value }),
            });
            dom.backupSuccess.textContent = 'Backup create accepted.';
            state.selectedBackupEnvironmentID = environmentID;
            await dashboardData.loadSiteEnvironmentBackupData();
          } catch (err) {
            dom.backupError.textContent = err.message || 'Backup create failed';
          }
        });

	      dom.restoreForm.addEventListener('submit', async (event) => {
	        event.preventDefault();
	        backupsView.clearMessages();

	        const environmentID = dom.backupEnvironment.value;
	        const backupID = dom.restoreBackup.value;
	        if (backupsView.hasRequestedMissingSite()) {
	          dom.restoreError.textContent = '404 requested site not found';
	          return;
	        }
	        if (!environmentID || !backupID) {
	          dom.restoreError.textContent = '400 environment and backup are required';
	          return;
	        }
	        if (dom.restoreConfirm.value !== 'yes') {
	          dom.restoreError.textContent = '400 explicit restore confirmation is required';
	          return;
	        }

	        try {
	          await api.request('/environments/' + encodeURIComponent(environmentID) + '/restore', {
	            method: 'POST',
	            body: JSON.stringify({ backup_id: backupID }),
	          });
	          dom.restoreSuccess.textContent = 'Restore accepted.';
	          dom.restoreConfirm.value = '';
	          await dashboardData.loadSiteEnvironmentBackupData();
	        } catch (err) {
	          dom.restoreError.textContent = err.message || 'Restore failed';
	        }
	      });
      },
    };

    const jobsView = {
      clearDetail(message) {
        dom.jobDetailTitle.textContent = message;
        dom.jobTimeline.innerHTML = '<li class="muted">' + message + '</li>';
        dom.jobDetailError.textContent = '';
      },
      buildTimeline(job) {
        const events = [{
          state: 'queued',
          at: job.created_at,
          detail: 'Job created and waiting for worker pickup.',
        }];

        if (job.attempt_count > 0 && job.started_at) {
          events.push({
            state: 'running',
            at: job.started_at,
            detail: 'Attempt ' + String(job.attempt_count) + ' of ' + String(job.max_attempts || 0) + ' started.',
          });
        }
        if (job.status === 'queued' && job.attempt_count > 0 && job.run_after) {
          events.push({
            state: 'queued',
            at: job.run_after,
            detail: 'Retry scheduled after previous failure.',
          });
        }
        if (job.status === 'running') {
          events.push({
            state: 'running',
            at: job.started_at || job.updated_at,
            detail: 'Worker currently executing this job.',
          });
        }
        if (job.status === 'succeeded') {
          events.push({
            state: 'succeeded',
            at: job.finished_at || job.updated_at,
            detail: 'Mutation completed successfully.',
          });
        }
        if (job.status === 'failed') {
          events.push({
            state: 'failed',
            at: job.finished_at || job.updated_at,
            detail: job.error_code ? 'Execution failed (' + job.error_code + ').' : 'Execution failed.',
          });
        }

        return events;
      },
      renderDetail(job) {
        dom.jobDetailTitle.textContent = 'Job ' + (job.id || '-') + ' timeline';
        dom.jobTimeline.innerHTML = jobsView.buildTimeline(job).map((event) => [
          '<li>',
          '<strong class="status ' + event.state + '">' + event.state + '</strong>',
          '<time>' + util.formatTimelineTimestamp(event.at) + '</time>',
          '<div class="muted">' + event.detail + '</div>',
          '</li>',
        ].join('')).join('');
        dom.jobDetailError.textContent = '';
      },
      async loadDetail(jobID) {
        if (!jobID) {
          jobsView.clearDetail('No job selected.');
          return;
        }
        const job = await api.request('/jobs/' + encodeURIComponent(jobID));
        state.selectedJobID = jobID;
        jobsView.renderDetail(job);
      },
      renderJobsTable(jobs) {
        if (!Array.isArray(jobs) || jobs.length === 0) {
          dom.jobsBody.innerHTML = '<tr><td colspan="5" class="muted">No jobs available.</td></tr>';
          state.selectedJobID = '';
          jobsView.clearDetail('No job selected.');
          return;
        }

        dom.jobsBody.innerHTML = jobs.map((job) => {
          const status = String(job.status || 'unknown');
          const attempts = String(job.attempt_count || 0) + '/' + String(job.max_attempts || 0);
          const err = job.error_code || '-';
          return [
            '<tr>',
            '<td><button class="job-link" type="button" data-job-id="' + (job.id || '') + '">' + (job.id || '-') + '</button></td>',
            '<td>' + (job.job_type || '-') + '</td>',
            '<td><span class="status ' + status + '">' + status + '</span></td>',
            '<td>' + attempts + '</td>',
            '<td>' + err + '</td>',
            '</tr>',
          ].join('');
        }).join('');

        if (state.selectedJobID) {
          const exists = jobs.some((job) => job.id === state.selectedJobID);
          if (!exists) {
            state.selectedJobID = '';
            jobsView.clearDetail('Selected job is no longer available.');
          }
        }
      },
      clear() {
        dom.jobsBody.innerHTML = '';
        jobsView.clearDetail('No job selected.');
      },
      bind() {
        dom.jobsBody.addEventListener('click', async (event) => {
          const target = event.target;
          if (!(target instanceof HTMLElement)) {
            return;
          }

          const button = target.closest('button[data-job-id]');
          if (!button) {
            return;
          }

          const jobID = button.getAttribute('data-job-id') || '';
          dom.jobDetailError.textContent = '';
          try {
            await jobsView.loadDetail(jobID);
          } catch (err) {
            dom.jobDetailError.textContent = err.message || 'Failed to load job detail';
          }
        });

        dom.refreshButton.addEventListener('click', async () => {
          try {
            await dashboardController.load();
          } catch (err) {
            if (err.status === 401) {
              shell.setAuthed(false);
              dom.authError.textContent = 'Session expired. Please login again.';
              return;
            }
            dom.authError.textContent = err.message || 'Refresh failed';
          }
        });
      },
    };

    const dashboardData = {
      async loadSiteEnvironmentBackupData() {
        const sites = await api.request('/sites');
        sitesView.render(sites);

        const environmentBySite = await Promise.all((Array.isArray(sites) ? sites : []).map((site) => {
          return api.request('/sites/' + encodeURIComponent(site.id) + '/environments')
            .then((environments) => ({ siteID: site.id, environments }))
            .catch(() => ({ siteID: site.id, environments: [] }));
        }));

        util.resetMap(state.siteEnvironmentsBySite);
        environmentBySite.forEach((entry) => {
          state.siteEnvironmentsBySite[entry.siteID] = Array.isArray(entry.environments) ? entry.environments : [];
        });

        environmentsView.renderTable();

        const environmentIDs = environmentBySite.flatMap((entry) => {
          const envs = Array.isArray(entry.environments) ? entry.environments : [];
          return envs.map((environment) => environment.id).filter(Boolean);
        });

        const backupsByEnvironment = await Promise.all(environmentIDs.map((environmentID) => {
          return api.request('/environments/' + encodeURIComponent(environmentID) + '/backups')
            .then((backups) => ({ environmentID, backups }))
            .catch(() => ({ environmentID, backups: [] }));
        }));

        util.resetMap(state.backupsByEnvironment);
        backupsByEnvironment.forEach((entry) => {
          state.backupsByEnvironment[entry.environmentID] = Array.isArray(entry.backups) ? entry.backups : [];
        });

        const wpVersionByEnvironment = await Promise.all(environmentIDs.map((environmentID) => {
          return api.request('/environments/' + encodeURIComponent(environmentID) + '/wordpress-version')
            .then((payload) => ({
              environmentID,
              version: payload && payload.wordpress_version ? payload.wordpress_version : '',
              error: '',
            }))
            .catch((err) => ({
              environmentID,
              version: '',
              error: err && err.code ? err.code : 'query_failed',
            }));
        }));

        util.resetMap(state.wpVersionByEnvironment);
        wpVersionByEnvironment.forEach((entry) => {
          state.wpVersionByEnvironment[entry.environmentID] = {
            version: entry.version,
            error: entry.error,
          };
        });

        backupsView.renderTable();
        sitesView.render(sites);
      },
    };

    const authController = {
      bind() {
        dom.loginForm.addEventListener('submit', async (event) => {
          event.preventDefault();
          dom.authError.textContent = '';

          try {
            await api.request('/login', {
              method: 'POST',
              body: JSON.stringify({
                email: document.getElementById('email').value,
                password: document.getElementById('password').value,
              }),
            });
            await dashboardController.load();
          } catch (err) {
            dom.authError.textContent = err.message || 'Login failed';
          }
        });

        dom.logoutButton.addEventListener('click', async () => {
          try {
            await api.request('/logout', { method: 'POST' });
          } catch (_) {}
          dashboardController.reset();
          shell.setAuthed(false);
        });
      },
    };

    const dashboardController = {
      async load() {
        shell.applySubsite(window.location.pathname, window.location.search);
        const [metrics, jobs, nodes, providers] = await Promise.all([
          api.request('/metrics'),
          api.request('/jobs'),
          api.request('/nodes'),
          api.request('/providers'),
        ]);

        overviewView.renderMetrics(metrics);
        nodesView.render(nodes);
        providersView.render(providers);
        jobsView.renderJobsTable(jobs);
        await dashboardData.loadSiteEnvironmentBackupData();

        if (state.selectedJobID) {
          try {
            await jobsView.loadDetail(state.selectedJobID);
          } catch (err) {
            dom.jobDetailError.textContent = err.message || 'Failed to load selected job';
          }
        } else {
          jobsView.clearDetail('Select a job from the table.');
        }

        shell.setAuthed(true);
        siteDetailView.applyFocus();
      },
      reset() {
        overviewView.clear();
        providersView.clear();
        nodesView.clear();
        sitesView.clear();
        environmentsView.clear();
        backupsView.clear();
        jobsView.clear();

        util.resetMap(state.nodesByID);
        util.resetMap(state.providersByID);
        util.resetMap(state.siteEnvironmentsBySite);
        util.resetMap(state.backupsByEnvironment);
        util.resetMap(state.wpVersionByEnvironment);
        state.selectedJobID = '';
        state.selectedSiteID = '';
        state.selectedBackupEnvironmentID = '';
        dom.authError.textContent = '';
      },
      bind() {
        shell.bindNavigation();
        authController.bind();
        providersView.bind();
        nodesView.bind();
        sitesView.bind();
        environmentsView.bind();
        backupsView.bind();
        jobsView.bind();
      },
      async init() {
        dashboardController.bind();
        try {
          await dashboardController.load();
        } catch (err) {
          if (err.status === 401) {
            shell.setAuthed(false);
            return;
          }
          dom.authError.textContent = err.message || 'Initialization failed';
          shell.setAuthed(false);
        }
      },
    };

    dashboardController.init();
  </script>
</body>
</html>`

type Server struct {
	httpServer *http.Server
	logger     *log.Logger
	addr       string
	worker     *jobs.Worker
	workerWg   sync.WaitGroup
	workerStop chan struct{}
}

func New(addr string, logger *log.Logger) *Server {
	sessionStore := store.NewInMemorySessionStore()
	authService := auth.NewService(sessionStore, "", "", 24*time.Hour)
	jobStore := jobs.NewInMemoryRepository(nil)
	nodeStore := nodes.NewInMemoryStore(nil)
	providerStore := providers.NewInMemoryStore(nil)
	siteStore := store.NewInMemorySiteStore(0)
	metricsService := metrics.NewService(jobStore, nodeStore, siteStore)
	auditStore := audit.NewInMemoryStore()
	auditService := audit.NewService(auditStore)
	backupStore := store.NewInMemoryBackupStore()
	restoreRequestStore := store.NewInMemoryRestoreRequestStore()
	apiHandler := api.NewRouter(logger, authService, jobStore, metricsService, auditService, nodeStore, providerStore)

	// Build job handlers for worker execution
	siteExecutor := sites.NewAnsibleSiteCreateExecutor()
	siteHandler := sites.NewSiteCreateHandler(store.DefaultSiteStore(), nodeStore, siteExecutor, logger)

	envExecutor := environments.NewAnsibleEnvCreateExecutor()
	envHandler := environments.NewEnvCreateHandler(store.DefaultSiteStore(), nodeStore, envExecutor, logger)

	backupExecutor := backups.NewAnsibleExecutor()
	backupHandler := backups.NewHandler(backupStore, backupExecutor)

	restoreExecutor := environments.NewAnsibleEnvRestoreExecutor()
	restoreHandler := environments.NewEnvRestoreHandler(store.DefaultSiteStore(), nodeStore, backupStore, restoreRequestStore, restoreExecutor, logger)

	nodeExecutor := nodes.NewAnsibleExecutor()
	nodeHandler := nodes.NewProvisionHandler(nodeStore, nodeExecutor, providerStore, nil, logger)

	handlers := map[string]jobs.Handler{
		"node_provision": nodeHandler.Handle,
		"site_create":    siteHandler.Handle,
		"env_create":     envHandler.Handle,
		"backup_create":  backupHandler.Handle,
		"env_restore":    restoreHandler.Handle,
	}

	worker := jobs.NewWorker(jobStore, "dev-worker", handlers, auditService)

	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", apiHandler))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !isDashboardRoute(r.URL.Path) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(dashboardHTML))
	})

	wrapped := requestLogger(logger, mux)

	return &Server{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: wrapped,
		},
		logger:     logger,
		addr:       addr,
		worker:     worker,
		workerStop: make(chan struct{}),
	}
}

func isDashboardRoute(path string) bool {
	switch path {
	case "/", "/providers", "/nodes", "/sites", "/jobs":
		return true
	default:
		if len(path) <= len("/sites/") || path[:len("/sites/")] != "/sites/" {
			return false
		}
		tail := path[len("/sites/"):]
		return tail != "" && !containsSlash(tail)
	}
}

func containsSlash(value string) bool {
	for _, r := range value {
		if r == '/' {
			return true
		}
	}
	return false
}

func seedJobs() []jobs.Job {
	now := time.Now().UTC()
	fiveMinutesAgo := now.Add(-5 * time.Minute)
	twoMinutesAgo := now.Add(-2 * time.Minute)
	lastMinute := now.Add(-1 * time.Minute)

	siteID := "11111111-1111-1111-1111-111111111111"
	environmentID := "22222222-2222-2222-2222-222222222222"
	nodeID := "33333333-3333-3333-3333-333333333333"
	workerID := "worker-1"
	errorCode := "node_unreachable"
	errorMessage := "ssh timeout while provisioning node"

	return []jobs.Job{
		{
			ID:            "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			JobType:       "node_provision",
			Status:        jobs.StatusQueued,
			SiteID:        nil,
			EnvironmentID: nil,
			NodeID:        &nodeID,
			AttemptCount:  0,
			MaxAttempts:   3,
			RunAfter:      nil,
			LockedAt:      nil,
			LockedBy:      nil,
			StartedAt:     nil,
			FinishedAt:    nil,
			ErrorCode:     nil,
			ErrorMessage:  nil,
			CreatedAt:     fiveMinutesAgo,
			UpdatedAt:     fiveMinutesAgo,
		},
		{
			ID:            "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
			JobType:       "environment_deploy",
			Status:        jobs.StatusRunning,
			SiteID:        &siteID,
			EnvironmentID: &environmentID,
			NodeID:        &nodeID,
			AttemptCount:  1,
			MaxAttempts:   3,
			RunAfter:      nil,
			LockedAt:      &twoMinutesAgo,
			LockedBy:      &workerID,
			StartedAt:     &twoMinutesAgo,
			FinishedAt:    nil,
			ErrorCode:     nil,
			ErrorMessage:  nil,
			CreatedAt:     twoMinutesAgo,
			UpdatedAt:     lastMinute,
		},
		{
			ID:            "cccccccc-cccc-cccc-cccc-cccccccccccc",
			JobType:       "environment_deploy",
			Status:        jobs.StatusFailed,
			SiteID:        &siteID,
			EnvironmentID: &environmentID,
			NodeID:        &nodeID,
			AttemptCount:  3,
			MaxAttempts:   3,
			RunAfter:      nil,
			LockedAt:      &fiveMinutesAgo,
			LockedBy:      &workerID,
			StartedAt:     &fiveMinutesAgo,
			FinishedAt:    &twoMinutesAgo,
			ErrorCode:     &errorCode,
			ErrorMessage:  &errorMessage,
			CreatedAt:     fiveMinutesAgo,
			UpdatedAt:     twoMinutesAgo,
		},
	}
}

func seedNodes() []nodes.Node {
	now := time.Now().UTC()

	return []nodes.Node{
		{
			ID:                "33333333-3333-3333-3333-333333333333",
			Hostname:          "127.0.0.1",
			PublicIP:          "127.0.0.1",
			SSHPort:           22,
			SSHUser:           "ubuntu",
			SSHPrivateKeyPath: "/var/lib/pressluft/secrets/node-33333333-3333-3333-3333-333333333333.pem",
			Status:            nodes.StatusActive,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}
}

func (s *Server) Addr() string {
	return s.addr
}

func (s *Server) Start() error {
	s.logger.Printf("event=startup addr=%s", s.addr)

	// Start the worker loop in a goroutine
	s.workerWg.Add(1)
	go s.runWorkerLoop()

	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen and serve: %w", err)
	}
	return nil
}

func (s *Server) runWorkerLoop() {
	defer s.workerWg.Done()

	s.logger.Printf("event=worker_start worker_id=dev-worker")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.workerStop:
			s.logger.Printf("event=worker_stop worker_id=dev-worker")
			return
		case <-ticker.C:
			ctx := context.Background()
			processed, err := s.worker.ProcessNext(ctx)
			if err != nil {
				s.logger.Printf("event=worker_error error=%v", err)
				continue
			}
			if processed {
				s.logger.Printf("event=worker_job_processed")
			}
		}
	}
}

func requestLogger(logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now().UTC()
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)
		elapsed := time.Since(started).Milliseconds()
		logger.Printf(
			"event=request ts=%s method=%s path=%s status=%d duration_ms=%d",
			started.Format(time.RFC3339),
			r.Method,
			r.URL.Path,
			rec.statusCode,
			elapsed,
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
