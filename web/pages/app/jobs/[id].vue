<template>
  <section class="job-detail">
    <header class="page-header">
      <div class="title-row">
        <div>
          <p class="eyebrow">Job</p>
          <h2 v-if="job" class="mono">{{ job.id }}</h2>
          <h2 v-else>Job</h2>
        </div>

        <NuxtLink class="back" to="/app/jobs">Back to Jobs</NuxtLink>
      </div>

      <p v-if="job" class="subtitle">
        <span class="pill" :data-status="job.status">{{ job.status }}</span>
        <span class="mono">{{ job.job_type || "(unknown)" }}</span>
      </p>
    </header>

    <article class="card" :aria-busy="isLoading">
      <p v-if="isLoading" role="status" aria-live="polite">Loading job...</p>
      <p v-else-if="errorMessage" class="error" role="alert">{{ errorMessage }}</p>

      <div v-else-if="job" class="content">
        <div class="actions">
          <button
            type="button"
            class="danger"
            data-testid="cancel-job"
            :disabled="isSubmitting || !isCancellable"
            @click="cancel"
          >
            {{ isSubmitting ? "Cancelling..." : "Cancel Job" }}
          </button>
          <p v-if="!isCancellable" class="muted" aria-live="polite">Only queued/running jobs can be cancelled.</p>
          <p v-if="actionMessage" class="muted" aria-live="polite">{{ actionMessage }}</p>
          <p v-if="actionError" class="error" role="alert">{{ actionError }}</p>
        </div>

        <dl class="details">
          <div class="row">
            <dt>Status</dt>
            <dd>{{ job.status }}</dd>
          </div>
          <div class="row">
            <dt>Attempt</dt>
            <dd class="mono">{{ job.attempt_count }} / {{ job.max_attempts }}</dd>
          </div>
          <div class="row" v-if="job.site_id">
            <dt>Site</dt>
            <dd class="mono">{{ job.site_id }}</dd>
          </div>
          <div class="row" v-if="job.environment_id">
            <dt>Environment</dt>
            <dd class="mono">{{ job.environment_id }}</dd>
          </div>
          <div class="row" v-if="job.node_id">
            <dt>Node</dt>
            <dd class="mono">{{ job.node_id }}</dd>
          </div>
          <div class="row" v-if="job.error_code">
            <dt>Error code</dt>
            <dd class="mono">{{ job.error_code }}</dd>
          </div>
          <div class="row" v-if="job.error_message">
            <dt>Error message</dt>
            <dd>{{ job.error_message }}</dd>
          </div>
        </dl>
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

const route = useRoute();
const api = useApiClient();
const auth = useAuthSession();

const jobId = computed(() => String(route.params.id ?? ""));

const isLoading = ref(true);
const errorMessage = ref("");
const job = ref<JobStatusResponse | null>(null);
const isSubmitting = ref(false);
const actionMessage = ref("");
const actionError = ref("");

const isCancellable = computed(() => job.value?.status === "queued" || job.value?.status === "running");

const loadJob = async (): Promise<void> => {
  isLoading.value = true;
  errorMessage.value = "";

  try {
    job.value = await api.getJob(jobId.value);
  } catch (error) {
    if (error instanceof ApiClientError && error.status === 401) {
      auth.setGuest();
      await navigateTo("/login", { replace: true });
      return;
    }
    if (error instanceof ApiClientError && error.status === 404) {
      errorMessage.value = "Job not found.";
      return;
    }
    errorMessage.value = "Could not load job.";
  } finally {
    isLoading.value = false;
  }
};

const cancel = async (): Promise<void> => {
  if (!isCancellable.value) return;

  isSubmitting.value = true;
  actionError.value = "";
  actionMessage.value = "";
  try {
    const resp = await api.cancelJob(jobId.value);
    actionMessage.value = resp.success ? `Cancel accepted (${resp.status}).` : "Cancel not accepted.";
    await loadJob();
  } catch (error) {
    if (error instanceof ApiClientError && error.status === 401) {
      auth.setGuest();
      await navigateTo("/login", { replace: true });
      return;
    }
    if (error instanceof ApiClientError) {
      actionError.value = error.message;
      return;
    }
    actionError.value = "Could not cancel job.";
  } finally {
    isSubmitting.value = false;
  }
};

void loadJob();
</script>

<style scoped>
.job-detail {
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

.card {
  border-radius: 0.9rem;
  border: 1px solid #e2e8f0;
  background: #ffffff;
  padding: 1rem;
  box-shadow: 0 20px 36px -34px rgba(15, 23, 42, 0.7);
}

.content {
  display: grid;
  gap: 1rem;
}

.actions {
  display: grid;
  gap: 0.5rem;
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

.danger {
  border: 0;
  border-radius: 0.75rem;
  padding: 0.7rem 0.95rem;
  font-size: 1rem;
  font-weight: 800;
  color: #ffffff;
  background: #b91c1c;
  cursor: pointer;
  width: fit-content;
}

.danger:disabled {
  opacity: 0.7;
  cursor: wait;
}

.muted {
  margin: 0;
  color: #475569;
}

.error {
  margin: 0;
  color: #b91c1c;
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
  .row {
    grid-template-columns: 1fr;
  }

  .title-row {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
