<template>
  <section class="shell-grid">
      <article class="card">
        <h2>Session</h2>
        <p>You are authenticated and inside the protected app shell.</p>
      </article>

      <article class="card">
        <h2>Sites</h2>
        <p v-if="isLoading" role="status" aria-live="polite">Loading sites...</p>
        <p v-else-if="loadError" class="error" role="alert">{{ loadError }}</p>
        <p v-else>{{ siteCount }} sites available.</p>
      </article>
  </section>
</template>

<script setup lang="ts">
import { ApiClientError } from "~/lib/api/client";

definePageMeta({
  layout: "app",
});

const api = useApiClient();
const auth = useAuthSession();
const isLoading = ref(true);
const siteCount = ref(0);
const loadError = ref("");

const loadSites = async (): Promise<void> => {
  isLoading.value = true;
  loadError.value = "";

  try {
    const sites = await api.listSites();
    siteCount.value = sites.length;
  } catch (error) {
    if (error instanceof ApiClientError && error.status === 401) {
      auth.setGuest();
      await navigateTo("/login", { replace: true });
      return;
    }
    loadError.value = "Could not load sites.";
  } finally {
    isLoading.value = false;
  }
};

void loadSites();
</script>

<style scoped>
.shell-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 1rem;
}

.card {
  border-radius: 0.9rem;
  border: 1px solid #e2e8f0;
  background: #ffffff;
  padding: 1rem;
  box-shadow: 0 20px 36px -34px rgba(15, 23, 42, 0.7);
}

.card h2 {
  margin-top: 0;
}

.error {
  color: #b91c1c;
}
</style>
