<template>
  <section class="site-detail">
    <header class="page-header">
      <div class="title-row">
        <div>
          <p class="eyebrow">Site</p>
          <h2 v-if="site">{{ site.name }}</h2>
          <h2 v-else>Site</h2>
        </div>

        <NuxtLink class="back" to="/app/sites">Back to Sites</NuxtLink>
      </div>

      <p v-if="site" class="subtitle">
        <span class="mono">{{ site.slug }}</span>
        <span class="pill" :data-status="site.status">{{ site.status }}</span>
      </p>
    </header>

    <div class="grid">
      <article class="card">
        <header class="card-header">
          <h3>Create environment</h3>
          <p class="hint">Create staging or clone environments for this site.</p>
        </header>

        <div v-if="site && site.status === 'failed'" class="reset-panel" aria-live="polite">
          <p class="warn">Site is in failed state. You can attempt a reset after validation.</p>
          <button type="button" class="danger" :disabled="isResetting" @click="resetSite">
            {{ isResetting ? "Resetting..." : "Reset Site" }}
          </button>
          <p v-if="resetMessage" class="muted">{{ resetMessage }}</p>
          <p v-if="resetError" class="error" role="alert">{{ resetError }}</p>
        </div>

        <form class="form" @submit.prevent="submitCreateEnvironment">
          <label for="env-type">Type</label>
          <select id="env-type" v-model="createType" name="type" :disabled="isCreating">
            <option value="staging">staging</option>
            <option value="clone">clone</option>
          </select>

          <label for="env-name">Name</label>
          <input
            id="env-name"
            v-model="createName"
            name="name"
            type="text"
            autocomplete="off"
            required
            :disabled="isCreating"
          />

          <label for="env-slug">Slug</label>
          <input
            id="env-slug"
            v-model="createSlug"
            name="slug"
            type="text"
            autocomplete="off"
            required
            :disabled="isCreating"
            @input="slugTouched = true"
          />

          <label for="env-preset">Promotion preset</label>
          <select id="env-preset" v-model="createPreset" name="promotion_preset" :disabled="isCreating">
            <option value="content-protect">content-protect</option>
            <option value="commerce-protect">commerce-protect</option>
          </select>

          <div v-if="createType === 'clone'" class="inline">
            <label for="env-source">Source environment</label>
            <select id="env-source" v-model="createSourceEnvId" name="source_environment_id" :disabled="isCreating">
              <option value="">Select source...</option>
              <option v-for="env in environments" :key="env.id" :value="env.id">
                {{ env.name }} ({{ env.environment_type }})
              </option>
            </select>
          </div>

          <p v-if="createError" class="error" role="alert">{{ createError }}</p>

          <button type="submit" :disabled="isCreating">
            {{ isCreating ? "Creating..." : "Create Environment" }}
          </button>
        </form>

        <div v-if="createJob" class="job-panel" aria-live="polite">
          <p class="job-line">
            Job <span class="mono">{{ createJob.id }}</span>
            <span class="pill" :data-status="createJob.status">{{ createJob.status }}</span>
          </p>
          <p v-if="createJob.status === 'failed'" class="error" role="alert">
            {{ createJob.error_message || "Environment creation failed." }}
          </p>
        </div>
      </article>

      <article class="card" :aria-busy="isLoading">
        <header class="card-header">
          <h3>Environments</h3>
          <p class="hint">List refreshes after a creation job completes.</p>
        </header>

        <p v-if="isLoading" role="status" aria-live="polite">Loading site...</p>
        <p v-else-if="loadError" class="error" role="alert">{{ loadError }}</p>

        <div v-else>
          <p v-if="environments.length === 0" class="muted" aria-live="polite">No environments yet.</p>
          <ul v-else class="env-list">
            <li v-for="env in environments" :key="env.id" class="env-row">
              <NuxtLink class="env-link" :to="`/app/environments/${env.id}`">
                <span class="env-name">{{ env.name }}</span>
                <span class="mono env-slug">{{ env.slug }}</span>
                <span class="meta">{{ env.environment_type }}</span>
              </NuxtLink>
              <span class="pill" :data-status="env.status">{{ env.status }}</span>
            </li>
          </ul>
        </div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import type {
  CreateEnvironmentRequest,
  Environment,
  EnvironmentCreateType,
  JobStatusResponse,
  PromotionPreset,
  Site,
} from "~/lib/api/types";
import { ApiClientError } from "~/lib/api/client";

definePageMeta({
  layout: "app",
});

const route = useRoute();
const api = useApiClient();
const auth = useAuthSession();

const siteId = computed(() => String(route.params.id ?? ""));

const isLoading = ref(true);
const loadError = ref("");
const site = ref<Site | null>(null);
const environments = ref<Environment[]>([]);

const createType = ref<EnvironmentCreateType>("staging");
const createName = ref("");
const createSlug = ref("");
const createPreset = ref<PromotionPreset>("content-protect");
const createSourceEnvId = ref("");
const slugTouched = ref(false);
const isCreating = ref(false);
const createError = ref("");
const createJob = ref<JobStatusResponse | null>(null);

const isResetting = ref(false);
const resetMessage = ref("");
const resetError = ref("");

const POLL_INTERVAL_MS = import.meta.env.MODE === "test" ? 0 : 1000;

let pollTimer: ReturnType<typeof setTimeout> | null = null;

const slugify = (value: string): string => {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
};

watch(
  () => createName.value,
  (next) => {
    if (!slugTouched.value) {
      createSlug.value = slugify(next);
    }
  },
);

watch(
  () => createType.value,
  (next) => {
    if (next !== "clone") {
      createSourceEnvId.value = "";
    }
  },
);

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

const loadAll = async (): Promise<void> => {
  isLoading.value = true;
  loadError.value = "";

  try {
    const [siteResp, envResp] = await Promise.all([
      api.getSite(siteId.value),
      api.listSiteEnvironments(siteId.value),
    ]);
    site.value = siteResp;
    environments.value = envResp;
  } catch (error) {
    if (await handleAuthError(error)) return;
    if (error instanceof ApiClientError && error.status === 404) {
      loadError.value = "Site not found.";
      return;
    }
    loadError.value = "Could not load site.";
  } finally {
    isLoading.value = false;
  }
};

const pollJob = async (jobId: string): Promise<void> => {
  const step = async (): Promise<void> => {
    try {
      createJob.value = await api.getJob(jobId);
    } catch (error) {
      if (await handleAuthError(error)) return;
      createError.value = "Could not refresh job status.";
      return;
    }

    if (!createJob.value) return;

    if (createJob.value.status === "queued" || createJob.value.status === "running") {
      pollTimer = setTimeout(() => {
        void step();
      }, POLL_INTERVAL_MS);
      return;
    }

    await loadAll();
  };

  await step();
};

const submitCreateEnvironment = async (): Promise<void> => {
  if (pollTimer) {
    clearTimeout(pollTimer);
    pollTimer = null;
  }

  isCreating.value = true;
  createError.value = "";
  createJob.value = null;

  const name = createName.value.trim();
  const slug = createSlug.value.trim();
  if (!name || !slug) {
    createError.value = "Name and slug are required.";
    isCreating.value = false;
    return;
  }

  if (createType.value === "clone" && !createSourceEnvId.value) {
    createError.value = "Clone requires a source environment.";
    isCreating.value = false;
    return;
  }

  const payload: CreateEnvironmentRequest = {
    name,
    slug,
    type: createType.value,
    promotion_preset: createPreset.value,
    source_environment_id: createType.value === "clone" ? createSourceEnvId.value : null,
  };

  try {
    const { job_id } = await api.createEnvironment(siteId.value, payload);
    await pollJob(job_id);
  } catch (error) {
    if (await handleAuthError(error)) return;
    if (error instanceof ApiClientError && error.status === 400) {
      createError.value = error.message;
      return;
    }
    createError.value = "Could not create environment.";
  } finally {
    isCreating.value = false;
  }
};

const resetSite = async (): Promise<void> => {
  if (!site.value) return;
  isResetting.value = true;
  resetMessage.value = "";
  resetError.value = "";

  try {
    const resp = await api.resetSite(siteId.value);
    resetMessage.value = resp.success ? `Reset accepted (${resp.status}).` : "Reset not accepted.";
    await loadAll();
  } catch (error) {
    if (await handleAuthError(error)) return;
    if (error instanceof ApiClientError) {
      resetError.value = error.message;
      return;
    }
    resetError.value = "Could not reset site.";
  } finally {
    isResetting.value = false;
  }
};

void loadAll();
</script>

<style scoped>
.site-detail {
  display: grid;
  gap: 1rem;
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

.grid {
  display: grid;
  grid-template-columns: minmax(260px, 380px) 1fr;
  gap: 1rem;
  align-items: start;
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

.form {
  display: grid;
  gap: 0.75rem;
  margin-top: 1rem;
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

.inline {
  display: grid;
  gap: 0.5rem;
}

button {
  margin-top: 0.25rem;
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

.env-list {
  margin: 1rem 0 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: 0.65rem;
}

.env-row {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  gap: 0.75rem;
  padding: 0.65rem 0.75rem;
  border-radius: 0.75rem;
  background: rgba(248, 250, 252, 0.8);
  border: 1px solid rgba(226, 232, 240, 0.9);
}

.env-link {
  display: grid;
  grid-template-columns: 1fr;
  gap: 0.25rem;
  text-decoration: none;
  color: inherit;
  min-width: 0;
}

.env-name {
  font-weight: 800;
}

.meta {
  font-size: 0.9rem;
  color: #475569;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
}

.env-slug {
  color: #334155;
  font-size: 0.95rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

.job-panel {
  margin-top: 1rem;
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

.error {
  margin: 0.35rem 0 0;
  font-size: 0.95rem;
  color: #b91c1c;
}

.reset-panel {
  margin-top: 1rem;
  padding: 0.85rem;
  border-radius: 0.85rem;
  border: 1px solid rgba(245, 158, 11, 0.35);
  background: rgba(255, 251, 235, 0.9);
}

.warn {
  margin: 0;
  color: #92400e;
  font-weight: 700;
}

.danger {
  margin-top: 0.65rem;
  border: 0;
  border-radius: 0.75rem;
  padding: 0.7rem 0.95rem;
  font-size: 1rem;
  font-weight: 800;
  color: #ffffff;
  background: #b91c1c;
  cursor: pointer;
}

.danger:disabled {
  opacity: 0.7;
  cursor: wait;
}

.muted {
  margin: 0.35rem 0 0;
  color: #475569;
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

@media (max-width: 900px) {
  .grid {
    grid-template-columns: 1fr;
  }

  .title-row {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
