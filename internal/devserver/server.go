package devserver

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"pressluft/internal/api"
	"pressluft/internal/audit"
	"pressluft/internal/auth"
	"pressluft/internal/jobs"
	"pressluft/internal/metrics"
	"pressluft/internal/nodes"
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
          <p class="muted">Wave 5 dashboard IA overhaul in progress with route-level subsites.</p>
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
          <a class="subsite-link" data-nav="sites" href="/sites">Sites</a>
          <a class="subsite-link" data-nav="environments" href="/environments">Environments</a>
          <a class="subsite-link" data-nav="backups" href="/backups">Backups</a>
          <a class="subsite-link" data-nav="jobs" href="/jobs">Jobs</a>
        </nav>

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
                <th>Primary Env</th>
              </tr>
            </thead>
            <tbody id="sites-body"></tbody>
          </table>
        </section>

        <section data-subsite="environments" id="subsite-environments">
          <article class="panel">
            <h2>Create Environment</h2>
            <form id="environment-form">
              <label>Site
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

        <section class="panel" data-subsite="environments" style="margin-bottom: 14px;">
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

        <section class="panel" data-subsite="backups" id="subsite-backups" style="margin-bottom: 14px;">
          <div class="head">
            <h2>Backups</h2>
            <p id="backup-title" class="muted">Select a site and environment to create and view backups.</p>
          </div>
          <form id="backup-form">
            <label>Site
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
      environmentForm: document.getElementById('environment-form'),
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
      backupSite: document.getElementById('backup-site'),
      backupEnvironment: document.getElementById('backup-environment'),
      backupScope: document.getElementById('backup-scope'),
      backupTitle: document.getElementById('backup-title'),
      backupSuccess: document.getElementById('backup-success'),
      backupError: document.getElementById('backup-error'),
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
      activeSubsite: 'overview',
      siteEnvironmentsBySite: {},
      backupsByEnvironment: {},
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
      normalizeSubsite(pathname) {
        switch (pathname) {
          case '/':
            return 'overview';
          case '/sites':
            return 'sites';
          case '/environments':
            return 'environments';
          case '/backups':
            return 'backups';
          case '/jobs':
            return 'jobs';
          default:
            return 'overview';
        }
      },
      applySubsite(pathname) {
        state.activeSubsite = shell.normalizeSubsite(pathname);

        dom.subsiteSections.forEach((section) => {
          const sectionName = section.getAttribute('data-subsite') || '';
          section.classList.toggle('hidden', sectionName !== state.activeSubsite);
        });

        dom.subsiteLinks.forEach((link) => {
          const linkName = link.getAttribute('data-nav') || '';
          const isActive = linkName === state.activeSubsite;
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
          shell.applySubsite(window.location.pathname);
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
          shell.applySubsite(nextPath);
        });

        window.addEventListener('popstate', () => {
          shell.applySubsite(window.location.pathname);
        });
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
            state.selectedSiteID = sites.length > 0 ? (sites[0].id || '') : '';
          }
        }
        dom.environmentSite.value = state.selectedSiteID || '';
        dom.backupSite.value = state.selectedSiteID || '';
      },
      renderSitesTable(sites) {
        if (sites.length === 0) {
          dom.sitesBody.innerHTML = '<tr><td colspan="4" class="muted">No sites available.</td></tr>';
          return;
        }

        dom.sitesBody.innerHTML = sites.map((site) => {
          const status = String(site.status || 'unknown');
          const primary = site.primary_environment_id || '-';
          return [
            '<tr>',
            '<td>' + util.escapeHTML(site.name || '-') + '</td>',
            '<td>' + util.escapeHTML(site.slug || '-') + '</td>',
            '<td><span class="status ' + util.escapeHTML(status) + '">' + util.escapeHTML(status) + '</span></td>',
            '<td>' + util.escapeHTML(primary) + '</td>',
            '</tr>',
          ].join('');
        }).join('');
      },
      render(sites) {
        const list = Array.isArray(sites) ? sites : [];
        dom.siteCount.textContent = String(list.length) + (list.length === 1 ? ' site' : ' sites');
        sitesView.renderSiteOptions(list);
        sitesView.renderSitesTable(list);
      },
      clearMessages() {
        dom.siteError.textContent = '';
        dom.siteSuccess.textContent = '';
      },
      clear() {
        dom.sitesBody.innerHTML = '';
        dom.siteCount.textContent = '0 sites';
        dom.environmentSite.innerHTML = '<option value="">Select a site</option>';
        dom.backupSite.innerHTML = '<option value="">Select site</option>';
        sitesView.clearMessages();
      },
      bind() {
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
      renderSourceOptions() {
        const envs = state.siteEnvironmentsBySite[state.selectedSiteID] || [];
        dom.environmentSource.innerHTML = ['<option value="">Select source environment</option>'].concat(
          envs.map((environment) => '<option value="' + util.escapeHTML(environment.id) + '">' + util.escapeHTML(environment.name) + ' (' + util.escapeHTML(environment.slug) + ')</option>')
        ).join('');
      },
      renderTable() {
        const envs = state.siteEnvironmentsBySite[state.selectedSiteID] || [];
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
        environmentsView.clearMessages();
      },
      bind() {
        dom.environmentSite.addEventListener('change', () => {
          state.selectedSiteID = dom.environmentSite.value;
          dom.backupSite.value = state.selectedSiteID;
          state.selectedBackupEnvironmentID = '';
          environmentsView.renderTable();
          backupsView.renderTable();
        });

        dom.environmentForm.addEventListener('submit', async (event) => {
          event.preventDefault();
          environmentsView.clearMessages();

          const siteID = dom.environmentSite.value;
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
      },
      renderTable() {
        const envs = state.siteEnvironmentsBySite[state.selectedSiteID] || [];
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
      },
      clear() {
        dom.backupsBody.innerHTML = '';
        dom.backupTitle.textContent = 'Select a site and environment to create and view backups.';
        dom.backupSite.innerHTML = '<option value="">Select site</option>';
        dom.backupEnvironment.innerHTML = '<option value="">Select environment</option>';
        backupsView.clearMessages();
      },
      bind() {
        dom.backupSite.addEventListener('change', () => {
          state.selectedSiteID = dom.backupSite.value;
          dom.environmentSite.value = state.selectedSiteID;
          state.selectedBackupEnvironmentID = '';
          environmentsView.renderTable();
          backupsView.renderTable();
        });

        dom.backupEnvironment.addEventListener('change', () => {
          state.selectedBackupEnvironmentID = dom.backupEnvironment.value;
          backupsView.renderTable();
        });

        dom.backupForm.addEventListener('submit', async (event) => {
          event.preventDefault();
          backupsView.clearMessages();

          const environmentID = dom.backupEnvironment.value;
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

        backupsView.renderTable();
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
        const [metrics, jobs] = await Promise.all([
          api.request('/metrics'),
          api.request('/jobs'),
        ]);

        overviewView.renderMetrics(metrics);
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
      },
      reset() {
        overviewView.clear();
        sitesView.clear();
        environmentsView.clear();
        backupsView.clear();
        jobsView.clear();

        util.resetMap(state.siteEnvironmentsBySite);
        util.resetMap(state.backupsByEnvironment);
        state.selectedJobID = '';
        state.selectedSiteID = '';
        state.selectedBackupEnvironmentID = '';
        dom.authError.textContent = '';
      },
      bind() {
        shell.bindNavigation();
        authController.bind();
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
}

func New(addr string, logger *log.Logger) *Server {
	sessionStore := store.NewInMemorySessionStore()
	authService := auth.NewService(sessionStore, "", "", 24*time.Hour)
	jobStore := jobs.NewInMemoryRepository(seedJobs())
	nodeStore := nodes.NewInMemoryStore(seedNodes())
	siteStore := store.NewInMemorySiteStore(1)
	metricsService := metrics.NewService(jobStore, nodeStore, siteStore)
	auditStore := audit.NewInMemoryStore()
	auditService := audit.NewService(auditStore)
	apiHandler := api.NewRouter(logger, authService, jobStore, metricsService, auditService)

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
		logger: logger,
		addr:   addr,
	}
}

func isDashboardRoute(path string) bool {
	switch path {
	case "/", "/sites", "/environments", "/backups", "/jobs":
		return true
	default:
		return false
	}
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
	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen and serve: %w", err)
	}
	return nil
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
