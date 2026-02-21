<template>
  <section class="env-detail">
    <header class="page-header">
      <div class="title-row">
        <div>
          <p class="eyebrow">Environment</p>
          <h2 v-if="env">{{ env.name }}</h2>
          <h2 v-else>Environment</h2>
        </div>

        <NuxtLink v-if="env" class="back" :to="`/app/sites/${env.site_id}`">Back to Site</NuxtLink>
        <NuxtLink v-else class="back" to="/app/sites">Back to Sites</NuxtLink>
      </div>

      <p v-if="env" class="subtitle">
        <span class="mono">{{ env.slug }}</span>
        <span class="pill" :data-status="env.status">{{ env.status }}</span>
      </p>
    </header>

    <div class="grid">
      <article class="card" :aria-busy="isLoading">
        <header class="card-header">
          <h3>Lifecycle Actions</h3>
          <p class="hint">All actions enqueue a job. Status updates as the job runs.</p>
        </header>

        <p v-if="isLoading" role="status" aria-live="polite">Loading environment...</p>
        <p v-else-if="errorMessage" class="error" role="alert">{{ errorMessage }}</p>

        <div v-else class="actions">
          <section v-if="env && env.status === 'failed'" class="action">
            <h4>Reset Failed State</h4>
            <p class="hint">Reset is only valid for environments currently in failed state.</p>
            <button
              type="button"
              class="danger"
              data-testid="reset-env"
              :disabled="isResetting || isSubmitting"
              @click="resetEnvironment"
            >
              {{ isResetting ? "Resetting..." : "Reset Environment" }}
            </button>
            <p v-if="resetMessage" class="muted">{{ resetMessage }}</p>
            <p v-if="resetError" class="error" role="alert">{{ resetError }}</p>
          </section>

          <section class="action">
            <h4>Deploy</h4>
            <form class="form" @submit.prevent="submitDeploy">
              <label for="deploy-type">Source type</label>
              <select id="deploy-type" v-model="deploySourceType" name="source_type" :disabled="isSubmitting">
                <option value="git">git</option>
                <option value="upload">upload</option>
              </select>

              <label for="deploy-ref">Source ref</label>
              <input
                id="deploy-ref"
                v-model="deploySourceRef"
                name="source_ref"
                type="text"
                autocomplete="off"
                required
                :disabled="isSubmitting"
              />

              <button type="submit" :disabled="isSubmitting">{{ isSubmitting ? "Starting..." : "Deploy" }}</button>
            </form>
          </section>

          <section class="action">
            <h4>Updates</h4>
            <form class="form" @submit.prevent="submitUpdates">
              <label for="updates-scope">Scope</label>
              <select id="updates-scope" v-model="updatesScope" name="scope" :disabled="isSubmitting">
                <option value="core">core</option>
                <option value="plugins">plugins</option>
                <option value="themes">themes</option>
                <option value="all">all</option>
              </select>

              <button type="submit" :disabled="isSubmitting">{{ isSubmitting ? "Starting..." : "Apply Updates" }}</button>
            </form>
          </section>

          <section class="action">
            <h4>Restore</h4>
            <p class="hint">
              Requires a completed backup id owned by this environment.
            </p>
            <form class="form" @submit.prevent="submitRestore">
              <label for="restore-backup">Backup id</label>
              <input
                id="restore-backup"
                v-model="restoreBackupId"
                name="backup_id"
                type="text"
                autocomplete="off"
                required
                :disabled="isSubmitting"
              />

              <button type="submit" :disabled="isSubmitting">{{ isSubmitting ? "Starting..." : "Restore" }}</button>
            </form>
          </section>

          <section class="action">
            <h4>Drift Check</h4>
            <p class="hint">
              Current drift: <span class="mono">{{ env?.drift_status ?? "unknown" }}</span>
            </p>
            <button type="button" class="primary" :disabled="isSubmitting" @click="submitDriftCheck">
              {{ isSubmitting ? "Starting..." : "Run Drift Check" }}
            </button>
          </section>

          <section class="action">
            <h4>Promote</h4>
            <p class="hint">
              Promotion requires clean drift and a fresh full backup on the target environment.
            </p>
            <p v-if="env && env.drift_status !== 'clean'" class="warn">
              Drift is <span class="mono">{{ env.drift_status }}</span>. Promotion may be blocked.
            </p>
            <form class="form" data-testid="promote-form" @submit.prevent="submitPromote">
              <label for="promote-target">Target environment</label>
              <select
                id="promote-target"
                v-model="promoteTargetEnvId"
                name="target_environment_id"
                required
                :disabled="isSubmitting"
              >
                <option value="">Select target...</option>
                <option v-for="candidate in promoteTargets" :key="candidate.id" :value="candidate.id">
                  {{ candidate.name }} ({{ candidate.environment_type }})
                </option>
              </select>

              <button type="submit" :disabled="isSubmitting">{{ isSubmitting ? "Starting..." : "Promote" }}</button>
            </form>
          </section>

          <section class="action">
            <h4>Backups</h4>
            <p class="hint">Create backups and track retention/status fields for this environment.</p>

            <form class="form" data-testid="backup-form" @submit.prevent="submitCreateBackup">
              <label for="backup-scope">Backup scope</label>
              <select id="backup-scope" v-model="createBackupScope" name="backup_scope" :disabled="isSubmitting">
                <option value="db">db</option>
                <option value="files">files</option>
                <option value="full">full</option>
              </select>
              <button type="submit" :disabled="isSubmitting">{{ isSubmitting ? "Starting..." : "Create Backup" }}</button>
            </form>

            <p v-if="isOpsLoading" class="muted">Loading backups...</p>
            <p v-else-if="opsError" class="error" role="alert">{{ opsError }}</p>
            <p v-else-if="backups.length === 0" class="muted" aria-live="polite">No backups yet.</p>
            <ul v-else class="list">
              <li v-for="backup in backups" :key="backup.id" class="list-row">
                <div class="list-main">
                  <span class="mono">{{ backup.backup_scope }}</span>
                  <span class="pill" :data-status="backup.status">{{ backup.status }}</span>
                </div>
                <div class="list-meta">
                  <span class="mono">retention {{ backup.retention_until }}</span>
                </div>
              </li>
            </ul>
          </section>

          <section class="action">
            <h4>Domains</h4>
            <p class="hint">Add/remove domains and observe TLS lifecycle status.</p>

            <form class="form" data-testid="domain-form" @submit.prevent="submitAddDomain">
              <label for="domain-host">Hostname</label>
              <input
                id="domain-host"
                v-model="domainHostname"
                name="hostname"
                type="text"
                autocomplete="off"
                required
                :disabled="isSubmitting"
              />
              <button type="submit" :disabled="isSubmitting">{{ isSubmitting ? "Starting..." : "Add Domain" }}</button>
            </form>

            <p v-if="isOpsLoading" class="muted">Loading domains...</p>
            <p v-else-if="opsError" class="error" role="alert">{{ opsError }}</p>
            <p v-else-if="domains.length === 0" class="muted" aria-live="polite">No domains configured.</p>
            <ul v-else class="list">
              <li v-for="domain in domains" :key="domain.id" class="list-row">
                <div class="list-main">
                  <span class="mono">{{ domain.hostname }}</span>
                  <span class="pill" :data-status="domain.tls_status">{{ domain.tls_status }}</span>
                </div>
                <button type="button" class="danger" :disabled="isSubmitting" @click="submitRemoveDomain(domain.id)">
                  Remove
                </button>
              </li>
            </ul>
          </section>

          <section class="action">
            <h4>Caching</h4>
            <p class="hint">Toggle cache flags and purge caches.</p>

            <form class="form" data-testid="cache-form" @submit.prevent="submitCacheSave">
              <label class="check">
                <input v-model="cacheFastcgi" type="checkbox" :disabled="isSubmitting" />
                <span>FastCGI cache</span>
              </label>

              <label class="check">
                <input v-model="cacheRedis" type="checkbox" :disabled="isSubmitting" />
                <span>Redis object cache</span>
              </label>

              <button type="submit" :disabled="isSubmitting">{{ isSubmitting ? "Starting..." : "Save Cache Settings" }}</button>
            </form>

            <button type="button" class="secondary" :disabled="isSubmitting" @click="submitCachePurge">
              {{ isSubmitting ? "Starting..." : "Purge Cache" }}
            </button>
          </section>

          <section class="action">
            <h4>Magic Login</h4>
            <p class="hint">Synchronous node query that returns a one-time WordPress admin URL.</p>

            <button
              type="button"
              class="primary"
              data-testid="magic-login-button"
              :disabled="isMagicLoading || isSubmitting"
              @click="submitMagicLogin"
            >
              {{ isMagicLoading ? "Generating..." : "Generate Magic Login" }}
            </button>

            <p v-if="magicLoginError" class="error" role="alert">{{ magicLoginError }}</p>
            <p v-else-if="magicLogin" class="muted" aria-live="polite">
              <a class="link" :href="magicLogin.login_url" target="_blank" rel="noreferrer">Open WP Admin</a>
              <span class="mono">(expires {{ magicLogin.expires_at }})</span>
            </p>
          </section>

          <div v-if="actionError" class="error-panel" aria-live="polite" role="status">
            <p class="error" role="alert">{{ actionError }}</p>
          </div>

          <div v-if="actionJob" class="job-panel" aria-live="polite" role="status">
            <p class="job-line">
              Job <span class="mono">{{ actionJob.id }}</span>
              <span class="pill" :data-status="actionJob.status">{{ actionJob.status }}</span>
            </p>
            <p v-if="actionJob.status === 'failed'" class="error" role="alert">
              {{ actionJob.error_message || "Job failed." }}
            </p>
          </div>
        </div>
      </article>

      <article class="card">
        <header class="card-header">
          <h3>Details</h3>
        </header>

        <dl v-if="env" class="details">
          <div class="row">
            <dt>Type</dt>
            <dd>{{ env.environment_type }}</dd>
          </div>
          <div class="row">
            <dt>Preview URL</dt>
            <dd>
              <a class="link" :href="env.preview_url" target="_blank" rel="noreferrer">{{ env.preview_url }}</a>
            </dd>
          </div>
          <div class="row">
            <dt>Promotion preset</dt>
            <dd>{{ env.promotion_preset }}</dd>
          </div>
          <div class="row">
            <dt>Drift status</dt>
            <dd>{{ env.drift_status }}</dd>
          </div>
        </dl>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import type {
  AddDomainRequest,
  Backup,
  BackupScope,
  CreateBackupRequest,
  DeploySourceType,
  DeployRequest,
  Domain,
  Environment,
  JobStatusResponse,
  MagicLoginResponse,
  PatchCacheRequest,
  PromoteRequest,
  UpdateScope,
  UpdatesRequest,
  RestoreRequest,
} from "~/lib/api/types";
import { ApiClientError } from "~/lib/api/client";

definePageMeta({
  layout: "app",
});

const route = useRoute();
const api = useApiClient();
const auth = useAuthSession();

const envId = computed(() => String(route.params.id ?? ""));

const isLoading = ref(true);
const errorMessage = ref("");
const env = ref<Environment | null>(null);

const deploySourceType = ref<DeploySourceType>("git");
const deploySourceRef = ref("");
const updatesScope = ref<UpdateScope>("all");
const restoreBackupId = ref("");
const promoteTargetEnvId = ref("");

const isSubmitting = ref(false);
const actionError = ref("");
const actionJob = ref<JobStatusResponse | null>(null);
const promoteTargets = ref<Environment[]>([]);

const isResetting = ref(false);
const resetMessage = ref("");
const resetError = ref("");

const isOpsLoading = ref(false);
const opsError = ref("");
const backups = ref<Backup[]>([]);
const domains = ref<Domain[]>([]);

const createBackupScope = ref<BackupScope>("full");
const domainHostname = ref("");

const cacheFastcgi = ref(false);
const cacheRedis = ref(false);

const isMagicLoading = ref(false);
const magicLogin = ref<MagicLoginResponse | null>(null);
const magicLoginError = ref("");

const POLL_INTERVAL_MS = import.meta.env.MODE === "test" ? 0 : 1000;

let pollTimer: ReturnType<typeof setTimeout> | null = null;

onBeforeUnmount(() => {
  if (pollTimer) {
    clearTimeout(pollTimer);
    pollTimer = null;
  }
});

const handleAuthError = async (error: unknown): Promise<boolean> => {
  if (error instanceof ApiClientError && error.status === 401) {
    auth.setGuest();
    await navigateTo("/login", { replace: true });
    return true;
  }
  return false;
};

const loadPromoteTargets = async (): Promise<void> => {
  if (!env.value) return;
  try {
    const envs = await api.listSiteEnvironments(env.value.site_id);
    promoteTargets.value = envs.filter((candidate) => candidate.id !== env.value?.id);
  } catch {
    promoteTargets.value = [];
  }
};

watch(
  () => env.value,
  (next) => {
    if (!next) return;
    cacheFastcgi.value = next.fastcgi_cache_enabled;
    cacheRedis.value = next.redis_cache_enabled;
  },
  { immediate: true },
);

const loadOperations = async (): Promise<void> => {
  if (!env.value) return;

  isOpsLoading.value = true;
  opsError.value = "";
  try {
    const [backupList, domainList] = await Promise.all([
      api.listEnvironmentBackups(envId.value),
      api.listEnvironmentDomains(envId.value),
    ]);
    backups.value = backupList;
    domains.value = domainList;
  } catch (error) {
    if (await handleAuthError(error)) return;
    opsError.value = "Could not load backups or domains.";
  } finally {
    isOpsLoading.value = false;
  }
};

const loadEnvironment = async (): Promise<void> => {
  isLoading.value = true;
  errorMessage.value = "";

  try {
    env.value = await api.getEnvironment(envId.value);
    await loadPromoteTargets();
    await loadOperations();
  } catch (error) {
    if (await handleAuthError(error)) return;
    if (error instanceof ApiClientError && error.status === 404) {
      errorMessage.value = "Environment not found.";
      return;
    }
    errorMessage.value = "Could not load environment.";
  } finally {
    isLoading.value = false;
  }
};

const pollJob = async (jobId: string): Promise<void> => {
  const step = async (): Promise<void> => {
    try {
      actionJob.value = await api.getJob(jobId);
    } catch (error) {
      if (await handleAuthError(error)) return;
      actionError.value = "Could not refresh job status.";
      return;
    }

    if (!actionJob.value) return;

    if (actionJob.value.status === "queued" || actionJob.value.status === "running") {
      pollTimer = setTimeout(() => {
        void step();
      }, POLL_INTERVAL_MS);
      return;
    }

    await loadEnvironment();
  };

  await step();
};

const startAction = async (fn: () => Promise<{ job_id: string }>): Promise<void> => {
  if (pollTimer) {
    clearTimeout(pollTimer);
    pollTimer = null;
  }

  isSubmitting.value = true;
  actionError.value = "";
  actionJob.value = null;

  try {
    const { job_id } = await fn();
    await pollJob(job_id);
  } catch (error) {
    if (await handleAuthError(error)) return;
    if (error instanceof ApiClientError) {
      actionError.value = error.message;
      return;
    }
    actionError.value = "Could not start job.";
  } finally {
    isSubmitting.value = false;
  }
};

const submitDeploy = async (): Promise<void> => {
  const sourceRef = deploySourceRef.value.trim();
  if (!sourceRef) {
    actionError.value = "Source reference is required.";
    return;
  }

  const payload: DeployRequest = {
    source_type: deploySourceType.value,
    source_ref: sourceRef,
  };

  await startAction(() => api.deployEnvironment(envId.value, payload));
};

const submitUpdates = async (): Promise<void> => {
  const payload: UpdatesRequest = {
    scope: updatesScope.value,
  };
  await startAction(() => api.applyEnvironmentUpdates(envId.value, payload));
};

const submitRestore = async (): Promise<void> => {
  const backupId = restoreBackupId.value.trim();
  if (!backupId) {
    actionError.value = "Backup id is required.";
    return;
  }
  const payload: RestoreRequest = {
    backup_id: backupId,
  };
  await startAction(() => api.restoreEnvironment(envId.value, payload));
};

const submitDriftCheck = async (): Promise<void> => {
  await startAction(() => api.runDriftCheck(envId.value));
};

const submitPromote = async (): Promise<void> => {
  const targetId = promoteTargetEnvId.value.trim();
  if (!targetId) {
    actionError.value = "Target environment is required.";
    return;
  }
  const payload: PromoteRequest = {
    target_environment_id: targetId,
  };
  await startAction(() => api.promoteEnvironment(envId.value, payload));
};

const submitCreateBackup = async (): Promise<void> => {
  const payload: CreateBackupRequest = {
    backup_scope: createBackupScope.value,
  };
  await startAction(() => api.createEnvironmentBackup(envId.value, payload));
};

const submitAddDomain = async (): Promise<void> => {
  const hostname = domainHostname.value.trim();
  if (!hostname) {
    actionError.value = "Hostname is required.";
    return;
  }
  const payload: AddDomainRequest = { hostname };
  await startAction(() => api.addEnvironmentDomain(envId.value, payload));
  domainHostname.value = "";
};

const submitRemoveDomain = async (domainId: string): Promise<void> => {
  await startAction(() => api.deleteDomain(domainId));
};

const submitCacheSave = async (): Promise<void> => {
  if (!env.value) return;

  const patch: PatchCacheRequest = {};
  if (cacheFastcgi.value !== env.value.fastcgi_cache_enabled) {
    patch.fastcgi_cache_enabled = cacheFastcgi.value;
  }
  if (cacheRedis.value !== env.value.redis_cache_enabled) {
    patch.redis_cache_enabled = cacheRedis.value;
  }

  if (Object.keys(patch).length === 0) {
    actionError.value = "No cache changes to apply.";
    return;
  }

  await startAction(() => api.updateEnvironmentCache(envId.value, patch));
};

const submitCachePurge = async (): Promise<void> => {
  await startAction(() => api.purgeEnvironmentCache(envId.value));
};

const submitMagicLogin = async (): Promise<void> => {
  magicLoginError.value = "";
  magicLogin.value = null;
  isMagicLoading.value = true;
  try {
    magicLogin.value = await api.createMagicLogin(envId.value);
  } catch (error) {
    if (await handleAuthError(error)) return;
    if (error instanceof ApiClientError) {
      switch (error.code) {
        case "environment_not_active":
          magicLoginError.value = "Environment is not active.";
          return;
        case "node_unreachable":
          magicLoginError.value = "Node unreachable (SSH failed or timed out).";
          return;
        case "wp_cli_error":
          magicLoginError.value = "WP-CLI error while generating magic login.";
          return;
        default:
          magicLoginError.value = error.message;
          return;
      }
    }
    magicLoginError.value = "Could not generate magic login.";
  } finally {
    isMagicLoading.value = false;
  }
};

const resetEnvironment = async (): Promise<void> => {
  isResetting.value = true;
  resetMessage.value = "";
  resetError.value = "";

  try {
    const resp = await api.resetEnvironment(envId.value);
    resetMessage.value = resp.success ? `Reset accepted (${resp.status}).` : "Reset not accepted.";
    await loadEnvironment();
  } catch (error) {
    if (await handleAuthError(error)) return;
    if (error instanceof ApiClientError) {
      resetError.value = error.message;
      return;
    }
    resetError.value = "Could not reset environment.";
  } finally {
    isResetting.value = false;
  }
};

void loadEnvironment();
</script>

<style scoped>
.env-detail {
  display: grid;
  gap: 1rem;
}

.grid {
  display: grid;
  grid-template-columns: minmax(280px, 520px) 1fr;
  gap: 1rem;
  align-items: start;
}

.title-row {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
}

.eyebrow {
  margin: 0;
  font-size: 0.75rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #475569;
}

.page-header h2 {
  margin: 0.25rem 0 0;
}

.subtitle {
  margin: 0.65rem 0 0;
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: center;
  color: #475569;
}

.card {
  border-radius: 0.9rem;
  border: 1px solid #e2e8f0;
  background: #ffffff;
  padding: 1rem;
  box-shadow: 0 20px 36px -34px rgba(15, 23, 42, 0.7);
}

.card-header h3 {
  margin: 0;
}

.hint {
  margin: 0.35rem 0 0;
  color: #475569;
  font-size: 0.95rem;
}

.actions {
  display: grid;
  gap: 1rem;
  margin-top: 1rem;
}

.action {
  padding: 0.85rem;
  border-radius: 0.85rem;
  border: 1px solid rgba(226, 232, 240, 0.9);
  background: rgba(248, 250, 252, 0.6);
}

.action h4 {
  margin: 0;
  font-size: 1rem;
}

.form {
  display: grid;
  gap: 0.65rem;
  margin-top: 0.65rem;
}

label {
  font-size: 0.9rem;
  color: #334155;
}

input,
select {
  width: 100%;
  border: 1px solid #cbd5e1;
  border-radius: 0.625rem;
  padding: 0.65rem 0.75rem;
  font-size: 1rem;
  background: #ffffff;
}

input:focus-visible,
select:focus-visible {
  outline: 2px solid #0f766e;
  outline-offset: 1px;
  border-color: #0f766e;
}

button {
  border: 0;
  border-radius: 0.75rem;
  padding: 0.7rem 0.95rem;
  font-size: 1rem;
  font-weight: 700;
  color: #ffffff;
  background: #0f766e;
  cursor: pointer;
}

button:disabled {
  opacity: 0.7;
  cursor: wait;
}

.primary {
  width: 100%;
  margin-top: 0.65rem;
}

.secondary {
  width: 100%;
  margin-top: 0.65rem;
  background: #0f172a;
}

.danger {
  border: 0;
  border-radius: 0.75rem;
  padding: 0.55rem 0.75rem;
  font-size: 0.95rem;
  font-weight: 800;
  color: #ffffff;
  background: #b91c1c;
  cursor: pointer;
}

.danger:disabled,
.secondary:disabled {
  opacity: 0.7;
  cursor: wait;
}

.check {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  font-size: 0.95rem;
  color: #0f172a;
}

.check input {
  width: 1.1rem;
  height: 1.1rem;
}

.muted {
  margin: 0.65rem 0 0;
  color: #475569;
}

.list {
  margin: 0.85rem 0 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: 0.55rem;
}

.list-row {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 0.75rem;
  align-items: center;
  padding: 0.65rem 0.75rem;
  border-radius: 0.75rem;
  border: 1px solid rgba(226, 232, 240, 0.9);
  background: rgba(255, 255, 255, 0.7);
}

.list-main {
  display: flex;
  flex-wrap: wrap;
  gap: 0.65rem;
  align-items: center;
  min-width: 0;
}

.list-meta {
  margin-top: 0.35rem;
  color: #475569;
  font-size: 0.9rem;
}

.details {
  margin: 0;
  display: grid;
  gap: 0.85rem;
}

.row {
  display: grid;
  grid-template-columns: 180px 1fr;
  gap: 0.75rem;
}

dt {
  font-weight: 800;
  color: #0f172a;
}

dd {
  margin: 0;
  color: #334155;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
}

.link {
  color: #0f766e;
  text-decoration: none;
  word-break: break-all;
}

.link:hover {
  text-decoration: underline;
}

.pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.25rem 0.55rem;
  border-radius: 999px;
  border: 1px solid rgba(226, 232, 240, 0.9);
  background: rgba(255, 255, 255, 0.7);
  font-size: 0.85rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.pill[data-status="succeeded"],
.pill[data-status="completed"],
.pill[data-status="active"] {
  border-color: rgba(15, 118, 110, 0.35);
  background: rgba(240, 253, 250, 0.9);
  color: #0f766e;
}

.pill[data-status="failed"],
.pill[data-status="cancelled"] {
  border-color: rgba(185, 28, 28, 0.25);
  background: rgba(254, 242, 242, 1);
  color: #b91c1c;
}

.error {
  margin: 0;
  color: #b91c1c;
}

.warn {
  margin: 0.65rem 0 0;
  padding: 0.55rem 0.65rem;
  border-radius: 0.75rem;
  border: 1px solid rgba(245, 158, 11, 0.35);
  background: rgba(255, 251, 235, 1);
  color: #92400e;
  font-size: 0.95rem;
}

.job-panel {
  padding-top: 0.85rem;
  border-top: 1px solid rgba(226, 232, 240, 0.9);
}

.job-line {
  margin: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
}

.error-panel {
  padding-top: 0.85rem;
  border-top: 1px solid rgba(226, 232, 240, 0.9);
}

.back {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.55rem 0.8rem;
  border-radius: 999px;
  border: 1px solid rgba(226, 232, 240, 0.9);
  background: rgba(255, 255, 255, 0.7);
  text-decoration: none;
  color: #0f172a;
  font-weight: 700;
}

@media (max-width: 720px) {
  .grid {
    grid-template-columns: 1fr;
  }

  .row {
    grid-template-columns: 1fr;
  }

  .title-row {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
