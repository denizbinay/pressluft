<template>
  <section class="jobs-page">
    <header class="page-header">
      <h2>Jobs</h2>
      <p class="subtitle">Recent and running jobs with error metadata.</p>
    </header>

    <article class="card" :aria-busy="isLoading">
      <p v-if="isLoading" role="status" aria-live="polite">Loading jobs...</p>
      <p v-else-if="errorMessage" class="error" role="alert">{{ errorMessage }}</p>

      <div v-else>
        <p v-if="sortedJobs.length === 0" class="muted" aria-live="polite">No jobs yet.</p>
        <ul v-else class="job-list">
          <li v-for="job in sortedJobs" :key="job.id" class="job-row">
            <NuxtLink class="job-link" :to="`/app/jobs/${job.id}`">
              <span class="mono job-id">{{ job.id }}</span>
              <span class="job-type">{{ job.job_type || "(unknown)" }}</span>
              <span v-if="job.error_code" class="mono job-error">{{ job.error_code }}</span>
            </NuxtLink>
            <span class="pill" :data-status="job.status">{{ job.status }}</span>
          </li>
        </ul>
      </div>
    </article>
  </section>
</template>

<script setup lang="ts">
import type { JobStatusResponse } from "~/lib/api/types";
import { ApiClientError } from "~/lib/api/client";

definePageMeta({
  layout: "app",
});

const api = useApiClient();
const auth = useAuthSession();

const isLoading = ref(true);
const errorMessage = ref("");
const jobs = ref<JobStatusResponse[]>([]);

const sortedJobs = computed(() => {
  return [...jobs.value].sort((a, b) => {
    const aKey = a.updated_at ?? a.created_at;
    const bKey = b.updated_at ?? b.created_at;
    return bKey.localeCompare(aKey);
  });
});

const loadJobs = async (): Promise<void> => {
  isLoading.value = true;
  errorMessage.value = "";

  try {
    jobs.value = await api.listJobs();
  } catch (error) {
    if (error instanceof ApiClientError && error.status === 401) {
      auth.setGuest();
      await navigateTo("/login", { replace: true });
      return;
    }
    errorMessage.value = "Could not load jobs.";
  } finally {
    isLoading.value = false;
  }
};

void loadJobs();
</script>

<style scoped>
.jobs-page {
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

.card {
  border-radius: 0.9rem;
  border: 1px solid #e2e8f0;
  background: #ffffff;
  padding: 1rem;
  box-shadow: 0 20px 36px -34px rgba(15, 23, 42, 0.7);
}

.job-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: 0.65rem;
}

.job-row {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  gap: 0.75rem;
  padding: 0.65rem 0.75rem;
  border-radius: 0.75rem;
  background: rgba(248, 250, 252, 0.8);
  border: 1px solid rgba(226, 232, 240, 0.9);
}

.job-link {
  display: grid;
  gap: 0.2rem;
  text-decoration: none;
  color: inherit;
  min-width: 0;
}

.job-id {
  font-size: 0.95rem;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.job-type {
  color: #475569;
}

.job-error {
  color: #b91c1c;
  font-weight: 700;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
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

.pill[data-status="succeeded"] {
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

.muted {
  margin: 0;
  color: #475569;
}
</style>
