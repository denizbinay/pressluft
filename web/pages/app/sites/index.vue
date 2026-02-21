<template>
  <section class="sites-page">
    <header class="page-header">
      <h2>Sites</h2>
      <p class="subtitle">Create and inspect sites. Mutations run asynchronously via jobs.</p>
    </header>

    <div class="grid">
      <article class="card">
        <header class="card-header">
          <h3>Create site</h3>
          <p class="hint">Creates a new site and enqueues a provisioning job.</p>
        </header>

        <form class="form" @submit.prevent="submitCreate">
          <label for="site-name">Name</label>
          <input
            id="site-name"
            v-model="createName"
            name="name"
            type="text"
            autocomplete="off"
            required
            :disabled="isCreating"
          />

          <label for="site-slug">Slug</label>
          <input
            id="site-slug"
            v-model="createSlug"
            name="slug"
            type="text"
            autocomplete="off"
            required
            :disabled="isCreating"
            @input="slugTouched = true"
          />

          <p v-if="createError" class="error" role="alert">{{ createError }}</p>

          <button type="submit" :disabled="isCreating">
            {{ isCreating ? "Creating..." : "Create Site" }}
          </button>
        </form>

        <div v-if="createJob" class="job-panel" aria-live="polite">
          <p class="job-line">
            Job <span class="mono">{{ createJob.id }}</span>
            <span class="pill" :data-status="createJob.status">{{ createJob.status }}</span>
          </p>
          <p v-if="createJob.status === 'failed'" class="error" role="alert">
            {{ createJob.error_message || "Site creation failed." }}
          </p>
        </div>
      </article>

      <article class="card" :aria-busy="isLoading">
        <header class="card-header">
          <h3>All sites</h3>
          <p class="hint">List refreshes after a creation job completes.</p>
        </header>

        <p v-if="isLoading" aria-live="polite">Loading sites...</p>
        <p v-else-if="errorMessage" class="error" role="alert">{{ errorMessage }}</p>

        <div v-else>
          <p v-if="sites.length === 0" class="muted" aria-live="polite">No sites yet. Create one to get started.</p>
          <ul v-else class="site-list">
            <li v-for="site in sites" :key="site.id" class="site-row">
              <NuxtLink class="site-link" :to="`/app/sites/${site.id}`">
                <span class="site-name">{{ site.name }}</span>
                <span class="site-slug mono">{{ site.slug }}</span>
              </NuxtLink>
              <span class="pill" :data-status="site.status">{{ site.status }}</span>
            </li>
          </ul>
        </div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { JobStatusResponse, Site } from "~/lib/api/types";
import { ApiClientError } from "~/lib/api/client";

definePageMeta({
  layout: "app",
});

const api = useApiClient();
const auth = useAuthSession();

const isLoading = ref(true);
const errorMessage = ref("");
const sites = ref<Site[]>([]);

const createName = ref("");
const createSlug = ref("");
const slugTouched = ref(false);
const isCreating = ref(false);
const createError = ref("");
const createJob = ref<JobStatusResponse | null>(null);

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

onBeforeUnmount(() => {
  if (pollTimer) {
    clearTimeout(pollTimer);
    pollTimer = null;
  }
});

const loadSites = async (): Promise<void> => {
  isLoading.value = true;
  errorMessage.value = "";

  try {
    sites.value = await api.listSites();
  } catch (error) {
    if (error instanceof ApiClientError && error.status === 401) {
      auth.setGuest();
      await navigateTo("/login", { replace: true });
      return;
    }
    errorMessage.value = "Could not load sites.";
  } finally {
    isLoading.value = false;
  }
};

const pollJob = async (jobId: string): Promise<void> => {
  const step = async (): Promise<void> => {
    try {
      createJob.value = await api.getJob(jobId);
    } catch (error) {
      if (error instanceof ApiClientError && error.status === 401) {
        auth.setGuest();
        await navigateTo("/login", { replace: true });
        return;
      }
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

    await loadSites();
  };

  await step();
};

const submitCreate = async (): Promise<void> => {
  if (pollTimer) {
    clearTimeout(pollTimer);
    pollTimer = null;
  }

  isCreating.value = true;
  createError.value = "";
  createJob.value = null;

  const slug = createSlug.value.trim();
  const name = createName.value.trim();
  if (!name || !slug) {
    createError.value = "Name and slug are required.";
    isCreating.value = false;
    return;
  }

  try {
    const { job_id } = await api.createSite({
      name,
      slug,
    });
    await pollJob(job_id);
  } catch (error) {
    if (error instanceof ApiClientError && error.status === 401) {
      auth.setGuest();
      await navigateTo("/login", { replace: true });
      return;
    }
    if (error instanceof ApiClientError && error.status === 400) {
      createError.value = error.message;
      return;
    }
    createError.value = "Could not create site.";
  } finally {
    isCreating.value = false;
  }
};

void loadSites();
</script>

<style scoped>
.sites-page {
  display: grid;
  gap: 1rem;
}

.page-header h2 {
  margin: 0;
}

.subtitle {
  margin: 0.25rem 0 0;
  color: #475569;
}

.grid {
  display: grid;
  grid-template-columns: minmax(260px, 340px) 1fr;
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

input {
  width: 100%;
  border: 1px solid #cbd5e1;
  border-radius: 0.625rem;
  padding: 0.65rem 0.75rem;
  font-size: 1rem;
}

input:focus-visible {
  outline: 2px solid #0f766e;
  outline-offset: 1px;
  border-color: #0f766e;
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

.site-list {
  margin: 1rem 0 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: 0.65rem;
}

.site-row {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  gap: 0.75rem;
  padding: 0.65rem 0.75rem;
  border-radius: 0.75rem;
  background: rgba(248, 250, 252, 0.8);
  border: 1px solid rgba(226, 232, 240, 0.9);
}

.site-link {
  display: grid;
  gap: 0.25rem;
  text-decoration: none;
  color: inherit;
  min-width: 0;
}

.site-name {
  font-weight: 800;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
}

.site-slug {
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

.muted {
  margin: 1rem 0 0;
  color: #475569;
}

@media (max-width: 900px) {
  .grid {
    grid-template-columns: 1fr;
  }
}
</style>
