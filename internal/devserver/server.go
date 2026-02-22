package devserver

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"pressluft/internal/api"
	"pressluft/internal/auth"
	"pressluft/internal/jobs"
	"pressluft/internal/metrics"
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
      max-width: 380px;
    }

    label { display: grid; gap: 6px; font-weight: 600; }

    input {
      width: 100%;
      border-radius: 10px;
      border: 1px solid var(--line);
      padding: 10px;
      font: inherit;
      color: var(--ink);
      background: #0f1925;
    }

    input:focus-visible, .btn:focus-visible {
      outline: 2px solid #59b6ff;
      outline-offset: 2px;
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

    th { color: var(--ink-soft); font-size: 0.82rem; text-transform: uppercase; letter-spacing: 0.04em; }

    .status { font-weight: 700; text-transform: capitalize; }
    .status.running { color: var(--brand-2); }
    .status.queued { color: #f0bc53; }
    .status.succeeded { color: var(--ok); }
    .status.failed { color: var(--danger); }

    .error {
      color: var(--danger);
      font-weight: 600;
      min-height: 1.2em;
      margin-top: 4px;
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
          <p class="muted">Wave 2 dashboard with authentication, jobs, and metrics visibility.</p>
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
        <div class="grid" id="metrics"></div>
        <section class="panel">
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
      </section>
    </section>
  </main>

  <script>
    const authPanel = document.getElementById('auth-panel');
    const dashboard = document.getElementById('dashboard');
    const authError = document.getElementById('auth-error');
    const loginForm = document.getElementById('login-form');
    const logoutButton = document.getElementById('logout');
    const refreshButton = document.getElementById('refresh');
    const metricsEl = document.getElementById('metrics');
    const jobsBody = document.getElementById('jobs-body');

    const metricCards = [
      { key: 'jobs_running', label: 'Jobs Running' },
      { key: 'jobs_queued', label: 'Jobs Queued' },
      { key: 'nodes_active', label: 'Nodes Active' },
      { key: 'sites_total', label: 'Sites Total' },
    ];

    async function request(path, options = {}) {
      const response = await fetch('/api' + path, {
        ...options,
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          ...(options.headers || {}),
        },
      });

      let body = null;
      try { body = await response.json(); } catch (_) {}

      if (!response.ok) {
        const err = new Error((body && body.message) || ('request failed: ' + response.status));
        err.status = response.status;
        throw err;
      }

      return body;
    }

    function setAuthed(authed) {
      authPanel.classList.toggle('hidden', authed);
      dashboard.classList.toggle('hidden', !authed);
      logoutButton.classList.toggle('hidden', !authed);
    }

    function renderMetrics(metrics) {
      metricsEl.innerHTML = metricCards.map((item) => {
        const value = Number.isFinite(metrics[item.key]) ? metrics[item.key] : 0;
        return '<article class="metric"><b>' + value + '</b><span class="muted">' + item.label + '</span></article>';
      }).join('');
    }

    function renderJobs(jobs) {
      if (!Array.isArray(jobs) || jobs.length === 0) {
        jobsBody.innerHTML = '<tr><td colspan="5" class="muted">No jobs available.</td></tr>';
        return;
      }

      jobsBody.innerHTML = jobs.map((job) => {
        const status = String(job.status || 'unknown');
        const attempts = String(job.attempt_count || 0) + '/' + String(job.max_attempts || 0);
        const err = job.error_code || '-';
        return [
          '<tr>',
          '<td>' + (job.id || '-') + '</td>',
          '<td>' + (job.job_type || '-') + '</td>',
          '<td><span class="status ' + status + '">' + status + '</span></td>',
          '<td>' + attempts + '</td>',
          '<td>' + err + '</td>',
          '</tr>',
        ].join('');
      }).join('');
    }

    async function loadDashboard() {
      const [metrics, jobs] = await Promise.all([
        request('/metrics'),
        request('/jobs'),
      ]);
      renderMetrics(metrics);
      renderJobs(jobs);
      setAuthed(true);
    }

    loginForm.addEventListener('submit', async (event) => {
      event.preventDefault();
      authError.textContent = '';
      const email = document.getElementById('email').value;
      const password = document.getElementById('password').value;

      try {
        await request('/login', {
          method: 'POST',
          body: JSON.stringify({ email, password }),
        });
        await loadDashboard();
      } catch (err) {
        authError.textContent = err.message || 'Login failed';
      }
    });

    logoutButton.addEventListener('click', async () => {
      try {
        await request('/logout', { method: 'POST' });
      } catch (_) {}
      setAuthed(false);
      jobsBody.innerHTML = '';
      metricsEl.innerHTML = '';
    });

    refreshButton.addEventListener('click', async () => {
      try {
        await loadDashboard();
      } catch (err) {
        if (err.status === 401) {
          setAuthed(false);
          authError.textContent = 'Session expired. Please login again.';
          return;
        }
        authError.textContent = err.message || 'Refresh failed';
      }
    });

    (async function init() {
      try {
        await loadDashboard();
      } catch (err) {
        if (err.status === 401) {
          setAuthed(false);
          return;
        }
        authError.textContent = err.message || 'Initialization failed';
        setAuthed(false);
      }
    })();
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
	nodeStore := store.NewInMemoryNodeStore(1)
	siteStore := store.NewInMemorySiteStore(1)
	metricsService := metrics.NewService(jobStore, nodeStore, siteStore)
	apiHandler := api.NewRouter(logger, authService, jobStore, metricsService)

	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", apiHandler))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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
